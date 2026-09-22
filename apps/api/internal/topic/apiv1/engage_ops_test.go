package apiv1

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
)

func TestTopicViewerMatchesCaps(t *testing.T) {
	const author, other, staff = 1, 2, 3
	published := &model.Topic{ID: 10, UserID: author}
	hiddenByMod := &model.Topic{ID: 12, UserID: author, Status: 1, HiddenBy: "moderator"}
	authorUser := &middleware.UserInfo{ID: author, Roles: []string{"user"}}
	otherUser := &middleware.UserInfo{ID: other, Roles: []string{"user"}}
	staffUser := &middleware.UserInfo{ID: staff, Roles: []string{"moderator"}}
	mine := map[string]struct{}{"like": {}}

	if topicViewer(published, nil, mine, true, true) != nil {
		t.Fatal("anonymous viewer must be nil")
	}
	got := topicViewer(published, authorUser, mine, true, false)
	if got.CanLike || got.CanUpvote || !got.CanEdit || !got.CanHide || !got.CanSetBestAnswer || !got.CanPinReply {
		t.Fatalf("author published: %+v", got)
	}
	if !got.HasLiked || got.HasDisliked || !got.HasFavorited || got.HasUpvoted {
		t.Fatalf("author has_*: %+v", got)
	}
	got = topicViewer(published, otherUser, nil, false, true)
	if !got.CanLike || !got.CanUpvote || got.CanEdit || got.CanHide {
		t.Fatalf("other published: %+v", got)
	}
	got = topicViewer(hiddenByMod, authorUser, nil, false, false)
	if got.CanUnhide || got.CanHide || got.CanLike {
		t.Fatalf("author on moderator hide: %+v", got)
	}
	got = topicViewer(published, staffUser, nil, false, false)
	if !got.CanHide || !got.CanSetBestAnswer || !got.CanPinReply || !got.CanLike {
		t.Fatalf("staff published: %+v", got)
	}
}

func TestReplyViewerMatchesCaps(t *testing.T) {
	const author, other = 1, 2
	published := &model.Topic{ID: 10, UserID: other}
	reply := &model.TopicReply{ID: 20, TopicID: 10, UserID: author}
	authorUser := &middleware.UserInfo{ID: author, Roles: []string{"user"}}
	otherUser := &middleware.UserInfo{ID: other, Roles: []string{"user"}}
	if replyViewer(published, reply, nil, nil) != nil {
		t.Fatal("anonymous reply viewer must be nil")
	}
	got := replyViewer(published, reply, authorUser, map[string]struct{}{"dislike": {}})
	if got.CanLike || !got.CanEdit || !got.CanDelete || got.HasLiked || !got.HasDisliked {
		t.Fatalf("reply author: %+v", got)
	}
	got = replyViewer(published, reply, otherUser, nil)
	if !got.CanLike || got.CanEdit || got.CanDelete {
		t.Fatalf("other on reply: %+v", got)
	}
}

func TestTrimUpvoteNote(t *testing.T) {
	hi := "  hi  "
	if got := trimUpvoteNote(&hi); got != "hi" {
		t.Fatalf("trim %q", got)
	}
	spaces := "   "
	if got := trimUpvoteNote(&spaces); got != "" {
		t.Fatalf("whitespace-only %q", got)
	}
	if got := trimUpvoteNote(nil); got != "" {
		t.Fatalf("nil %q", got)
	}
	if notePtr("") != nil {
		t.Fatal("empty note must be nil")
	}
	if notePtr("x") == nil || *notePtr("x") != "x" {
		t.Fatal("non-empty note")
	}
}

func TestAfterCommitSkipsAwardsOnError(t *testing.T) {
	var got []pendingAward
	x := &Interactions{award: func(userID, delta int, reason, ref, key string) {
		got = append(got, pendingAward{userID, delta, reason, ref, key})
	}}
	jobs := []pendingAward{{userID: 1, delta: 1, reason: "liked", ref: "topic:1", key: "k"}}
	if err := x.afterCommit(errors.New("rolled back"), jobs); err == nil {
		t.Fatal("expected error")
	}
	if len(got) != 0 {
		t.Fatalf("awards sent before commit: %+v", got)
	}
	if err := x.afterCommit(nil, jobs); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].key != "k" {
		t.Fatalf("awards after commit: %+v", got)
	}
}

func TestHistoryCursorBoundToTargetAndList(t *testing.T) {
	fp := historyFingerprint("topic_upvotes", 7)
	cur := collect.EncodeCursor(historySort, fp, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), "12")
	if _, p := decodeHistoryCursor(cur, fp); p != nil {
		t.Fatalf("same list: %v", p)
	}
	if _, p := decodeHistoryCursor(cur, historyFingerprint("topic_reactions", 7)); p == nil {
		t.Fatal("cursor from another list must be INVALID_CURSOR")
	}
	if _, p := decodeHistoryCursor(cur, historyFingerprint("topic_upvotes", 8)); p == nil {
		t.Fatal("cursor from another topic must be INVALID_CURSOR")
	}
	if _, p := decodeHistoryCursor(cur, fp); p != nil {
		t.Fatal("round trip")
	}
	if got := historyLimit(0); got != collect.DefaultLimit {
		t.Fatalf("limit 0 -> %d", got)
	}
	if strconv.Itoa(12) != "12" {
		t.Fatal("sanity")
	}
}
