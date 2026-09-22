package app

import (
	"net/http"
	"strconv"
	"testing"
)

func TestV1GetComment(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.getComment(t, w5CommentAlice, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous get %d %+v", resp.StatusCode, out)
	}
	if out["object"] != "comment" || out["id"] != strconv.Itoa(w5CommentAlice) {
		t.Fatalf("%+v", out)
	}
	if out["viewer"] != nil {
		t.Fatalf("anonymous viewer %v", out["viewer"])
	}
	if docPlainText(t, commentContent(t, out)) != "alice-comment" {
		t.Fatalf("content %v", out["content"])
	}
	// in_reply_to_user is the stored column, not a derivation: the retired
	// "comment at someone" picker wrote third parties into it.
	target, _ := out["in_reply_to_user"].(map[string]any)
	if target["id"] != strconv.Itoa(w3UserBob) {
		t.Fatalf("in_reply_to_user %v", out["in_reply_to_user"])
	}

	resp, out = f.getComment(t, w5CommentAlice, "sess-bob")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, out)
	}
	v := viewerOf(t, out)
	if v["can_edit"] != false || v["can_delete"] != false || v["can_like"] != true || v["has_liked"] != false {
		t.Fatalf("bob on alice's comment: %v", v)
	}

	resp, out = f.getComment(t, w5CommentAlice, "sess-staff")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff get %d %+v", resp.StatusCode, out)
	}
	v = viewerOf(t, out)
	if v["can_edit"] != true || v["can_delete"] != true {
		t.Fatalf("staff caps %v", v)
	}
}

func TestV1GetCommentNotFound(t *testing.T) {
	f := newCommentFix(t, nil)

	for name, id := range map[string]int{
		"hidden by trust": w5CommentHidden,
		"missing":         930000499,
	} {
		resp, out := f.getComment(t, id, "sess-alice")
		if resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
			t.Fatalf("%s: %d %+v", name, resp.StatusCode, out)
		}
	}
	// The topic grants the creator role, which other does not hold.
	resp, out := f.getComment(t, w5CommentRole, "sess-other")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("role-scoped topic %d %+v", resp.StatusCode, out)
	}
	resp, out = f.getComment(t, w5CommentRole, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("its author %d %+v", resp.StatusCode, out)
	}
}

// K20: a comment is plain text. Markdown characters stay literal, newlines
// survive as break nodes, and an image token becomes an image node.
func TestV1CommentContentIsNotMarkdown(t *testing.T) {
	f := newCommentFix(t, nil)

	text := "*not emphasis* **not strong**\n# not a heading /image/" + w5ImageHash + " end"
	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(80), map[string]any{"text": text})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	nodes := commentNodes(t, out)
	kinds := make([]string, len(nodes))
	for i, n := range nodes {
		kinds[i], _ = n["object"].(string)
	}
	want := []string{"text", "break", "text", "image", "text"}
	if len(kinds) != len(want) {
		t.Fatalf("nodes %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("nodes %v, want %v", kinds, want)
		}
	}
	if nodes[0]["value"] != "*not emphasis* **not strong**" {
		t.Fatalf("markup was eaten: %q", nodes[0]["value"])
	}
	if nodes[2]["value"] != "# not a heading " {
		t.Fatalf("leading # was eaten: %q", nodes[2]["value"])
	}
	img, _ := nodes[3]["image"].(map[string]any)
	if img["hash"] != w5ImageHash {
		t.Fatalf("image node %v", nodes[3])
	}
	if asInt(img["width"]) != 64 {
		t.Fatalf("image meta not resolved: %v", img)
	}
	if nodes[4]["value"] != " end" {
		t.Fatalf("tail %q", nodes[4]["value"])
	}

	// The source face gives back exactly what is stored, tokens included.
	id := asInt(out["id"])
	resp, src := f.commentCall(t, http.MethodGet, "/source", id, "sess-alice", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("source %d %+v", resp.StatusCode, src)
	}
	if src["object"] != "comment_source" || src["text"] != text {
		t.Fatalf("source %+v", src)
	}
}

func TestV1CommentContentEmptyAndBreaks(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(81), map[string]any{"text": "a\n\nb"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	nodes := commentNodes(t, out)
	if len(nodes) != 4 {
		t.Fatalf("nodes %v", nodes)
	}
	if nodes[1]["object"] != "break" || nodes[2]["object"] != "break" {
		t.Fatalf("a blank line is two breaks, got %v", nodes)
	}
	if docPlainText(t, commentContent(t, out)) != "a\n\nb" {
		t.Fatalf("round trip %q", docPlainText(t, commentContent(t, out)))
	}
}

func TestV1GetCommentSourceNeedsEdit(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.commentCall(t, http.MethodGet, "/source", w5CommentAlice, "sess-alice", nil)
	if resp.StatusCode != http.StatusOK || out["text"] != "alice-comment" {
		t.Fatalf("author %d %+v", resp.StatusCode, out)
	}
	if out["comment_id"] != strconv.Itoa(w5CommentAlice) || out["reply_id"] != strconv.Itoa(w3ReplyMin) {
		t.Fatalf("source ids %+v", out)
	}

	resp, out = f.commentCall(t, http.MethodGet, "/source", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("another user %d %+v", resp.StatusCode, out)
	}

	resp, out = f.commentCall(t, http.MethodGet, "/source", w5CommentAlice, "", nil)
	if resp.StatusCode != http.StatusUnauthorized || out["code"] != "MISSING_CREDENTIAL" {
		t.Fatalf("anonymous %d %+v", resp.StatusCode, out)
	}
}

// The comment read face orders by (created, id). Two comments share an instant
// here, so dropping the id tie-breaker makes the order unstable.
func TestV1ReplyCommentsOrderBreaksTiesByID(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, body := f.doJSON(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(w3ReplyMin+2), "sess-alice",
		"/replies/{reply_id}", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get reply %d %s", resp.StatusCode, body)
	}
	reply := replyBody(t, body)
	comments, _ := reply["comments"].([]any)

	var want []string
	rows, err := f.db.Raw(`SELECT id FROM topic_comment WHERE topic_reply_id = ? AND status = 0 ORDER BY created ASC, id ASC`,
		w3ReplyMin+2).Rows()
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		want = append(want, strconv.Itoa(id))
	}
	_ = rows.Close()

	if len(comments) != len(want) {
		t.Fatalf("comments %d, want %d", len(comments), len(want))
	}
	for i, raw := range comments {
		c, _ := raw.(map[string]any)
		if c["id"] != want[i] {
			t.Fatalf("comment %d id %v, want %s (order: %v)", i, c["id"], want[i], want)
		}
	}
}
