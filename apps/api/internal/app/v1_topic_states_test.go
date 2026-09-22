package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

const statesPath = "/api/v1/me/topic-states"

func newStateFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	return f
}

func (f *writeFix) topicStates(t *testing.T, session string, ids ...string) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodGet, statesPath+"?topic_ids="+strings.Join(ids, ","),
		session, "/me/topic-states", "", nil, nil)
	return resp, problemMap(t, body)
}

func stateByTopic(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := map[string]map[string]any{}
	for _, item := range raw {
		row, _ := item.(map[string]any)
		if row["object"] != "topic_state" {
			t.Errorf("object %v", row["object"])
		}
		out[strID(row["topic_id"])] = row
	}
	return out
}

func missingIDs(body map[string]any) []string {
	raw, _ := body["missing"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, strID(v))
	}
	return out
}

func TestV1TopicStatesMixedRequest(t *testing.T) {
	f := newStateFix(t)

	if err := f.db.Exec(`INSERT INTO topic_favorite (topic_id, user_id, updated) VALUES (?, ?, NOW())`,
		w3TopicPub, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"like", "heart"} {
		if err := f.db.Exec(`INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, ?, NOW())`,
			w3TopicFloors, w3UserAlice, token).Error; err != nil {
			t.Fatal(err)
		}
	}

	pub := strconv.Itoa(w3TopicPub)
	floors := strconv.Itoa(w3TopicFloors)
	role := strconv.Itoa(w3TopicRole)
	gone := "930000499"

	resp, body := f.topicStates(t, "sess-other", pub, floors, role, gone)
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("states %d %+v", resp.StatusCode, body)
	}
	if body["next_cursor"] != nil {
		t.Errorf("a batch read is not paginated, got next_cursor %v", body["next_cursor"])
	}

	items := stateByTopic(t, body)
	// other holds no role grant, so the role-scoped topic is missing, and the
	// id that does not exist is missing for the same reason as far as the
	// caller can tell.
	if got := missingIDs(body); fmt.Sprint(got) != fmt.Sprint([]string{role, gone}) {
		t.Fatalf("missing %v, want [%s %s]", got, role, gone)
	}
	if len(items) != 2 {
		t.Fatalf("items %v", items)
	}
	// A readable topic the caller has no state on is an answer, not a miss.
	for _, id := range []string{pub, floors} {
		row, ok := items[id]
		if !ok {
			t.Fatalf("topic %s should be present: %v", id, items)
		}
		if row["has_favorited"] != false {
			t.Errorf("topic %s has_favorited %v for a caller with no state", id, row["has_favorited"])
		}
		if fmt.Sprint(row["reaction_tokens"]) != "[]" {
			t.Errorf("topic %s reaction_tokens must be [], got %v", id, row["reaction_tokens"])
		}
	}
}

