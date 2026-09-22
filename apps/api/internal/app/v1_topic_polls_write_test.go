package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func pollBody(options ...string) map[string]any {
	opts := make([]map[string]any, len(options))
	for i, text := range options {
		opts[i] = map[string]any{"text": text}
	}
	return map[string]any{
		"title":             "which route",
		"choice_type":       "multiple",
		"result_visibility": "always",
		"options":           opts,
	}
}

func TestV1CreatePollValuesAndSideEffects(t *testing.T) {
	f := newPollFix(t, nil)

	before := f.scalar(t, `SELECT EXTRACT(EPOCH FROM status_update_time)::int FROM topic WHERE id = ?`, w3TopicPub)
	body := pollBody("a", "b", "c")
	body["description"] = " and why "
	body["closes_at"] = "2027-01-02T03:04:05Z"
	body["is_anonymous"] = true
	body["can_change_vote"] = false

	resp, poll := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(500), body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, poll)
	}
	id := strID(poll["id"])
	if got := resp.Header.Get("Location"); got != "/api/v1/polls/"+id {
		t.Fatalf("location %q", got)
	}
	if poll["object"] != "poll" || strID(poll["topic_id"]) != strconv.Itoa(w3TopicPub) {
		t.Fatalf("object/topic_id %+v", poll)
	}
	if poll["title"] != "which route" || poll["description"] != "and why" {
		t.Fatalf("title/description %+v", poll)
	}
	if poll["closes_at"] != "2027-01-02T03:04:05Z" {
		t.Fatalf("closes_at %v; the write face must not move the instant", poll["closes_at"])
	}
	if poll["is_anonymous"] != true || poll["can_change_vote"] != false {
		t.Fatalf("flags %+v", poll)
	}
	// multiple with no bounds: one to every option.
	if asInt(poll["min_choice"]) != 1 || asInt(poll["max_choice"]) != 3 {
		t.Fatalf("default bounds %+v", poll)
	}
	results := pollResultsOf(t, poll)
	if asInt(results["total_vote_count"]) != 0 || len(results["sample_voters"].([]any)) != 0 {
		t.Fatalf("fresh results %+v", results)
	}
	author, _ := poll["author"].(map[string]any)
	if author["id"] != strconv.Itoa(w3UserAlice) {
		t.Fatalf("author %v", poll["author"])
	}

	var updated any
	pollID, _ := strconv.Atoi(id)
	if err := f.db.Raw(`SELECT updated FROM topic_poll WHERE id = ?`, pollID).Row().Scan(&updated); err != nil {
		t.Fatal(err)
	}
	if updated == nil {
		t.Fatal("topic_poll.updated is NOT NULL with no default; the insert must set it")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll_option WHERE poll_id = ?`, pollID); n != 3 {
		t.Fatalf("options stored %d", n)
	}
	after := f.scalar(t, `SELECT EXTRACT(EPOCH FROM status_update_time)::int FROM topic WHERE id = ?`, w3TopicPub)
	if after <= before {
		t.Fatalf("creating a poll bumps the topic: %d then %d", before, after)
	}

	// single pins both bounds to 1 whatever the body asks.
	single := pollBody("x", "y")
	single["choice_type"] = "single"
	single["min_choice"] = 2
	single["max_choice"] = 2
	resp, poll = f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(501), single)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("single create %d %+v", resp.StatusCode, poll)
	}
	if asInt(poll["min_choice"]) != 1 || asInt(poll["max_choice"]) != 1 {
		t.Fatalf("single bounds %+v", poll)
	}
}

func TestV1CreatePollChecksTopicVisibilityAndPermission(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.pollCreate(t, w3TopicRole, "sess-other", keyUUID(510), pollBody("a", "b"))
	if resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
		t.Fatalf("create into a topic the caller cannot read %d %+v", resp.StatusCode, out)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll WHERE topic_id = ?`, w3TopicRole); n != 1 {
		t.Fatalf("the role topic gained a poll: %d", n)
	}

	resp, out = f.pollCreate(t, w3TopicPub, "sess-bob", keyUUID(511), pollBody("a", "b"))
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("a stranger creating a poll %d %+v", resp.StatusCode, out)
	}
	// poll.create_any reaches another author's topic.
	resp, out = f.pollCreate(t, w3TopicPub, "sess-staff", keyUUID(512), pollBody("a", "b"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("staff create %d %+v", resp.StatusCode, out)
	}
}

