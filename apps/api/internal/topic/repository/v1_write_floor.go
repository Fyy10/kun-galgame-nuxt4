package repository

import (
	"database/sql"
	"errors"

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
