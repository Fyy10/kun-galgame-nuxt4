package service

import (
	"context"
	"testing"

	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"
)

func TestVisibleTo(t *testing.T) {
	const author int64 = 7
	cases := []struct {
		name   string
		status int32
		viewer int
		want   bool
	}{
		{"visible seen by anyone", communityclient.PostVisible, 0, true},
		{"deleted seen by anyone", communityclient.PostDeleted, 0, true},
		{"held seen by its author", communityclient.PostHeld, 7, true},
		{"held hidden from others", communityclient.PostHeld, 8, false},
		{"held hidden from anon", communityclient.PostHeld, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := visibleTo(tc.status, author, tc.viewer); got != tc.want {
				t.Errorf("visibleTo(%d, %d, %d) = %v, want %v", tc.status, author, tc.viewer, got, tc.want)
			}
		})
	}
}

func sampleAuthor() userclient.User {
	return userclient.User{ID: 7, Name: "kun", Avatar: "a.webp", Status: 0}
}

func TestBuildCommunityItemDeleted(t *testing.T) {
	p := communityclient.PostView{ID: 100, AuthorID: 7, ContentRaw: "secret", Status: communityclient.PostDeleted, CreatedAt: "2026-07-13T00:00:00Z"}
	item := buildCommunityItem(p, 42, sampleAuthor(), 0, false)
	if item.Content != "" {
		t.Errorf("deleted content = %q, want blank", item.Content)
	}
	if !item.Deleted || item.Held {
		t.Errorf("deleted=%v held=%v, want deleted=true held=false", item.Deleted, item.Held)
	}
	if item.GalgameID != 42 || item.ID != 100 {
		t.Errorf("item = %+v", item)
	}
}

func TestBuildCommunityItemHeld(t *testing.T) {
	p := communityclient.PostView{ID: 101, AuthorID: 7, ContentRaw: "hi", Status: communityclient.PostHeld, ReplyToPostID: 0, RootPostID: 0}
	item := buildCommunityItem(p, 42, sampleAuthor(), 3, true)
	if item.Content != "hi" || !item.Held || item.Deleted {
		t.Errorf("held item = %+v", item)
	}
	if item.ParentCommentID != nil || item.RootCommentID != nil {
		t.Errorf("top-level pointers should be nil, got parent=%v root=%v", item.ParentCommentID, item.RootCommentID)
	}
	if item.LikeCount != 3 || item.User.ID != 7 {
		t.Errorf("item = %+v", item)
	}
}

func TestBuildCommunityItemPointers(t *testing.T) {
	p := communityclient.PostView{ID: 102, AuthorID: 7, ContentRaw: "re", ReplyToPostID: 50, RootPostID: 40, EditedAt: "2026-07-13T01:00:00Z", EditedByModerator: true}
	item := buildCommunityItem(p, 42, sampleAuthor(), 0, false)
	if item.ParentCommentID == nil || *item.ParentCommentID != 50 {
		t.Errorf("parent = %v, want 50", item.ParentCommentID)
	}
	if item.RootCommentID == nil || *item.RootCommentID != 40 {
		t.Errorf("root = %v, want 40", item.RootCommentID)
	}
	if item.Edited == nil || *item.Edited != "2026-07-13T01:00:00Z" || !item.EditedByModerator {
		t.Errorf("edited = %v, edited_by_moderator = %v", item.Edited, item.EditedByModerator)
	}
}

func TestLikeEffects(t *testing.T) {
	cases := []struct {
		name      string
		added     bool
		authorID  int64
		userID    int
		wantLocal bool
		wantDelta int
	}{
		{"add, other", true, 7, 9, true, 1},
		{"add, self", true, 9, 9, true, 0},
		{"remove, other", false, 7, 9, false, -1},
		{"remove, self", false, 9, 9, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			local, delta := likeEffects(tc.added, tc.authorID, tc.userID)
			if local != tc.wantLocal || delta != tc.wantDelta {
				t.Errorf("likeEffects(%v,%d,%d) = (%v,%d), want (%v,%d)",
					tc.added, tc.authorID, tc.userID, local, delta, tc.wantLocal, tc.wantDelta)
			}
		})
	}
}

func TestPrepareMentionIDs(t *testing.T) {
	known := map[int]userclient.User{2: {ID: 2}, 3: {ID: 3}}
	t.Run("self dropped", func(t *testing.T) {
		got, err := prepareMentionIDs([]int{1, 2}, 1, known, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 1 || got[0] != 2 {
			t.Errorf("got %v, want [2]", got)
		}
	})
	t.Run("duplicates dropped", func(t *testing.T) {
		got, err := prepareMentionIDs([]int{2, 2, 3, 2}, 1, known, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 2 || got[0] != 2 || got[1] != 3 {
			t.Errorf("got %v, want [2 3]", got)
		}
	})
	t.Run("lookup error keeps ids", func(t *testing.T) {
		got, err := prepareMentionIDs([]int{2, 99}, 1, known, context.Canceled)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 2 || got[0] != 2 || got[1] != 99 {
			t.Errorf("got %v, want [2 99]", got)
		}
	})
	t.Run("unknown ids dropped on success", func(t *testing.T) {
		got, err := prepareMentionIDs([]int{2, 99}, 1, known, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 1 || got[0] != 2 {
			t.Errorf("got %v, want [2]", got)
		}
	})
	t.Run("more than 20 rejected", func(t *testing.T) {
		ids := make([]int, 21)
		knownAll := map[int]userclient.User{}
		for i := range ids {
			ids[i] = i + 1
			knownAll[i+1] = userclient.User{ID: i + 1}
		}
		_, err := prepareMentionIDs(ids, 0, knownAll, nil)
		if err == nil {
			t.Fatal("want validation error")
		}
	})
}

func TestNewlyAddedMentionIDs(t *testing.T) {
	old := "[@a](kungal-user:2) hi"
	newBody := "[@a](kungal-user:2) [@b](kungal-user:3) [@me](kungal-user:9)"
	got := newlyAddedMentionIDs(old, newBody, 9)
	if len(got) != 1 || got[0] != 3 {
		t.Errorf("got %v, want [3]", got)
	}
	if got := newlyAddedMentionIDs("", newBody, 9); len(got) != 2 {
		t.Errorf("empty old = %v, want two new ids", got)
	}
}

func TestClampReadLimit(t *testing.T) {
	cases := map[int]string{0: "50", -1: "50", 1: "1", 30: "30", 50: "50", 51: "50", 999: "50"}
	for in, want := range cases {
		if got := clampReadLimit(in); got != want {
			t.Errorf("clampReadLimit(%d) = %q, want %q", in, got, want)
		}
	}
}
