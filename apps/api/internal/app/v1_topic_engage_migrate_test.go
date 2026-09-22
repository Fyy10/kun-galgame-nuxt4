package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func migration099Path(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations", "099_topic_engagement_recount.up.sql")
}

func splitSQLStatements(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ";") {
		var kept []string
		for _, ln := range strings.Split(part, "\n") {
			trim := strings.TrimSpace(ln)
			if trim == "" || strings.HasPrefix(trim, "--") {
				continue
			}
			kept = append(kept, ln)
		}
		s := strings.TrimSpace(strings.Join(kept, "\n"))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func TestV1TopicEngagementRecountMigration099(t *testing.T) {
	f := newEngageFix(t)
	base := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

	f.runSQL(t, `INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES
		(?, ?, 'like', ?), (?, ?, 'like', ?), (?, ?, 'dislike', ?)`,
		e1TopicMigrateA, e1UserBob, base,
		e1TopicMigrateA, e1UserCarol, base,
		e1TopicMigrateA, e1UserDave, base)
	f.runSQL(t, `INSERT INTO topic_favorite (topic_id, user_id, created, updated) VALUES (?, ?, ?, ?)`,
		e1TopicMigrateA, e1UserBob, base, base)
	f.runSQL(t, `INSERT INTO topic_upvote (topic_id, user_id, description, created, updated) VALUES
		(?, ?, '', ?, ?), (?, ?, '', ?, ?)`,
		e1TopicMigrateA, e1UserBob, base, base,
		e1TopicMigrateA, e1UserCarol, base, base)
	f.runSQL(t, `INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction, created) VALUES
		(?, ?, 'like', ?), (?, ?, 'dislike', ?)`,
		e1ReplyMigrate, e1UserBob, base, e1ReplyMigrate, e1UserCarol, base)
	f.runSQL(t, `INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, 'like', ?)`,
		e1TopicMigrateB, e1UserBob, base)

	f.runSQL(t, `UPDATE topic SET like_count = 900, dislike_count = 900, favorite_count = 900, upvote_count = 900, updated = ?
		WHERE id = ?`, base, e1TopicMigrateA)
	f.runSQL(t, `UPDATE topic SET like_count = 1, dislike_count = 0, favorite_count = 0, upvote_count = 0, updated = ?
		WHERE id = ?`, base, e1TopicMigrateB)
	f.runSQL(t, `UPDATE topic_reply SET like_count = 50, dislike_count = 50, updated = ? WHERE id = ?`,
		base, e1ReplyMigrate)

	xminA := f.xmin(t, "topic", e1TopicMigrateA)
	xminB := f.xmin(t, "topic", e1TopicMigrateB)
	xminR := f.xmin(t, "topic_reply", e1ReplyMigrate)
	updatedB := f.topicTime(t, e1TopicMigrateB, "updated")

	raw, err := os.ReadFile(migration099Path(t))
	if err != nil {
		t.Fatal(err)
	}
	stmts := splitSQLStatements(string(raw))
	if len(stmts) != 9 {
		t.Fatalf("099 statements %d, want 6 updates + 3 indexes", len(stmts))
	}
	for _, stmt := range stmts {
		f.runSQL(t, stmt)
	}

	if f.topicInt(t, e1TopicMigrateA, "like_count") != 2 {
		t.Errorf("A like_count %d want 2", f.topicInt(t, e1TopicMigrateA, "like_count"))
	}
	if f.topicInt(t, e1TopicMigrateA, "dislike_count") != 1 {
		t.Errorf("A dislike_count %d want 1", f.topicInt(t, e1TopicMigrateA, "dislike_count"))
	}
	if f.topicInt(t, e1TopicMigrateA, "favorite_count") != 1 {
		t.Errorf("A favorite_count %d want 1", f.topicInt(t, e1TopicMigrateA, "favorite_count"))
	}
	if f.topicInt(t, e1TopicMigrateA, "upvote_count") != 2 {
		t.Errorf("A upvote_count %d want 2", f.topicInt(t, e1TopicMigrateA, "upvote_count"))
	}
	if f.replyInt(t, e1ReplyMigrate, "like_count") != 1 {
		t.Errorf("reply like_count %d want 1", f.replyInt(t, e1ReplyMigrate, "like_count"))
	}
	if f.replyInt(t, e1ReplyMigrate, "dislike_count") != 1 {
		t.Errorf("reply dislike_count %d want 1", f.replyInt(t, e1ReplyMigrate, "dislike_count"))
	}
	if f.topicInt(t, e1TopicMigrateB, "like_count") != 1 ||
		f.topicInt(t, e1TopicMigrateB, "dislike_count") != 0 ||
		f.topicInt(t, e1TopicMigrateB, "favorite_count") != 0 ||
		f.topicInt(t, e1TopicMigrateB, "upvote_count") != 0 {
		t.Fatalf("B counters drifted like=%d dislike=%d fav=%d up=%d",
			f.topicInt(t, e1TopicMigrateB, "like_count"),
			f.topicInt(t, e1TopicMigrateB, "dislike_count"),
			f.topicInt(t, e1TopicMigrateB, "favorite_count"),
			f.topicInt(t, e1TopicMigrateB, "upvote_count"))
	}
	if f.xmin(t, "topic", e1TopicMigrateB) != xminB {
		t.Fatal("correct topic row was rewritten")
	}
	if !f.topicTime(t, e1TopicMigrateB, "updated").Equal(updatedB) {
		t.Fatal("correct topic updated timestamp changed")
	}
	if f.xmin(t, "topic", e1TopicMigrateA) == xminA {
		t.Fatal("drifted topic row was left alone")
	}
	if f.xmin(t, "topic_reply", e1ReplyMigrate) == xminR {
		t.Fatal("drifted reply row was left alone")
	}

	for _, name := range []string{
		"idx_topic_upvote_topic_created_id",
		"idx_topic_reaction_topic_created_id",
		"idx_topic_reply_reaction_reply_created_id",
	} {
		if f.count(t, `SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?`, name) != 1 {
			t.Errorf("missing index %s", name)
		}
	}
}
