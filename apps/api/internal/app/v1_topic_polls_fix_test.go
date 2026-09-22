package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/trust/gate"
)

const (
	w5bPollAlways    = 930000601
	w5bPollSingle    = 930000602
	w5bPollAfterVote = 930000603
	w5bPollAnon      = 930000604
	w5bPollClosed    = 930000605
	w5bPollRole      = 930000606
	w5bPollBanned    = 930000607
	w5bPollLog       = 930000608

	w5bOptAlwaysA = 930000611
	w5bOptAlwaysB = 930000612
	w5bOptAlwaysC = 930000613
	w5bOptSingleA = 930000621
	w5bOptSingleB = 930000622
	w5bOptAfterA  = 930000631
	w5bOptAfterB  = 930000632
	w5bOptAnonA   = 930000641
	w5bOptAnonB   = 930000642
	w5bOptClosedA = 930000651
	w5bOptClosedB = 930000652
	w5bOptRoleA   = 930000661
	w5bOptRoleB   = 930000662
	w5bOptBanA    = 930000671
	w5bOptBanB    = 930000672
	w5bOptLogA    = 930000681
	w5bOptLogB    = 930000682

	w5bVoteMin = 930000801
)

type pollSeed struct {
	id         int
	topic      int
	author     int
	title      string
	choiceType string
	minChoice  int
	maxChoice  int
	visibility string
	anonymous  bool
	changeable bool
	deadline   *time.Time
	options    []int
}

// The vote log's keyset needs rows that share an instant and cross a page
// boundary: with a distinct second per voter the walk stays correct even after
// the id tie-breaker is deleted.
type voteSeed struct {
	id       int
	poll     int
	option   int
	user     int
	atSecond int
}

func newPollFix(t *testing.T, checker gate.Checker) *writeFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.seedPollDomain(t)
	return f
}

func (f *writeFix) seedPollDomain(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	past := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed polls: %v\n%s", err, q)
		}
	}
	seeds := []pollSeed{
		{w5bPollAlways, w3TopicPub, w3UserAlice, "always", "multiple", 1, 2, "always", false, true, nil,
			[]int{w5bOptAlwaysA, w5bOptAlwaysB, w5bOptAlwaysC}},
		{w5bPollSingle, w3TopicPub, w3UserAlice, "single", "single", 1, 1, "always", false, false, nil,
			[]int{w5bOptSingleA, w5bOptSingleB}},
		{w5bPollAfterVote, w3TopicPub, w3UserAlice, "after vote", "multiple", 1, 2, "after_vote", false, true, nil,
			[]int{w5bOptAfterA, w5bOptAfterB}},
		{w5bPollAnon, w3TopicPub, w3UserAlice, "anonymous", "single", 1, 1, "always", true, true, nil,
			[]int{w5bOptAnonA, w5bOptAnonB}},
		{w5bPollClosed, w3TopicPub, w3UserAlice, "closed", "single", 1, 1, "always", false, true, &past,
			[]int{w5bOptClosedA, w5bOptClosedB}},
		{w5bPollRole, w3TopicRole, w3UserAlice, "role", "single", 1, 1, "always", false, true, nil,
			[]int{w5bOptRoleA, w5bOptRoleB}},
		{w5bPollBanned, w3TopicPub, w3UserBanned, "banned", "single", 1, 1, "always", false, true, nil,
			[]int{w5bOptBanA, w5bOptBanB}},
		{w5bPollLog, w3TopicPub, w3UserAlice, "log", "multiple", 1, 3, "always", false, true, nil,
			[]int{w5bOptLogA, w5bOptLogB}},
	}
	for i, s := range seeds {
		at := base.Add(time.Duration(i) * time.Minute)
		run(`INSERT INTO topic_poll (id, title, description, type, min_choice, max_choice, deadline, status,
			result_visibility, is_anonymous, can_change_vote, topic_id, user_id, created, updated)
			VALUES (?, ?, '', ?, ?, ?, ?, 'open', ?, ?, ?, ?, ?, ?, ?)`,
			s.id, s.title, s.choiceType, s.minChoice, s.maxChoice, s.deadline,
			s.visibility, s.anonymous, s.changeable, s.topic, s.author, at, at)
		for j, opt := range s.options {
			run(`INSERT INTO topic_poll_option (id, text, poll_id, vote_count, created, updated)
				VALUES (?, ?, ?, 0, ?, ?)`, opt, fmt.Sprintf("o%d", j+1), s.id, at, at)
		}
	}
	f.seedVoteLog(t, base)
}

