package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func pollItems(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		row, _ := item.(map[string]any)
		out = append(out, row)
	}
	return out
}

func TestV1ListTopicPollsShapeAndOrder(t *testing.T) {
	f := newPollFix(t, nil)

	resp, body := f.pollList(t, w3TopicPub, "sess-alice")
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	if body["next_cursor"] != nil {
		t.Errorf("the poll list is not paginated, got next_cursor %v", body["next_cursor"])
	}
	items := pollItems(t, body)
	var ids []string
	for _, poll := range items {
		ids = append(ids, strID(poll["id"]))
		if poll["object"] != "poll" {
			t.Errorf("object %v", poll["object"])
		}
		if _, ok := poll["status"]; ok {
			t.Errorf("status must not be sent: %v", poll)
		}
		if _, ok := poll["state"]; ok {
			t.Errorf("state must not be sent: %v", poll)
		}
	}
	// created DESC, id DESC; the banned author's poll is left out.
	want := []string{
		strconv.Itoa(w5bPollLog), strconv.Itoa(w5bPollClosed),
		strconv.Itoa(w5bPollAnon), strconv.Itoa(w5bPollAfterVote),
		strconv.Itoa(w5bPollSingle), strconv.Itoa(w5bPollAlways),
	}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("order %v\nwant %v (w5bPollBanned and the role topic's poll must not appear)", ids, want)
	}

	first := items[0]
	if first["choice_type"] != "multiple" || asInt(first["min_choice"]) != 1 || asInt(first["max_choice"]) != 3 {
		t.Errorf("choice fields %+v", first)
	}
	if first["description"] != "" {
		t.Errorf("an empty description is the empty string, got %v", first["description"])
	}
	if first["closes_at"] != nil {
		t.Errorf("closes_at %v", first["closes_at"])
	}
	if len(pollOptionIDs(t, first)) != 2 {
		t.Errorf("options %v", first["options"])
	}
}

func TestV1ListTopicPollsAnonymousViewerIsNull(t *testing.T) {
	f := newPollFix(t, nil)

	resp, body := f.pollList(t, w3TopicPub, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous list %d %+v", resp.StatusCode, body)
	}
	for _, poll := range pollItems(t, body) {
		if poll["viewer"] != nil {
			t.Fatalf("anonymous viewer %v", poll["viewer"])
		}
	}
}

func TestV1GetPollViewerAndCaps(t *testing.T) {
	f := newPollFix(t, nil)

	resp, poll := f.pollGet(t, w5bPollAlways, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, poll)
	}
	v := viewerOf(t, poll)
	if v["has_voted"] != false || v["can_vote"] != true || v["can_edit"] != true || v["can_delete"] != true {
		t.Fatalf("author viewer %v", v)
	}
	if v["can_view_results"] != true || v["can_change_vote"] != true {
		t.Fatalf("author viewer %v", v)
	}
	if ids, _ := v["chosen_option_ids"].([]any); len(ids) != 0 {
		t.Fatalf("chosen_option_ids %v", v["chosen_option_ids"])
	}

	_, poll = f.pollGet(t, w5bPollAlways, "sess-bob")
	v = viewerOf(t, poll)
	if v["can_edit"] != false || v["can_delete"] != false {
		t.Fatalf("stranger caps %v", v)
	}

	// A closed poll refuses a vote, and the viewer flag says so before the try.
	_, poll = f.pollGet(t, w5bPollClosed, "sess-bob")
	if viewerOf(t, poll)["can_vote"] != false {
		t.Fatalf("closed poll can_vote %v", viewerOf(t, poll))
	}
}

func TestV1GetPollHidesResultsUntilTheViewerVotes(t *testing.T) {
	f := newPollFix(t, nil)

	resp, poll := f.pollGet(t, w5bPollAfterVote, "sess-bob")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, poll)
	}
	if poll["results"] != nil {
		t.Fatalf("result_visibility=after_vote must hide the whole results block, got %v", poll["results"])
	}
	if viewerOf(t, poll)["can_view_results"] != false {
		t.Fatalf("can_view_results %v", viewerOf(t, poll))
	}

	options := pollOptionIDs(t, poll)
	if resp, out := f.voteSet(t, w5bPollAfterVote, "sess-bob", options[:1]); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	_, poll = f.pollGet(t, w5bPollAfterVote, "sess-bob")
	results := pollResultsOf(t, poll)
	if asInt(results["total_votes"]) != 1 || asInt(results["voter_count"]) != 1 {
		t.Fatalf("results after voting %+v", results)
	}

	// The poll's author sees them whatever the visibility says.
	_, poll = f.pollGet(t, w5bPollAfterVote, "sess-alice")
	if poll["results"] == nil {
		t.Fatal("the poll's author always sees results")
	}
	// So does a moderator holding poll.view_restricted.
	_, poll = f.pollGet(t, w5bPollAfterVote, "sess-staff")
	if poll["results"] == nil {
		t.Fatal("poll.view_restricted sees results")
	}
}

