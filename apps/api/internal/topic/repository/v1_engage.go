package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func returningID(tx *gorm.DB, q string, args ...any) (int, bool, error) {
	rows, err := tx.Raw(q, args...).Rows()
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, false, rows.Err()
	}
	var id int
	if err := rows.Scan(&id); err != nil {
		return 0, false, err
	}
	return id, true, rows.Err()
}

func (r *TopicRepository) InsertTopicReaction(tx *gorm.DB, topicID, userID int, reaction string) (int, bool, error) {
	return returningID(tx, `
		INSERT INTO topic_reaction (topic_id, user_id, reaction)
		VALUES (?, ?, ?)
		ON CONFLICT (topic_id, user_id, reaction) DO NOTHING
		RETURNING id`, topicID, userID, reaction)
}

func (r *TopicRepository) DeleteTopicReaction(tx *gorm.DB, topicID, userID int, reaction string) (int, bool, error) {
	return returningID(tx, `
		DELETE FROM topic_reaction
		WHERE topic_id = ? AND user_id = ? AND reaction = ?
		RETURNING id`, topicID, userID, reaction)
}

func (r *TopicRepository) InsertTopicFavorite(tx *gorm.DB, topicID, userID int) (int, bool, error) {
	return returningID(tx, `
		INSERT INTO topic_favorite (topic_id, user_id)
		VALUES (?, ?)
		ON CONFLICT (topic_id, user_id) DO NOTHING
		RETURNING id`, topicID, userID)
}

func (r *TopicRepository) DeleteTopicFavoriteRow(tx *gorm.DB, topicID, userID int) (int, bool, error) {
	return returningID(tx, `
		DELETE FROM topic_favorite
		WHERE topic_id = ? AND user_id = ?
		RETURNING id`, topicID, userID)
}

func (r *TopicRepository) InsertTopicUpvote(tx *gorm.DB, topicID, userID int, note string) (int, time.Time, error) {
	rows, err := tx.Raw(`
		INSERT INTO topic_upvote (topic_id, user_id, description)
		VALUES (?, ?, ?)
		RETURNING id, created`, topicID, userID, note).Rows()
	if err != nil {
		return 0, time.Time{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		err := rows.Err()
		if err == nil {
			err = gorm.ErrRecordNotFound
		}
		return 0, time.Time{}, err
	}
	var id int
	var created time.Time
	if err := rows.Scan(&id, &created); err != nil {
		return 0, time.Time{}, err
	}
	return id, created, rows.Err()
}

func (r *TopicRepository) AdjustTopicCount(tx *gorm.DB, topicID int, column string, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE topic SET `+column+` = `+column+` + ? WHERE id = ?`, delta, topicID).Error
	}
	return tx.Exec(`UPDATE topic SET `+column+` = GREATEST(`+column+` + ?, 0) WHERE id = ?`, delta, topicID).Error
}

func (r *TopicRepository) LockMoemoepoint(tx *gorm.DB, userID int) (int, error) {
	var row struct {
		Moemoepoint int `gorm:"column:moemoepoint"`
	}
	err := tx.Table("kungal_user_state").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("moemoepoint").
		Where("user_id = ?", userID).
		Take(&row).Error
	return row.Moemoepoint, err
}

func (r *TopicRepository) SetBestAnswerID(tx *gorm.DB, topicID int, replyID *int, bump bool, now time.Time) error {
	if replyID == nil {
		return tx.Exec(`UPDATE topic SET best_answer_id = NULL WHERE id = ?`, topicID).Error
	}
	if !bump {
		return tx.Exec(`UPDATE topic SET best_answer_id = ? WHERE id = ?`, *replyID, topicID).Error
	}
	return tx.Exec(`
		UPDATE topic SET
			best_answer_id = ?,
			status_update_time = CASE WHEN created > ? THEN ? ELSE status_update_time END
		WHERE id = ?`, *replyID, model.BumpCutoff(now), now, topicID).Error
}

func (r *TopicRepository) SetPinnedReplyID(tx *gorm.DB, topicID int, replyID *int) error {
	if replyID == nil {
		return tx.Exec(`UPDATE topic SET pinned_reply_id = NULL WHERE id = ?`, topicID).Error
	}
	return tx.Exec(`UPDATE topic SET pinned_reply_id = ? WHERE id = ?`, *replyID, topicID).Error
}

func (r *ReplyRepository) InsertReplyReaction(tx *gorm.DB, replyID, userID int, reaction string) (int, bool, error) {
	return returningID(tx, `
		INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction)
		VALUES (?, ?, ?)
		ON CONFLICT (topic_reply_id, user_id, reaction) DO NOTHING
		RETURNING id`, replyID, userID, reaction)
}

func (r *ReplyRepository) DeleteReplyReaction(tx *gorm.DB, replyID, userID int, reaction string) (int, bool, error) {
	return returningID(tx, `
		DELETE FROM topic_reply_reaction
		WHERE topic_reply_id = ? AND user_id = ? AND reaction = ?
		RETURNING id`, replyID, userID, reaction)
}

func (r *ReplyRepository) AdjustReplyCount(tx *gorm.DB, replyID int, column string, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE topic_reply SET `+column+` = `+column+` + ? WHERE id = ?`, delta, replyID).Error
	}
	return tx.Exec(`UPDATE topic_reply SET `+column+` = GREATEST(`+column+` + ?, 0) WHERE id = ?`, delta, replyID).Error
}