// Twenty votes over twelve voters, in seven groups that share an instant, so a
// walk at limit 2 or 3 crosses a tie on every page boundary.
func (f *writeFix) seedVoteLog(t *testing.T, base time.Time) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed votes: %v\n%s", err, q)
		}
	}
	var seeds []voteSeed
	id := w5bVoteMin
	second := 0
	for i := range 12 {
		user := w3MentionMin + i
		seeds = append(seeds, voteSeed{id, w5bPollLog, w5bOptLogA, user, second})
		id++
		if i%2 == 1 {
			seeds = append(seeds, voteSeed{id, w5bPollLog, w5bOptLogB, user, second})
			id++
		}
		if i%2 == 1 {
			second++
		}
	}
	// A banned voter's rows are skipped by the face but still sit between the
	// others in the table, so the page-filling loop has to read past them.
	seeds = append(seeds, voteSeed{id, w5bPollLog, w5bOptLogA, w3UserBanned, 2})

	countA, countB := 0, 0
	for _, s := range seeds {
		at := base.Add(time.Duration(s.atSecond) * time.Second)
		run(`INSERT INTO topic_poll_vote (id, poll_id, option_id, user_id, created, updated)
			VALUES (?, ?, ?, ?, ?, ?)`, s.id, s.poll, s.option, s.user, at, at)
		if s.option == w5bOptLogA {
			countA++
		} else {
			countB++
		}
	}
	run(`UPDATE topic_poll_option SET vote_count = ? WHERE id = ?`, countA, w5bOptLogA)
	run(`UPDATE topic_poll_option SET vote_count = ? WHERE id = ?`, countB, w5bOptLogB)
}

func (f *writeFix) pollGet(t *testing.T, pollID int, session string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/polls/"+strconv.Itoa(pollID), session,
		"/polls/{poll_id}", "", nil, nil)
	return resp, problemMap(t, raw)
}

func (f *writeFix) pollList(t *testing.T, topicID int, session string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/polls", topicID), session,
		"/topics/{topic_id}/polls", "", nil, nil)
	return resp, problemMap(t, raw)
}

func (f *writeFix) pollCreate(t *testing.T, topicID int, session, idem string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/topics/%d/polls", topicID), session,
		"/topics/{topic_id}/polls", idem, nil, payload)
	return resp, problemMap(t, raw)
}

func (f *writeFix) pollPatch(t *testing.T, pollID int, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPatch, "/api/v1/polls/"+strconv.Itoa(pollID), session,
		"/polls/{poll_id}", "", nil, payload)
	return resp, problemMap(t, raw)
}

func (f *writeFix) pollDelete(t *testing.T, pollID int, session string) *http.Response {
	t.Helper()
	resp, _ := f.doJSON(t, http.MethodDelete, "/api/v1/polls/"+strconv.Itoa(pollID), session,
		"/polls/{poll_id}", "", nil, nil)
	return resp
}

func (f *writeFix) voteSet(t *testing.T, pollID int, session string, optionIDs []string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/polls/%d/vote", pollID), session,
		"/polls/{poll_id}/vote", "", nil, map[string]any{"option_ids": optionIDs})
	return resp, problemMap(t, raw)
}

func (f *writeFix) voteClear(t *testing.T, pollID int, session string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/polls/%d/vote", pollID), session,
		"/polls/{poll_id}/vote", "", nil, nil)
	return resp, problemMap(t, raw)
}

func (f *writeFix) voteList(t *testing.T, rawURL, session string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodGet, rawURL, session, "/polls/{poll_id}/votes", "", nil, nil)
	return resp, problemMap(t, raw)
}

func pollOptionIDs(t *testing.T, poll map[string]any) []string {
	t.Helper()
	raw, _ := poll["options"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		opt, _ := item.(map[string]any)
		out = append(out, strID(opt["id"]))
	}
	return out
}

func pollResultsOf(t *testing.T, poll map[string]any) map[string]any {
	t.Helper()
	results, ok := poll["results"].(map[string]any)
	if !ok {
		t.Fatalf("results %v, want an object", poll["results"])
	}
	return results
}

func pollVoteCounts(t *testing.T, poll map[string]any) map[string]int {
	t.Helper()
	out := map[string]int{}
	raw, _ := pollResultsOf(t, poll)["options"].([]any)
	for _, item := range raw {
		row, _ := item.(map[string]any)
		out[strID(row["option_id"])] = asInt(row["vote_count"])
	}
	return out
}

func (f *writeFix) storedVoteCount(t *testing.T, optionID int) int {
	t.Helper()
	return f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, optionID)
}

func (f *writeFix) storedVoteRows(t *testing.T, pollID, optionID int) int {
	t.Helper()
	return f.scalar(t, `SELECT COUNT(*) FROM topic_poll_vote WHERE poll_id = ? AND option_id = ?`, pollID, optionID)
}
