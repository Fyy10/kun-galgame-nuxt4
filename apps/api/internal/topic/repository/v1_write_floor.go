package repository

import (
	"database/sql"
	"errors"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

func NextReplyFloor(tx *gorm.DB, topicID int) (int, error) {
	var floor int
	err := tx.Raw(
		`UPDATE topic SET last_reply_floor = GREATEST(last_reply_floor, (SELECT COALESCE(MAX(floor), 0) FROM topic_reply WHERE topic_id = ?)) + 1 WHERE id = ? RETURNING last_reply_floor`,
		topicID, topicID,
	).Row().Scan(&floor)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, gorm.ErrRecordNotFound
	}
	if err != nil {
		return 0, err
	}
	return floor, nil
}

// A comment read on its own does not know the floor of the reply it sits
// under, and the caller needs it to scroll a deep link to the right place.
// Reading it whole would drag the reply's body along for one integer.
func (r *ReplyRepository) FloorByID(replyID int) (int, error) {
	var floor int
	err := r.db.Model(&model.TopicReply{}).
		Where("id = ?", replyID).Limit(1).Pluck("floor", &floor).Error
	return floor, err
}
