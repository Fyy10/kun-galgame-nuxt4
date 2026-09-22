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

ALTER TABLE topic_reply DROP COLUMN IF EXISTS comment_count;
