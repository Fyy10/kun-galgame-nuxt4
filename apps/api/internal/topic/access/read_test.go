package access

import (
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
)

func TestAllowed(t *testing.T) {
	grants := []model.TopicAccessGrant{
		{SubjectType: "role", SubjectValue: "creator"},
		{SubjectType: "user", SubjectValue: "4"},
	}
	viewers := []struct {
		name    string
		viewer  View
		allowed [4]bool
	}{
		{"anonymous", View{}, [4]bool{true, false, false, false}},
		{"unrelated login", View{ID: 2, Authenticated: true, Roles: []string{"user"}}, [4]bool{true, true, false, false}},
		{"second role granted", View{ID: 3, Authenticated: true, Roles: []string{"user", "creator"}}, [4]bool{true, true, true, false}},
		{"user granted", View{ID: 4, Authenticated: true}, [4]bool{true, true, false, true}},
		{"author", View{ID: 1, Authenticated: true}, [4]bool{true, true, true, true}},
		{"restricted permission", View{ID: 5, Authenticated: true, ViewRestricted: true}, [4]bool{true, true, true, true}},
		{"hidden permission", View{ID: 6, Authenticated: true, ViewHidden: true}, [4]bool{true, true, false, false}},
		{"both permissions", View{ID: 7, Authenticated: true, ViewHidden: true, ViewRestricted: true}, [4]bool{true, true, true, true}},
		{"hidden role granted", View{ID: 8, Authenticated: true, ViewHidden: true, Roles: []string{"creator"}}, [4]bool{true, true, true, false}},
		{"hidden user granted", View{ID: 4, Authenticated: true, ViewHidden: true}, [4]bool{true, true, false, true}},
	}
	for i, scope := range []string{"public", "login", "role", "users"} {
		for _, tt := range viewers {
			for _, hidden := range []bool{false, true} {
				name := scope + "/" + tt.name
				if hidden {
					name += "/hidden"
				}
				t.Run(name, func(t *testing.T) {
					topic := &model.Topic{UserID: 1, AccessScope: scope}
					want := tt.allowed[i]
					if hidden {
						topic.Status = 1
						want = want && (tt.viewer.ID == 1 || tt.viewer.ViewHidden)
					}
					if got := Allowed(topic, tt.viewer, grants); got != want {
						t.Fatalf("allowed = %v, want %v", got, want)
					}
				})
			}
		}
	}
}

func TestAllowedGrantTypesAndDecimalIDs(t *testing.T) {
	viewer := View{ID: 4, Authenticated: true, Roles: []string{"creator"}}
	for _, tt := range []struct{ scope, kind, value string }{
		{"users", "user", "5"}, {"users", "user", "04"}, {"users", "role", "4"},
		{"role", "user", "creator"}, {"role", "role", "admin"}, {"unknown", "user", "4"},
	} {
		t.Run(tt.scope+"/"+tt.kind+"/"+tt.value, func(t *testing.T) {
			if Allowed(&model.Topic{UserID: 1, AccessScope: tt.scope}, viewer, []model.TopicAccessGrant{{SubjectType: tt.kind, SubjectValue: tt.value}}) {
				t.Fatal("unexpected grant match")
			}
		})
	}
}

func TestNeedsGrants(t *testing.T) {
	for _, tt := range []struct {
		scope string
		want  bool
	}{
		{"public", false}, {"login", false}, {"role", true}, {"users", true}, {"", false},
	} {
		if got := NeedsGrants(&model.Topic{AccessScope: tt.scope}); got != tt.want {
			t.Errorf("%s: %v, want %v", tt.scope, got, tt.want)
		}
	}
	if NeedsGrants(nil) {
		t.Fatal("nil topic")
	}
}

func TestCanReadDerivesStaffFromUserCan(t *testing.T) {
	topic := &model.Topic{UserID: 1, Status: 1, AccessScope: "public"}
	staff := &middleware.UserInfo{ID: 9, Roles: []string{"moderator"}}
	if !CanRead(topic, staff, nil) {
		t.Fatal("cookie moderator must read a hidden public topic")
	}
	stranger := &middleware.UserInfo{ID: 9, Roles: []string{"user"}}
	if CanRead(topic, stranger, nil) {
		t.Fatal("cookie user without view_hidden must not read a hidden topic")
	}
	if CanRead(topic, nil, nil) {
		t.Fatal("anonymous must not read a hidden topic")
	}
}

func TestSnapshotAnonymous(t *testing.T) {
	v := Snapshot(nil)
	if v.Authenticated || v.ViewHidden || v.ViewRestricted || v.ID != 0 {
		t.Fatalf("%+v", v)
	}
}
