package repository

import (
	"kun-galgame-api/internal/topic/model"
)

func (r *TopicRepository) GetUserTopicReactions(topicID, userID int) ([]string, error) {
	if userID <= 0 {
		return nil, nil
	}
	var keys []string
	err := r.db.Model(&model.TopicReaction{}).
		Where("topic_id = ? AND user_id = ?", topicID, userID).Pluck("reaction", &keys).Error
	return keys, err
}

func (r *ReplyRepository) GetUserRepliesReactions(replyIDs []int, userID int) (map[int][]string, error) {
	out := map[int][]string{}
	if userID <= 0 || len(replyIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		TopicReplyID int    `gorm:"column:topic_reply_id"`
		Reaction     string `gorm:"column:reaction"`
	}
	if err := r.db.Table("topic_reply_reaction").
		Select("topic_reply_id, reaction").
		Where("topic_reply_id IN ? AND user_id = ?", replyIDs, userID).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TopicReplyID] = append(out[row.TopicReplyID], row.Reaction)
	}
	return out, nil
}
