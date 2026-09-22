package app

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/trust/gate"
)

const (
	w3PollA    = 930000401
	w3PollB    = 930000402
	w3OptA1    = 930000411
	w3OptA2    = 930000412
	w3OptB1    = 930000421
	w3Stranger = 930000900
)

func (f *writeFix) scalar(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *writeFix) seedPolls(t *testing.T) {
	t.Helper()
	at := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed polls: %v", err)
		}
	}
	run(`INSERT INTO topic_poll (id, title, description, type, min_choice, max_choice, status,
		result_visibility, is_anonymous, can_change_vote, topic_id, user_id, created, updated)
		VALUES (?, 'poll a', '', 'multiple', 1, 2, 'open', 'always', false, true, ?, ?, ?, ?)`,
		w3PollA, w3TopicPub, w3UserAlice, at, at)
	run(`INSERT INTO topic_poll (id, title, description, type, min_choice, max_choice, status,
		result_visibility, is_anonymous, can_change_vote, topic_id, user_id, created, updated)
		VALUES (?, 'poll b', '', 'single', 1, 1, 'open', 'always', false, true, ?, ?, ?, ?)`,
		w3PollB, w3TopicFloors, w3UserAlice, at, at)
	run(`INSERT INTO topic_poll_option (id, text, poll_id, vote_count, created, updated)
		VALUES (?, 'a1', ?, 0, ?, ?), (?, 'a2', ?, 0, ?, ?), (?, 'b1', ?, 0, ?, ?)`,
		w3OptA1, w3PollA, at, at, w3OptA2, w3PollA, at, at, w3OptB1, w3PollB, at, at)
}

// The vote path never tied option_id to poll_id, and there is no composite key.
func TestLegacyPollVoteRefusesAnotherPollsOption(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)
	f.seedPolls(t)

	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/poll/vote", w3TopicPub), "sess-bob",
		map[string]any{"poll_id": w3PollA, "option_id_array": []int{w3OptB1}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("a foreign option answered %d, want 400: %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, w3OptB1); n != 0 {
		t.Fatalf("foreign option vote_count %d, want 0", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll_vote WHERE poll_id = ?`, w3PollA); n != 0 {
		t.Fatalf("vote rows %d, want 0", n)
	}

	resp, raw = f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/poll/vote", w3TopicPub), "sess-bob",
		map[string]any{"poll_id": w3PollA, "option_id_array": []int{w3OptA1, w3OptB1}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("a mixed option set answered %d, want 400: %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, w3OptA1); n != 0 {
		t.Fatalf("own option moved on a rejected vote: %d", n)
	}
}

func TestLegacyPollVoteRefusesTheSameOptionTwice(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)
	f.seedPolls(t)

	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/poll/vote", w3TopicPub), "sess-bob",
		map[string]any{"poll_id": w3PollA, "option_id_array": []int{w3OptA1, w3OptA1}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("a duplicated option answered %d, want 400: %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, w3OptA1); n != 0 {
		t.Fatalf("vote_count %d, want 0", n)
	}
}

func TestLegacyPollVoteStillAcceptsItsOwnOptions(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)
	f.seedPolls(t)

	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/poll/vote", w3TopicPub), "sess-bob",
		map[string]any{"poll_id": w3PollA, "option_id_array": []int{w3OptA1, w3OptA2}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a valid vote was refused: %d %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, w3OptA1); n != 1 {
		t.Fatalf("a1 vote_count %d, want 1", n)
	}
	if n := f.scalar(t, `SELECT vote_count FROM topic_poll_option WHERE id = ?`, w3OptA2); n != 1 {
		t.Fatalf("a2 vote_count %d, want 1", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_poll_vote WHERE poll_id = ? AND user_id = ?`, w3PollA, w3UserBob); n != 2 {
		t.Fatalf("vote rows %d, want 2", n)
	}
}

// target_user_id came straight from the request body and nothing checked it.
func TestLegacyCommentDerivesItsTarget(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)

	// w3ReplyMin is Bob's reply on Alice's topic; Alice comments and names a stranger.
	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/comment", w3TopicFloors), "sess-alice",
		map[string]any{
			"topic_id": w3TopicFloors, "reply_id": w3ReplyMin,
			"target_user_id": w3Stranger, "content": "top level",
		})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("comment %d %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_comment WHERE target_user_id = ?`, w3Stranger); n != 0 {
		t.Fatalf("the named stranger was stored as the target %d times", n)
	}
	var parentID int
	if err := f.db.Raw(`SELECT id FROM topic_comment WHERE topic_reply_id = ? AND user_id = ?`,
		w3ReplyMin, w3UserAlice).Scan(&parentID).Error; err != nil {
		t.Fatal(err)
	}
	if got := f.scalar(t, `SELECT target_user_id FROM topic_comment WHERE id = ?`, parentID); got != w3UserBob {
		t.Fatalf("target %d, want the reply author %d", got, w3UserBob)
	}
	for _, a := range f.snapshotAwards() {
		if a.userID == w3Stranger {
			t.Fatalf("the stranger was paid: %+v", a)
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ?`, w3Stranger); n != 0 {
		t.Fatalf("the stranger was notified %d times", n)
	}

	// A child comment targets the parent comment's author, not whatever is sent.
	resp, raw = f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/comment", w3TopicFloors), "sess-other",
		map[string]any{
			"topic_id": w3TopicFloors, "reply_id": w3ReplyMin,
			"parent_comment_id": parentID,
			"target_user_id":    w3Stranger, "content": "child",
		})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("child comment %d %s", resp.StatusCode, raw)
	}
	if got := f.scalar(t,
		`SELECT target_user_id FROM topic_comment WHERE topic_reply_id = ? AND user_id = ?`,
		w3ReplyMin, w3UserOther); got != w3UserAlice {
		t.Fatalf("child target %d, want the parent author %d", got, w3UserAlice)
	}
}

func TestLegacyCommentRefusesAReplyOfAnotherTopic(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)

	// w3ReplyMin lives on w3TopicFloors; naming w3TopicPub used to be accepted.
	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/comment", w3TopicPub), "sess-bob",
		map[string]any{
			"topic_id": w3TopicPub, "reply_id": w3ReplyMin,
			"target_user_id": w3UserAlice, "content": "cross topic",
		})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-topic comment %d %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_comment WHERE topic_reply_id = ?`, w3ReplyMin); n != 0 {
		t.Fatalf("comment rows %d, want 0", n)
	}
	if got := f.scalar(t, `SELECT comment_count FROM topic WHERE id = ?`, w3TopicPub); got != 0 {
		t.Fatalf("comment_count of the named topic %d, want 0", got)
	}
}

func TestLegacyCommentRefusesADeletedReply(t *testing.T) {
	f := newWriteFix(t, gate.Checker(nil))
	f.alice(t)
	if err := f.db.Exec(`UPDATE topic_reply SET status = 1 WHERE id = ?`, w3ReplyMin+2).Error; err != nil {
		t.Fatal(err)
	}

	resp, raw := f.doLegacy(t, http.MethodPost,
		fmt.Sprintf("/api/topic/%d/comment", w3TopicFloors), "sess-bob",
		map[string]any{
			"topic_id": w3TopicFloors, "reply_id": w3ReplyMin + 2,
			"target_user_id": w3UserBob, "content": "on a deleted reply",
		})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("comment on a deleted reply %d %s", resp.StatusCode, raw)
	}
}