func TestV1GetPollAnonymousHasNoSampleVoters(t *testing.T) {
	f := newPollFix(t, nil)

	options := pollOptionIDs(t, mustPoll(t, f, w5bPollAnon, "sess-bob"))
	if resp, out := f.voteSet(t, w5bPollAnon, "sess-bob", options[:1]); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	results := pollResultsOf(t, mustPoll(t, f, w5bPollAnon, "sess-alice"))
	if asInt(results["voter_count"]) != 1 {
		t.Fatalf("voter_count %v", results["voter_count"])
	}
	sample, ok := results["sample_voters"].([]any)
	if !ok || len(sample) != 0 {
		t.Fatalf("an anonymous poll names nobody, got %v", results["sample_voters"])
	}
}

func TestV1GetPollSampleVotersAreTheEarliestRenderable(t *testing.T) {
	f := newPollFix(t, nil)

	results := pollResultsOf(t, mustPoll(t, f, w5bPollLog, "sess-alice"))
	sample, _ := results["sample_voters"].([]any)
	var got []string
	for _, item := range sample {
		row, _ := item.(map[string]any)
		got = append(got, strID(row["id"]))
	}
	var want []string
	for i := range 5 {
		want = append(want, strconv.Itoa(w3MentionMin+i))
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("sample_voters %v\nwant the five earliest by (created, id) %v", got, want)
	}
	if asInt(results["total_votes"]) != 19 || asInt(results["voter_count"]) != 13 {
		t.Fatalf("totals %+v", results)
	}
}

func TestV1PollReadsFollowTopicVisibility(t *testing.T) {
	f := newPollFix(t, nil)

	for _, session := range []string{"", "sess-other"} {
		resp, out := f.pollGet(t, w5bPollRole, session)
		if resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
			t.Fatalf("session %q read a role-scoped topic's poll: %d %+v", session, resp.StatusCode, out)
		}
		resp, out = f.pollList(t, w3TopicRole, session)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("session %q listed a role-scoped topic's polls: %d %+v", session, resp.StatusCode, out)
		}
		resp, out = f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes", w5bPollRole), session)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("session %q read a role-scoped topic's vote log: %d %+v", session, resp.StatusCode, out)
		}
	}
	if resp, out := f.pollGet(t, w5bPollRole, "sess-alice"); resp.StatusCode != http.StatusOK {
		t.Fatalf("the topic's author reads its poll: %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.pollGet(t, w5bPollBanned, "sess-alice"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("a banned author's poll is not readable: %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.pollGet(t, 930000999, "sess-alice"); resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
		t.Fatalf("missing poll %d %+v", resp.StatusCode, out)
	}
}

// K17: the visibility decision needs OAuth, and a write must not proceed while
// it is unknown.
func TestV1PollWritesCloseWhenOAuthIsDown(t *testing.T) {
	f := newPollFix(t, nil)
	f.failOA.Store(true)

	resp, out := f.voteSet(t, w5bPollAlways, "sess-bob", []string{strconv.Itoa(w5bOptAlwaysA)})
	if resp.StatusCode != http.StatusServiceUnavailable || out["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("vote with OAuth down %d %+v", resp.StatusCode, out)
	}
	if n := f.storedVoteRows(t, w5bPollAlways, w5bOptAlwaysA); n != 0 {
		t.Fatalf("a vote was written while the author check was unknown: %d rows", n)
	}
}

func mustPoll(t *testing.T, f *writeFix, pollID int, session string) map[string]any {
	t.Helper()
	resp, poll := f.pollGet(t, pollID, session)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get poll %d: %d %+v", pollID, resp.StatusCode, poll)
	}
	return poll
}
