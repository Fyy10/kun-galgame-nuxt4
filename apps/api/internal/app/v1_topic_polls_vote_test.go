package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func firstError(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	errs, _ := out["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors %v", out["errors"])
	}
	e, _ := errs[0].(map[string]any)
	return e
}

// K16: the slot is set, not toggled, and only a row that really moved moves a
// counter. An unconditional vote_count + 1 would double every repeat.
func TestV1SetPollVoteIsASlot(t *testing.T) {
	f := newPollFix(t, nil)
	a, b := strconv.Itoa(w5bOptAlwaysA), strconv.Itoa(w5bOptAlwaysB)

	resp, poll := f.voteSet(t, w5bPollAlways, "sess-bob", []string{a})
	if resp.StatusCode != http.StatusOK || poll["object"] != "poll" {
		t.Fatalf("first vote %d %+v", resp.StatusCode, poll)
	}
	if got := pollVoteCounts(t, poll); got[a] != 1 || got[b] != 0 {
		t.Fatalf("counts after the first vote %v", got)
	}
	v := viewerOf(t, poll)
	if v["has_voted"] != true {
		t.Fatalf("viewer %v", v)
	}
	if ids, _ := v["chosen_option_ids"].([]any); len(ids) != 1 || strID(ids[0]) != a {
		t.Fatalf("chosen_option_ids %v", v["chosen_option_ids"])
	}

	for i := range 2 {
		resp, poll = f.voteSet(t, w5bPollAlways, "sess-bob", []string{a})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("repeat %d: %d %+v", i, resp.StatusCode, poll)
		}
		if got := pollVoteCounts(t, poll); got[a] != 1 {
			t.Fatalf("repeat %d moved a counter: %v", i, got)
		}
	}
	if n := f.storedVoteCount(t, w5bOptAlwaysA); n != 1 {
		t.Fatalf("stored vote_count %d after three identical PUTs", n)
	}
	if n := f.storedVoteRows(t, w5bPollAlways, w5bOptAlwaysA); n != 1 {
		t.Fatalf("stored rows %d", n)
	}

	// Widening the choice adds only the new row.
	resp, poll = f.voteSet(t, w5bPollAlways, "sess-bob", []string{a, b})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("widen %d %+v", resp.StatusCode, poll)
	}
	if got := pollVoteCounts(t, poll); got[a] != 1 || got[b] != 1 {
		t.Fatalf("counts after widening %v", got)
	}
	// Narrowing removes only the one given up.
	resp, poll = f.voteSet(t, w5bPollAlways, "sess-bob", []string{b})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("narrow %d %+v", resp.StatusCode, poll)
	}
	if got := pollVoteCounts(t, poll); got[a] != 0 || got[b] != 1 {
		t.Fatalf("counts after narrowing %v", got)
	}
	if n := f.storedVoteCount(t, w5bOptAlwaysA); n != 0 {
		t.Fatalf("stored vote_count of the abandoned option %d", n)
	}
}

func TestV1ClearPollVote(t *testing.T) {
	f := newPollFix(t, nil)
	a := strconv.Itoa(w5bOptAlwaysA)

	if resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{a}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	resp, poll := f.voteClear(t, w5bPollAlways, "sess-bob")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear %d %+v", resp.StatusCode, poll)
	}
	if got := pollVoteCounts(t, poll); got[a] != 0 {
		t.Fatalf("counts after clearing %v", got)
	}
	if viewerOf(t, poll)["has_voted"] != false {
		t.Fatalf("viewer %v", viewerOf(t, poll))
	}
	if n := f.storedVoteRows(t, w5bPollAlways, w5bOptAlwaysA); n != 0 {
		t.Fatalf("stored rows %d", n)
	}

	// Retracting what was never cast is a 200 that changes nothing.
	resp, poll = f.voteClear(t, w5bPollAlways, "sess-bob")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second clear %d %+v", resp.StatusCode, poll)
	}
	if n := f.storedVoteCount(t, w5bOptAlwaysA); n != 0 {
		t.Fatalf("stored vote_count %d", n)
	}
}

func TestV1SetPollVoteRejectsAnotherPollsOption(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{strconv.Itoa(w5bOptSingleA)})
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
		t.Fatalf("cross-poll option %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["pointer"] != "/option_ids/0" || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Fatalf("errors[0] %v", e)
	}
	if n := f.storedVoteCount(t, w5bOptSingleA); n != 0 {
		t.Fatalf("another poll's option gained %d votes", n)
	}
	if n := f.storedVoteRows(t, w5bPollAlways, w5bOptSingleA); n != 0 {
		t.Fatalf("a vote row was written across polls: %d", n)
	}
}

func TestV1SetPollVoteRejectsADuplicateOption(t *testing.T) {
	f := newPollFix(t, nil)
	a := strconv.Itoa(w5bOptAlwaysA)

	resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{a, a})
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
		t.Fatalf("duplicate option %d %+v (a unique index must not turn this into a 500)", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["reason"] != "DUPLICATE_ITEM" || !strings.HasPrefix(fmt.Sprint(e["pointer"]), "/option_ids") {
		t.Fatalf("errors[0] %v", e)
	}
}

