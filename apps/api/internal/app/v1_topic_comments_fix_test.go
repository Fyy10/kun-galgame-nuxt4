package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/trustclient"
)

const (
	w5CommentAlice  = 930000401
	w5CommentBob    = 930000402
	w5CommentOnR2   = 930000403
	w5CommentTieA   = 930000404
	w5CommentTieB   = 930000405
	w5CommentHidden = 930000406
	w5CommentRole   = 930000407

	w5ReplyRole = 930000311

	w5ImageHash = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type countingChecker struct {
	calls  atomic.Int32
	deny   atomic.Bool
	lastID atomic.Int64
}

func (c *countingChecker) Check(_ context.Context, req trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	c.calls.Add(1)
	if req.AuthorID != nil {
		c.lastID.Store(*req.AuthorID)
	}
	if c.deny.Load() {
		return &trustclient.CheckResult{Decision: gate.DecisionDeny, Matched: []string{"x"}}, nil
	}
	return &trustclient.CheckResult{Decision: gate.DecisionAllow}, nil
}

// seedComments adds the comment fixtures on top of newWriteFix's topics and
// replies. writeFix.cleanup already removes every topic_comment of these users.
func (f *writeFix) seedComments(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed comments: %v\n%s", err, q)
		}
	}
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'role-r1', 1, ?, ?, 0, 0, ?, ?)`, w5ReplyRole, w3UserAlice, w3TopicRole, base, base)

	ins := func(id, replyID, topicID, user, target int, parent *int, status int, body string, at time.Time) {
		run(`INSERT INTO topic_comment
			(id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, body, topicID, replyID, user, target, parent, status, at, at)
	}
	// reply w3ReplyMin is bob's, reply w3ReplyMin+1 is alice's.
	ins(w5CommentAlice, w3ReplyMin, w3TopicFloors, w3UserAlice, w3UserBob, nil, 0, "alice-comment", base)
	ins(w5CommentBob, w3ReplyMin, w3TopicFloors, w3UserBob, w3UserAlice, nil, 0, "bob-comment", base.Add(time.Second))
	ins(w5CommentOnR2, w3ReplyMin+1, w3TopicFloors, w3UserBob, w3UserAlice, nil, 0, "on-r2", base.Add(2*time.Second))
	// Same instant, ids out of insertion order: the id tie-breaker is the only
	// thing that makes the comment order stable.
	ins(w5CommentTieB, w3ReplyMin+2, w3TopicFloors, w3UserBob, w3UserBob, nil, 0, "tie-b", base.Add(3*time.Second))
	ins(w5CommentTieA, w3ReplyMin+2, w3TopicFloors, w3UserAlice, w3UserBob, nil, 0, "tie-a", base.Add(3*time.Second))
	ins(w5CommentHidden, w3ReplyMin, w3TopicFloors, w3UserAlice, w3UserBob, nil, 1, "hidden-comment", base.Add(4*time.Second))
	ins(w5CommentRole, w5ReplyRole, w3TopicRole, w3UserAlice, w3UserAlice, nil, 0, "role-comment", base.Add(5*time.Second))
}

func newCommentFix(t *testing.T, checker gate.Checker) *writeFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.seedComments(t)
	return f
}

func (f *writeFix) postComment(t *testing.T, replyID int, session, idem string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/replies/%d/comments", replyID), session,
		"/replies/{reply_id}/comments", idem, nil, payload)
	return resp, problemMap(t, raw)
}

func (f *writeFix) commentCall(t *testing.T, method, suffix string, commentID int, session string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	path := "/api/v1/comments/" + strconv.Itoa(commentID) + suffix
	resp, raw := f.doJSON(t, method, path, session, "/comments/{comment_id}"+suffix, "", nil, payload)
	if len(raw) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, raw)
}

func (f *writeFix) getComment(t *testing.T, commentID int, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.commentCall(t, http.MethodGet, "", commentID, session, nil)
}

func (f *writeFix) commentRowCount(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Row().Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// docPlainText concatenates every text node value of a ContentDocument, with a
// newline for each break node, so a test can assert the visible text without
// spelling out the node tree.
func docPlainText(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var doc struct {
		Children []struct {
			Object   string `json:"object"`
			Children []struct {
				Object string `json:"object"`
				Value  string `json:"value"`
			} `json:"children"`
		} `json:"children"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("content document: %v\n%s", err, raw)
	}
	out := ""
	for _, block := range doc.Children {
		for _, inline := range block.Children {
			switch inline.Object {
			case "text":
				out += inline.Value
			case "break":
				out += "\n"
			}
		}
	}
	return out
}

func commentContent(t *testing.T, comment map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(comment["content"])
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func commentNodes(t *testing.T, comment map[string]any) []map[string]any {
	t.Helper()
	doc, _ := comment["content"].(map[string]any)
	blocks, _ := doc["children"].([]any)
	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) != 1 {
		t.Fatalf("a comment holds at most one paragraph, got %d blocks: %v", len(blocks), blocks)
	}
	block, _ := blocks[0].(map[string]any)
	if block["object"] != "paragraph" {
		t.Fatalf("comment block object %v, want paragraph", block["object"])
	}
	inlines, _ := block["children"].([]any)
	out := make([]map[string]any, 0, len(inlines))
	for _, n := range inlines {
		node, _ := n.(map[string]any)
		out = append(out, node)
	}
	return out
}

func viewerOf(t *testing.T, comment map[string]any) map[string]any {
	t.Helper()
	v, ok := comment["viewer"].(map[string]any)
	if !ok {
		t.Fatalf("viewer %v", comment["viewer"])
	}
	return v
}

func awardsFor(calls []awardCall, ref string) []awardCall {
	var out []awardCall
	for _, c := range calls {
		if c.ref == ref {
			out = append(out, c)
		}
	}
	return out
}
