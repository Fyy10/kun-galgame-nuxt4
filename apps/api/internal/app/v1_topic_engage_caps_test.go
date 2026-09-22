package app

import (
	"encoding/json"
	"net/http"
	"testing"
)

type topicViewerJSON struct {
	HasLiked         bool `json:"has_liked"`
	HasDisliked      bool `json:"has_disliked"`
	HasFavorited     bool `json:"has_favorited"`
	HasUpvoted       bool `json:"has_upvoted"`
	CanEdit          bool `json:"can_edit"`
	CanHide          bool `json:"can_hide"`
	CanUnhide        bool `json:"can_unhide"`
	CanLike          bool `json:"can_like"`
	CanUpvote        bool `json:"can_upvote"`
	CanSetBestAnswer bool `json:"can_set_best_answer"`
	CanPinReply      bool `json:"can_pin_reply"`
}

type replyViewerJSON struct {
	HasLiked    bool `json:"has_liked"`
	HasDisliked bool `json:"has_disliked"`
	CanEdit     bool `json:"can_edit"`
	CanDelete   bool `json:"can_delete"`
	CanLike     bool `json:"can_like"`
}

func TestV1TopicViewerCapsAuthor(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if !v.CanEdit || !v.CanHide || v.CanUnhide || v.CanLike || v.CanUpvote || !v.CanSetBestAnswer || !v.CanPinReply {
		t.Fatalf("author published: %+v", v)
	}
}

func TestV1TopicViewerCapsOther(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asBob(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if v.CanEdit || v.CanHide || !v.CanLike || !v.CanUpvote || v.CanSetBestAnswer || v.CanPinReply {
		t.Fatalf("other published: %+v", v)
	}
}

func TestV1TopicViewerCapsCookieModerator(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asStaff(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if !v.CanEdit || !v.CanHide || !v.CanLike || !v.CanSetBestAnswer || !v.CanPinReply {
		t.Fatalf("cookie moderator: %+v", v)
	}
}

func TestV1TopicViewerCapsBearerModeratorNoStaff(t *testing.T) {
	f := newEngageFix(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), "", "/topics/{topic_id}", nil, authBearer("staff-token"))
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if v.CanHide || v.CanEdit || v.CanSetBestAnswer || v.CanPinReply || !v.CanLike || !v.CanUpvote {
		t.Fatalf("bearer moderator must not hold staff flags: %+v", v)
	}
}

func TestV1TopicViewerCapsHiddenAuthorAndModerator(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicHiddenAuthor), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if !v.CanUnhide || v.CanHide || v.CanLike {
		t.Fatalf("author on own hide: %+v", v)
	}
	staff := f.asStaff(t)
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicHiddenAuthor), staff, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v = topicViewerOf(t, body)
	if !v.CanUnhide || v.CanHide {
		t.Fatalf("cookie moderator on author hide: %+v", v)
	}
}

func TestV1TopicViewerCapsModeratorHiddenAuthorCannotUnhide(t *testing.T) {
	f := newEngageFix(t)
	sess := f.asAlice(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicHiddenMod), sess, "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	v := topicViewerOf(t, body)
	if v.CanUnhide {
		t.Fatalf("author on moderator hide can_unhide: %+v", v)
	}
}

func TestV1TopicViewerAnonymousNull(t *testing.T) {
	f := newEngageFix(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic), "", "/topics/{topic_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["viewer"] != nil {
		t.Fatalf("anonymous viewer %v", obj["viewer"])
	}
}

func TestV1ReplyViewerCapsGetAndPage(t *testing.T) {
	f := newEngageFix(t)
	alice := f.asAlice(t)
	bob := f.asBob(t)
	dave := f.asDave(t)

	assertReply := func(session string, want replyViewerJSON, viaPage bool) {
		t.Helper()
		if viaPage {
			resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic)+"/replies?limit=20", session, "/topics/{topic_id}/replies", nil, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("%d %s", resp.StatusCode, body)
			}
			var page struct {
				Items []struct {
					ID     string           `json:"id"`
					Viewer *replyViewerJSON `json:"viewer"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &page); err != nil {
				t.Fatal(err)
			}
			for _, it := range page.Items {
				if it.ID == strID(e1ReplyDave) {
					if it.Viewer == nil {
						t.Fatal("viewer nil")
					}
					if *it.Viewer != want {
						t.Fatalf("page viewer %+v want %+v", *it.Viewer, want)
					}
					return
				}
			}
			t.Fatal("dave reply missing from page")
		}
		resp, body := f.do(t, http.MethodGet, "/api/v1/replies/"+strID(e1ReplyDave), session, "/replies/{reply_id}", nil, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, body)
		}
		got := replyViewerOf(t, body)
		if got != want {
			t.Fatalf("getReply viewer %+v want %+v", got, want)
		}
	}

	assertReply(dave, replyViewerJSON{CanEdit: true, CanDelete: true}, false)
	assertReply(dave, replyViewerJSON{CanEdit: true, CanDelete: true}, true)
	assertReply(bob, replyViewerJSON{CanLike: true}, false)
	assertReply(bob, replyViewerJSON{CanLike: true}, true)
	assertReply(alice, replyViewerJSON{CanLike: true}, false)
	staff := f.asStaff(t)
	assertReply(staff, replyViewerJSON{CanEdit: true, CanDelete: true, CanLike: true}, false)
	resp, body := f.do(t, http.MethodGet, "/api/v1/replies/"+strID(e1ReplyDave), "", "/replies/{reply_id}", nil, authBearer("staff-token"))
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	got := replyViewerOf(t, body)
	if got.CanEdit || got.CanDelete || !got.CanLike {
		t.Fatalf("bearer moderator reply caps %+v", got)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/replies/"+strID(e1ReplyDave), "", "/replies/{reply_id}", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	obj := jsonObj(t, body)
	if obj["viewer"] != nil {
		t.Fatalf("anonymous reply viewer %v", obj["viewer"])
	}
}

func topicViewerOf(t *testing.T, body []byte) topicViewerJSON {
	t.Helper()
	var wrap struct {
		Viewer *topicViewerJSON `json:"viewer"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Viewer == nil {
		t.Fatal("viewer is null")
	}
	return *wrap.Viewer
}

func replyViewerOf(t *testing.T, body []byte) replyViewerJSON {
	t.Helper()
	var wrap struct {
		Viewer *replyViewerJSON `json:"viewer"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Viewer == nil {
		t.Fatal("viewer is null")
	}
	return *wrap.Viewer
}