func TestV1CreatePollRejectsABlankOrOverlongLabel(t *testing.T) {
	f := newPollFix(t, nil)

	for i, c := range []struct{ name, label string }{{"empty", ""}, {"whitespace", "   "}} {
		name := c.name
		body := pollBody("a", "b")
		body["options"] = []map[string]any{{"text": c.label}, {"text": "b"}}
		resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(520+i), body)
		if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
			t.Fatalf("%s label %d %+v (a varchar(100) overflow must not become a 500 either)", name, resp.StatusCode, out)
		}
		e := firstError(t, out)
		if e["pointer"] != "/options/0/text" || e["reason"] != "TOO_SHORT" {
			t.Fatalf("%s label errors[0] %v", name, e)
		}
	}

	body := pollBody("a", "b")
	body["options"] = []map[string]any{{"text": strings.Repeat("x", 101)}, {"text": "b"}}
	resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(530), body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("overlong label %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["pointer"] != "/options/0/text" || e["reason"] != "TOO_LONG" {
		t.Fatalf("overlong label errors[0] %v", e)
	}
	params, _ := e["params"].(map[string]any)
	if asInt(params["max_length"]) != 100 {
		t.Fatalf("params %v", e["params"])
	}

	body = pollBody("only one")
	resp, out = f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(531), body)
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["reason"] != "TOO_FEW_ITEMS" {
		t.Fatalf("one option %d %+v", resp.StatusCode, out)
	}
}

// The legacy face threw the parse error away and stored a null deadline.
func TestV1CreatePollRejectsADeadlineItCannotParse(t *testing.T) {
	f := newPollFix(t, nil)

	for i, c := range []struct{ name, value string }{
		// These two match the schema's pattern and reach the handler, which is
		// where the legacy face threw the error away.
		{"impossible day", "2026-02-30T00:00:00Z"},
		{"month 13", "2026-13-01T00:00:00Z"},
		{"date only", "2026-02-01"},
		{"with offset", "2026-02-01T00:00:00+08:00"},
	} {
		body := pollBody("a", "b")
		body["closes_at"] = c.value
		resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(540+i), body)
		if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
			t.Fatalf("%s: %d %+v", c.name, resp.StatusCode, out)
		}
		if !hasFieldError(t, out, "/closes_at", "INVALID_FORMAT") {
			t.Fatalf("%s: errors %v", c.name, out["errors"])
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll WHERE topic_id = ? AND deadline IS NULL`, w3TopicPub); n != 6 {
		t.Fatalf("a poll was created with a silently null deadline: %d null-deadline polls", n)
	}
}

func TestV1CreatePollNeedsAnIdempotencyKeyAndReplaysIt(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", "", pollBody("a", "b"))
	if resp.StatusCode != http.StatusBadRequest || out["code"] != "INVALID_PARAMETER" {
		t.Fatalf("no Idempotency-Key %d %+v", resp.StatusCode, out)
	}
	if e := firstError(t, out); e["header"] != "Idempotency-Key" || e["reason"] != "REQUIRED" {
		t.Fatalf("errors[0] %v", firstError(t, out))
	}

	key := keyUUID(550)
	resp, first := f.pollCreate(t, w3TopicPub, "sess-alice", key, pollBody("a", "b"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, first)
	}
	resp, again := f.pollCreate(t, w3TopicPub, "sess-alice", key, pollBody("a", "b"))
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay %d %q", resp.StatusCode, resp.Header.Get("Idempotency-Replayed"))
	}
	if strID(again["id"]) != strID(first["id"]) {
		t.Fatalf("replay made a second poll %v vs %v", again["id"], first["id"])
	}
	resp, out = f.pollCreate(t, w3TopicPub, "sess-alice", key, pollBody("c", "d"))
	if resp.StatusCode != http.StatusConflict || out["code"] != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("same key, other body %d %+v", resp.StatusCode, out)
	}
}

func TestV1CreatePollStopsAtTheTopicCap(t *testing.T) {
	f := newPollFix(t, nil)
	at := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)
	// Six polls already sit on this topic; fill the rest of the 30.
	for i := range 24 {
		if err := f.db.Exec(`INSERT INTO topic_poll (title, description, type, min_choice, max_choice, status,
			result_visibility, is_anonymous, can_change_vote, topic_id, user_id, created, updated)
			VALUES (?, '', 'single', 1, 1, 'open', 'always', false, true, ?, ?, ?, ?)`,
			fmt.Sprintf("filler %d", i), w3TopicPub, w3UserAlice, at, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(560), pollBody("a", "b"))
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
		t.Fatalf("the 31st poll %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["parameter"] != "topic_id" || e["reason"] != "TOO_MANY_ITEMS" {
		t.Fatalf("errors[0] %v", e)
	}
}

func TestV1CreatePollRunsTheTrustGate(t *testing.T) {
	f := newPollFix(t, denyChecker{})

	resp, out := f.pollCreate(t, w3TopicPub, "sess-alice", keyUUID(570), pollBody("a", "b"))
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "CONTENT_REJECTED" {
		t.Fatalf("denied text %d %+v", resp.StatusCode, out)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll WHERE title = 'which route'`); n != 0 {
		t.Fatalf("a refused poll was written: %d", n)
	}
}

