package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func (f *writeFix) sqlVoteOrder(t *testing.T, pollID int) []string {
	t.Helper()
	var ids []int
	err := f.db.Raw(`
		SELECT id FROM topic_poll_vote
		WHERE poll_id = ? AND user_id <> ?
		ORDER BY created DESC, id DESC`, pollID, w3UserBanned).Scan(&ids).Error
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.Itoa(id)
	}
	return out
}

func (f *writeFix) walkVoteLog(t *testing.T, pollID, limit int, session string) []string {
	t.Helper()
	var got []string
	path := fmt.Sprintf("/api/v1/polls/%d/votes?limit=%d", pollID, limit)
	for page := range 50 {
		resp, body := f.voteList(t, path, session)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		if body["object"] != "list" {
			t.Fatalf("page %d object %v", page, body["object"])
		}
		if _, ok := body["total"]; ok {
			t.Fatalf("the vote log is a cursor collection and sends no total: %v", body)
		}
		items := pollItems(t, body)
		next, hasNext := body["next_cursor"].(string)
		if hasNext && len(items) != limit {
			t.Errorf("page %d holds %d items and a cursor, want %d", page, len(items), limit)
		}
		for _, row := range items {
			got = append(got, strID(row["id"]))
			if row["object"] != "poll_vote" || strID(row["poll_id"]) != strconv.Itoa(pollID) {
				t.Errorf("row %+v", row)
			}
		}
		if !hasNext {
			return got
		}
		path = fmt.Sprintf("/api/v1/polls/%d/votes?limit=%d&cursor=%s", pollID, limit, next)
	}
	t.Fatal("the vote log did not end within 50 pages")
	return nil
}

// The fixture puts several votes on the same instant across every page
// boundary: with one distinct second per voter this walk stays correct even
// after the id tie-breaker is deleted from the keyset.
func TestV1ListPollVotesTraversalParity(t *testing.T) {
	f := newPollFix(t, nil)
	want := f.sqlVoteOrder(t, w5bPollLog)
	if len(want) != 18 {
		t.Fatalf("the fixture must hold 18 renderable votes, got %d", len(want))
	}

	for _, limit := range []int{1, 2, 3, 5} {
		got := f.walkVoteLog(t, w5bPollLog, limit, "sess-alice")
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("limit %d walked\n%v\nwant\n%v", limit, got, want)
		}
		seen := map[string]bool{}
		for _, id := range got {
			if seen[id] {
				t.Fatalf("limit %d repeated %s", limit, id)
			}
			seen[id] = true
		}
	}
}

func TestV1ListPollVotesLeavesOutBannedVoters(t *testing.T) {
	f := newPollFix(t, nil)

	got := f.walkVoteLog(t, w5bPollLog, 20, "sess-alice")
	var banned int
	if err := f.db.Raw(`SELECT id FROM topic_poll_vote WHERE poll_id = ? AND user_id = ?`,
		w5bPollLog, w3UserBanned).Row().Scan(&banned); err != nil {
		t.Fatal(err)
	}
	for _, id := range got {
		if id == strconv.Itoa(banned) {
			t.Fatalf("the banned voter's row %d is in the log", banned)
		}
	}
}

// The legacy face answered 200 with an empty array whenever the caller could
// not see results, so "nobody voted" and "not for you" looked the same.
func TestV1ListPollVotesRefusesWhatItMayNotShow(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes", w5bPollAnon), "sess-alice")
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("an anonymous poll's log %d %+v", resp.StatusCode, out)
	}

	resp, out = f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes", w5bPollAfterVote), "sess-bob")
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("after_vote before voting %d %+v", resp.StatusCode, out)
	}
	if resp, out := f.voteSet(t, w5bPollAfterVote, "sess-bob", []string{strconv.Itoa(w5bOptAfterA)}); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, out)
	}
	resp, out = f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes", w5bPollAfterVote), "sess-bob")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("after_vote once voted %d %+v", resp.StatusCode, out)
	}
	if len(pollItems(t, out)) != 1 {
		t.Fatalf("items %v", out["items"])
	}
}

func TestV1ListPollVotesRejectsABadPage(t *testing.T) {
	f := newPollFix(t, nil)

	resp, out := f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes?limit=101", w5bPollLog), "sess-alice")
	if resp.StatusCode != http.StatusBadRequest || out["code"] != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit 101 %d %+v", resp.StatusCode, out)
	}
	resp, out = f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes?cursor=cur_nonsense", w5bPollLog), "sess-alice")
	if resp.StatusCode != http.StatusBadRequest || out["code"] != "INVALID_CURSOR" {
		t.Fatalf("garbage cursor %d %+v", resp.StatusCode, out)
	}
	// A cursor is bound to the collection that made it.
	_, body := f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes?limit=1", w5bPollLog), "sess-alice")
	cursor, _ := body["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("no cursor on a full page")
	}
	resp, out = f.voteList(t, fmt.Sprintf("/api/v1/polls/%d/votes?cursor=%s", w5bPollAfterVote, cursor), "sess-alice")
	if resp.StatusCode != http.StatusBadRequest || out["code"] != "INVALID_CURSOR" {
		t.Fatalf("another poll's cursor %d %+v", resp.StatusCode, out)
	}
}