// The single easiest mistake in a me face is dropping the user_id predicate.
func TestV1TopicStatesAreTheCallersOwn(t *testing.T) {
	f := newStateFix(t)

	if err := f.db.Exec(`INSERT INTO topic_favorite (topic_id, user_id, updated) VALUES (?, ?, NOW())`,
		w3TopicPub, w3UserBob).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, 'clap', NOW())`,
		w3TopicPub, w3UserBob).Error; err != nil {
		t.Fatal(err)
	}

	pub := strconv.Itoa(w3TopicPub)
	_, body := f.topicStates(t, "sess-alice", pub)
	row := stateByTopic(t, body)[pub]
	if row["has_favorited"] != false {
		t.Errorf("bob's favorite showed up as alice's: %+v", row)
	}
	if fmt.Sprint(row["reaction_tokens"]) != "[]" {
		t.Errorf("bob's reaction showed up as alice's: %+v", row)
	}

	_, body = f.topicStates(t, "sess-bob", pub)
	row = stateByTopic(t, body)[pub]
	if row["has_favorited"] != true || fmt.Sprint(row["reaction_tokens"]) != "[clap]" {
		t.Errorf("bob cannot see his own state: %+v", row)
	}
}

func TestV1TopicStatesReturnsReactionsInOrder(t *testing.T) {
	f := newStateFix(t)

	for _, token := range []string{"heart", "like", "clap"} {
		if err := f.db.Exec(`INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, ?, NOW())`,
			w3TopicPub, w3UserAlice, token).Error; err != nil {
			t.Fatal(err)
		}
	}
	pub := strconv.Itoa(w3TopicPub)
	_, body := f.topicStates(t, "sess-alice", pub)
	row := stateByTopic(t, body)[pub]
	if fmt.Sprint(row["reaction_tokens"]) != "[heart like clap]" {
		t.Fatalf("reaction_tokens come back oldest first, got %v", row["reaction_tokens"])
	}
}

func TestV1TopicStatesIDLimits(t *testing.T) {
	f := newStateFix(t)

	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(930000001 + i)
	}
	// The cap lives in the schema, so the platform layer refuses it before the
	// handler runs: 400 INVALID_PARAMETER, not the handler's 422.
	resp, body := f.topicStates(t, "sess-alice", ids...)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("101 ids %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors %v", body["errors"])
	}
	e, _ := errs[0].(map[string]any)
	if e["reason"] != "TOO_MANY_ITEMS" || e["parameter"] != "topic_ids" {
		t.Fatalf("field error %+v", e)
	}
	params, _ := e["params"].(map[string]any)
	if asInt(params["max_items"]) != 100 {
		t.Errorf("max_items %v", params)
	}

	// Exactly 100 is the limit, not over it.
	resp, body = f.topicStates(t, "sess-alice", ids[:100]...)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("100 ids %d %+v", resp.StatusCode, body)
	}
	if len(missingIDs(body)) != 100 {
		t.Errorf("none of those ids exist, so all 100 are missing: %v", body["missing"])
	}
	if fmt.Sprint(body["items"]) != "[]" {
		t.Errorf("items must be [], got %v", body["items"])
	}

	// 0 is decimal digits, so only the handler can refuse it.
	resp, body = f.topicStates(t, "sess-alice", "0")
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
		t.Fatalf("a non-positive id %d %+v", resp.StatusCode, body)
	}
	resp, body = f.topicStates(t, "sess-alice", "abc")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("a non-numeric id %d %+v", resp.StatusCode, body)
	}
	// An absent required query parameter is refused by the platform layer
	// before the handler sees it, so it is INVALID_PARAMETER and 400, not the
	// handler's own VALIDATION_FAILED.
	resp, body = f.topicStates(t, "sess-alice", "")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("an empty topic_ids %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicStatesNeedCredentials(t *testing.T) {
	f := newStateFix(t)

	resp, body := f.topicStates(t, "", strconv.Itoa(w3TopicPub))
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Fatalf("anonymous %d %+v", resp.StatusCode, body)
	}
}

// A comment read on its own carries the floor a deep link scrolls to, and it
// is the same value the reply reports.
func TestV1CommentCarriesItsReplyFloor(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, comment := f.getComment(t, w5CommentAlice, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get comment %d %+v", resp.StatusCode, comment)
	}
	floor := asInt(comment["reply_floor"])
	if floor <= 0 {
		t.Fatalf("reply_floor %v", comment["reply_floor"])
	}

	replyID := strID(comment["reply_id"])
	resp, reply := f.doJSONMap(t, http.MethodGet, "/api/v1/replies/"+replyID, "sess-alice", "/replies/{reply_id}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get reply %d %+v", resp.StatusCode, reply)
	}
	if asInt(reply["floor"]) != floor {
		t.Fatalf("comment says floor %d, its reply says %v", floor, reply["floor"])
	}

	// The same comment embedded in the reply carries the same value.
	embedded, _ := reply["comments"].([]any)
	found := false
	for _, item := range embedded {
		row, _ := item.(map[string]any)
		if strID(row["id"]) != strID(comment["id"]) {
			continue
		}
		found = true
		if asInt(row["reply_floor"]) != floor {
			t.Fatalf("embedded reply_floor %v, standalone %d", row["reply_floor"], floor)
		}
	}
	if !found {
		t.Fatalf("comment %v is not among its reply's comments", comment["id"])
	}
}