func TestV1UpdatePollIsPartial(t *testing.T) {
	f := newPollFix(t, nil)

	resp, poll := f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{"title": "  renamed  "})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d %+v", resp.StatusCode, poll)
	}
	if poll["title"] != "renamed" {
		t.Fatalf("title %v", poll["title"])
	}
	if poll["result_visibility"] != "always" || poll["is_anonymous"] != false || poll["can_change_vote"] != true {
		t.Fatalf("an absent field must keep its value: %+v", poll)
	}
	if len(pollOptionIDs(t, poll)) != 3 {
		t.Fatalf("options %v", poll["options"])
	}

	resp, poll = f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{
		"option_changes": map[string]any{
			"add":    []map[string]any{{"text": "d"}},
			"update": []map[string]any{{"option_id": strconv.Itoa(w5bOptAlwaysA), "text": "a2"}},
			"remove": []string{strconv.Itoa(w5bOptAlwaysC)},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("option changes %d %+v", resp.StatusCode, poll)
	}
	texts := map[string]string{}
	for _, item := range poll["options"].([]any) {
		row, _ := item.(map[string]any)
		texts[strID(row["id"])] = fmt.Sprint(row["text"])
	}
	if texts[strconv.Itoa(w5bOptAlwaysA)] != "a2" {
		t.Fatalf("relabel %v", texts)
	}
	if _, gone := texts[strconv.Itoa(w5bOptAlwaysC)]; gone {
		t.Fatalf("remove %v", texts)
	}
	if len(texts) != 3 {
		t.Fatalf("resulting options %v", texts)
	}
}

