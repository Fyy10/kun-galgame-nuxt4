-- 102: drop topic_reply.comment_count (deploy-then-drop).
--
-- Migration 004 filled this column once, on 2026-06-05, and nothing in the
-- tree has written it since: of the 14,865 replies created after 004 and 020
-- ran, exactly one carries a non-zero value. By 2026-09-22 the snapshot had
-- rotted into 420 replies reading 0 that have comments and 18 whose number is
-- simply wrong. Nothing reads it either — the four services that map a
-- comment_count read it off `topic`, not off the reply.
--
-- DEPLOY ORDER MATTERS. ReplyRepository.CreateReply is tx.Create(reply), and
-- GORM builds the INSERT column list from the struct, so it names every
-- mapped column. Dropping this column while model.TopicReply still declares
-- CommentCount makes every reply insert fail. The field is removed in the
-- deploy that precedes this migration; run 102 only after that deploy is
-- live, with --only=102.

-- AND RESTART THE API IMMEDIATELY AFTER. Dropping a column changes the
-- result type of every prepared statement that selected it, and pgx v5 caches
-- prepared statements per connection, so live connections answer
-- "cached plan must not change result type (SQLSTATE 0A000)" until they are
-- recycled. Running this on 2026-09-22 at 15:38:22 UTC put six reply-list
-- requests into 500 from reply_keyset.go:40 over the next 57 seconds; a
-- `docker compose restart kungal-api` ended it at once. ConnMaxLifetime would
-- have cleared it eventually, which is not the same as not breaking.

ALTER TABLE topic_reply DROP COLUMN IF EXISTS comment_count;
