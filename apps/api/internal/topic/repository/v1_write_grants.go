package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

func ListAccessGrants(db *gorm.DB, topicID int) ([]model.TopicAccessGrant, error) {
	var rows []model.TopicAccessGrant
	err := db.Raw(
		`SELECT topic_id, subject_type, subject_value FROM topic_access_grant WHERE topic_id = ? ORDER BY subject_type, CASE WHEN subject_type = 'user' THEN LPAD(subject_value, 20, '0') ELSE subject_value END`,
		topicID,
	).Scan(&rows).Error
	return rows, err
}