func TestV1UpdatePollRejectsForeignAndVotedOptions(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{
		"option_changes": map[string]any{
			"update": []map[string]any{{"option_id": strconv.Itoa(w5bOptSingleA), "text": "x"}},
		},
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("another poll's option %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["pointer"] != "/option_changes/update/0/option_id" || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Fatalf("errors[0] %v", e)
	}

	if resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{strconv.Itoa(w5bOptAlwaysA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	resp, out = f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{
		"option_changes": map[string]any{
			"update": []map[string]any{{"option_id": strconv.Itoa(w5bOptAlwaysA), "text": "renamed"}},
		},
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["reason"] != "IMMUTABLE" {
		t.Fatalf("relabelling a voted option %d %+v", resp.StatusCode, out)
	}
	resp, out = f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{
		"option_changes": map[string]any{"remove": []string{strconv.Itoa(w5bOptAlwaysA)}},
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["reason"] != "IMMUTABLE" {
		t.Fatalf("removing a voted option %d %+v", resp.StatusCode, out)
	}
	resp, out = f.pollPatch(t, w5bPollAlways, "sess-alice", map[string]any{
		"option_changes": map[string]any{"remove": []string{
			strconv.Itoa(w5bOptAlwaysB), strconv.Itoa(w5bOptAlwaysC)}},
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["reason"] != "TOO_FEW_ITEMS" {
		t.Fatalf("leaving one option %d %+v", resp.StatusCode, out)
	}
}

// Census §10.6: votes cast under a promise of anonymity were de-anonymized by
// flipping the flag, and the vote log then named every one of them.
func TestV1UpdatePollKeepsAnonymityAndChoiceTypeOnceVoted(t *testing.T) {
	f := newPollFix(t, nil)

	if resp, out := f.pollPatch(t, w5bPollAnon, "sess-alice", map[string]any{"is_anonymous": false}); resp.StatusCode != http.StatusOK {
		t.Fatalf("an unvoted anonymous poll may still be opened up: %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.pollPatch(t, w5bPollAnon, "sess-alice", map[string]any{"is_anonymous": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("back to anonymous %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.voteSet(t, w5bPollAnon, "sess-bob", []string{strconv.Itoa(w5bOptAnonA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	resp, out := f.pollPatch(t, w5bPollAnon, "sess-alice", map[string]any{"is_anonymous": false})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("de-anonymizing a voted poll %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["pointer"] != "/is_anonymous" || e["reason"] != "IMMUTABLE" {
		t.Fatalf("errors[0] %v", e)
	}
	if !f.boolScalar(t, `SELECT is_anonymous FROM topic_poll WHERE id = ?`, w5bPollAnon) {
		t.Fatal("the stored flag was flipped anyway")
	}

	if resp, out := f.voteSet(t, w5bPollLog, "sess-bob", []string{strconv.Itoa(w5bOptLogA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	resp, out = f.pollPatch(t, w5bPollLog, "sess-alice", map[string]any{"choice_type": "single"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("flipping a voted poll to single %d %+v", resp.StatusCode, out)
	}
	if e := firstError(t, out); e["pointer"] != "/choice_type" || e["reason"] != "IMMUTABLE" {
		t.Fatalf("errors[0] %v", e)
	}
}

func TestV1UpdatePollClearsTheDeadline(t *testing.T) {
	f := newPollFix(t, nil)

	resp, poll := f.pollPatch(t, w5bPollClosed, "sess-alice", map[string]any{"title": "still closed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d %+v", resp.StatusCode, poll)
	}
	if poll["closes_at"] == nil {
		t.Fatal("a patch that leaves closes_at out keeps it")
	}
	resp, poll = f.pollPatch(t, w5bPollClosed, "sess-alice", map[string]any{"closes_at": nil})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear %d %+v", resp.StatusCode, poll)
	}
	if poll["closes_at"] != nil {
		t.Fatalf("closes_at %v", poll["closes_at"])
	}
	if resp, out := f.voteSet(t, w5bPollClosed, "sess-bob", []string{strconv.Itoa(w5bOptClosedA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("the reopened poll takes votes: %d %+v", resp.StatusCode, out)
	}
	resp, out := f.pollPatch(t, w5bPollClosed, "sess-alice", map[string]any{"closes_at": "2026-02-30T00:00:00Z"})
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["pointer"] != "/closes_at" {
		t.Fatalf("unparsable deadline on patch %d %+v", resp.StatusCode, out)
	}
}

func TestV1UpdatePollPermission(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.pollPatch(t, w5bPollAlways, "sess-bob", map[string]any{"title": "hijack"})
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("a stranger patching %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.pollPatch(t, w5bPollAlways, "sess-staff", map[string]any{"title": "moderated"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("poll.edit_any %d %+v", resp.StatusCode, out)
	}
	// A poll on a topic the caller cannot read is not there at all.
	resp, out = f.pollPatch(t, w5bPollRole, "sess-other", map[string]any{"title": "x"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("patch into a role-scoped topic %d %+v", resp.StatusCode, out)
	}
}

func TestV1DeletePoll(t *testing.T) {
	f := newPollFix(t, nil)

	if resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{strconv.Itoa(w5bOptAlwaysA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	if resp := f.pollDelete(t, w5bPollAlways, "sess-bob"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a stranger deleting %d", resp.StatusCode)
	}
	resp := f.pollDelete(t, w5bPollAlways, "sess-alice")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM topic_poll WHERE id = ?`,
		`SELECT COUNT(*) FROM topic_poll_option WHERE poll_id = ?`,
		`SELECT COUNT(*) FROM topic_poll_vote WHERE poll_id = ?`,
	} {
		if n := f.scalar(t, q, w5bPollAlways); n != 0 {
			t.Fatalf("%s left %d rows", q, n)
		}
	}
	if resp, out := f.pollGet(t, w5bPollAlways, "sess-alice"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete %d %+v", resp.StatusCode, out)
	}
	if resp := f.pollDelete(t, 930000999, "sess-alice"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete a missing poll %d", resp.StatusCode)
	}
}

func hasFieldError(t *testing.T, out map[string]any, pointer, reason string) bool {
	t.Helper()
	errs, _ := out["errors"].([]any)
	for _, raw := range errs {
		e, _ := raw.(map[string]any)
		if e["pointer"] == pointer && e["reason"] == reason {
			return true
		}
	}
	return false
}

func (f *writeFix) boolScalar(t *testing.T, query string, args ...any) bool {
	t.Helper()
	var b bool
	if err := f.db.Raw(query, args...).Scan(&b).Error; err != nil {
		t.Fatal(err)
	}
	return b
}
