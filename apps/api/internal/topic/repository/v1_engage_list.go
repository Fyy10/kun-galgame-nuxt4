package repository

import "time"

type EngageHistoryRow struct {
	ID          int       `gorm:"column:id"`
	UserID      int       `gorm:"column:user_id"`
	Reaction    string    `gorm:"column:reaction"`
	Description string    `gorm:"column:description"`
	Created     time.Time `gorm:"column:created"`
}

type EngageHistoryPos struct {
	Created time.Time
	ID      int
}

func (r *TopicRepository) ListTopicReactionsKeyset(topicID, limit int, pos *EngageHistoryPos) ([]EngageHistoryRow, error) {
	q := r.db.Table("topic_reaction").
		Select("id, user_id, reaction, created").
		Where("topic_id = ?", topicID)
	if pos != nil {
		q = q.Where("(created, id) < (?, ?)", pos.Created, pos.ID)
	}
	var rows []EngageHistoryRow
	err := q.Order("created DESC, id DESC").Limit(limit + 1).Scan(&rows).Error
	return rows, err
}

func (r *TopicRepository) ListTopicUpvotesKeyset(topicID, limit int, pos *EngageHistoryPos) ([]EngageHistoryRow, error) {
	q := r.db.Table("topic_upvote").
		Select("id, user_id, description, created").
		Where("topic_id = ?", topicID)
	if pos != nil {
		q = q.Where("(created, id) < (?, ?)", pos.Created, pos.ID)
	}
	var rows []EngageHistoryRow
	err := q.Order("created DESC, id DESC").Limit(limit + 1).Scan(&rows).Error
	return rows, err
}

func (r *ReplyRepository) ListReplyReactionsKeyset(replyID, limit int, pos *EngageHistoryPos) ([]EngageHistoryRow, error) {
	q := r.db.Table("topic_reply_reaction").
		Select("id, user_id, reaction, created").
		Where("topic_reply_id = ?", replyID)
	if pos != nil {
		q = q.Where("(created, id) < (?, ?)", pos.Created, pos.ID)
	}
	var rows []EngageHistoryRow
	err := q.Order("created DESC, id DESC").Limit(limit + 1).Scan(&rows).Error
	return rows, err
}
