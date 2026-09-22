package repository

import (
	"time"

	"gorm.io/gorm"
)

// topic_comment.updated and topic_comment_like.updated are NOT NULL with no
// default: leaving either out of a raw INSERT is what made every v1 favorite
// and upvote answer 500 with SQLSTATE 23502 in the wave before this one.
func (r *CommentRepository) InsertComment(tx *gorm.DB, replyID, topicID, userID, inReplyToUserID int, parentID *int, body string) (int, time.Time, error) {
	var (
		id      int
		created time.Time
	)
	row := tx.Raw(`
		INSERT INTO topic_comment
			(content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, now(), now())
		RETURNING id, created`, body, topicID, replyID, userID, inReplyToUserID, parentID).Row()
	if err := row.Err(); err != nil {
		return 0, time.Time{}, err
	}
	if err := row.Scan(&id, &created); err != nil {
		return 0, time.Time{}, err
	}
	return id, created, nil
}

func (r *CommentRepository) UpdateCommentBody(tx *gorm.DB, commentID int, body string, edited time.Time) error {
	return tx.Exec(`
		UPDATE topic_comment SET content = ?, edited = ?, updated = now() WHERE id = ?`,
		body, edited, commentID).Error
}

func (r *CommentRepository) DeleteComment(tx *gorm.DB, commentID int) error {
	return tx.Exec(`DELETE FROM topic_comment WHERE id = ?`, commentID).Error
}

func (r *CommentRepository) InsertCommentLike(tx *gorm.DB, commentID, userID int) (int, bool, error) {
	return returningID(tx, `
		INSERT INTO topic_comment_like (topic_comment_id, user_id, created, updated)
		VALUES (?, ?, now(), now())
		ON CONFLICT (topic_comment_id, user_id) DO NOTHING
		RETURNING id`, commentID, userID)
}

func (r *CommentRepository) DeleteCommentLikeRow(tx *gorm.DB, commentID, userID int) (int, bool, error) {
	return returningID(tx, `
		DELETE FROM topic_comment_like
		WHERE topic_comment_id = ? AND user_id = ?
		RETURNING id`, commentID, userID)
}