func TestV1SetPollVoteHonoursTheChoiceBounds(t *testing.T) {
	f := newPollFix(t, nil)

	// single accepts exactly one.
	resp, out := f.voteSet(t, w5bPollSingle, "sess-bob",
		[]string{strconv.Itoa(w5bOptSingleA), strconv.Itoa(w5bOptSingleB)})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("two options in a single-choice poll %d %+v", resp.StatusCode, out)
	}
	e := firstError(t, out)
	if e["pointer"] != "/option_ids" || e["reason"] != "TOO_MANY_ITEMS" {
		t.Fatalf("errors[0] %v", e)
	}
	params, _ := e["params"].(map[string]any)
	if asInt(params["max_items"]) != 1 {
		t.Fatalf("params %v", e["params"])
	}

	// max_choice is 2 on w5bPollAlways.
	resp, out = f.voteSet(t, w5bPollAlways, "sess-bob", []string{
		strconv.Itoa(w5bOptAlwaysA), strconv.Itoa(w5bOptAlwaysB), strconv.Itoa(w5bOptAlwaysC)})
	if resp.StatusCode != http.StatusUnprocessableEntity || firstError(t, out)["reason"] != "TOO_MANY_ITEMS" {
		t.Fatalf("three options over max_choice %d %+v", resp.StatusCode, out)
	}

	if resp, out := f.pollPatch(t, w5bPollLog, "sess-alice", map[string]any{"min_choice": 2}); resp.StatusCode != http.StatusOK {
		t.Fatalf("raise min_choice %d %+v", resp.StatusCode, out)
	}
	resp, out = f.voteSet(t, w5bPollLog, "sess-bob", []string{strconv.Itoa(w5bOptLogA)})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("one option under min_choice %d %+v", resp.StatusCode, out)
	}
	e = firstError(t, out)
	if e["pointer"] != "/option_ids" || e["reason"] != "TOO_FEW_ITEMS" {
		t.Fatalf("errors[0] %v", e)
	}
}

func TestV1SetPollVoteRefusesAClosedPoll(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.voteSet(t, w5bPollClosed, "sess-bob", []string{strconv.Itoa(w5bOptClosedA)})
	if resp.StatusCode != http.StatusConflict || out["code"] != "POLL_CLOSED" {
		t.Fatalf("vote past closes_at %d %+v", resp.StatusCode, out)
	}
	if n := f.storedVoteRows(t, w5bPollClosed, w5bOptClosedA); n != 0 {
		t.Fatalf("a closed poll took %d rows", n)
	}
	if n := f.storedVoteCount(t, w5bOptClosedA); n != 0 {
		t.Fatalf("a closed poll moved a counter to %d", n)
	}
}

func TestV1SetPollVoteRefusesAChangeWhenThePollForbidsIt(t *testing.T) {
	f := newPollFix(t, nil)
	a, b := strconv.Itoa(w5bOptSingleA), strconv.Itoa(w5bOptSingleB)

	if resp, out := f.voteSet(t, w5bPollSingle, "sess-bob", []string{a}); resp.StatusCode != http.StatusOK {
		t.Fatalf("first vote %d %+v", resp.StatusCode, out)
	}
	resp, out := f.voteSet(t, w5bPollSingle, "sess-bob", []string{b})
	if resp.StatusCode != http.StatusConflict || out["code"] != "VOTE_ALREADY_CAST" {
		t.Fatalf("changing a vote the poll pinned %d %+v", resp.StatusCode, out)
	}
	if n := f.storedVoteCount(t, w5bOptSingleB); n != 0 {
		t.Fatalf("the refused change moved a counter to %d", n)
	}
	if n := f.storedVoteCount(t, w5bOptSingleA); n != 1 {
		t.Fatalf("the refused change disturbed the stored vote: %d", n)
	}

	// Repeating the same choice is the slot being set to what it holds.
	resp, poll := f.voteSet(t, w5bPollSingle, "sess-bob", []string{a})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("repeat of the same choice %d %+v", resp.StatusCode, poll)
	}
	if got := pollVoteCounts(t, poll); got[a] != 1 {
		t.Fatalf("counts %v", got)
	}

	resp, out = f.voteClear(t, w5bPollSingle, "sess-bob")
	if resp.StatusCode != http.StatusConflict || out["code"] != "VOTE_ALREADY_CAST" {
		t.Fatalf("retracting a vote the poll pinned %d %+v", resp.StatusCode, out)
	}
	if viewerOf(t, mustPoll(t, f, w5bPollSingle, "sess-bob"))["can_vote"] != false {
		t.Fatal("can_vote must be false once the vote is pinned")
	}
}

func TestV1PollVoteFollowsTopicVisibility(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.voteSet(t, w5bPollRole, "sess-other", []string{strconv.Itoa(w5bOptRoleA)})
	if resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
		t.Fatalf("vote in a role-scoped topic %d %+v", resp.StatusCode, out)
	}
	if n := f.storedVoteRows(t, w5bPollRole, w5bOptRoleA); n != 0 {
		t.Fatalf("rows %d", n)
	}
	resp, out = f.voteClear(t, w5bPollRole, "sess-other")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("clear in a role-scoped topic %d %+v", resp.StatusCode, out)
	}
}
