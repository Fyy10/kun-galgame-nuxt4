package repository

import (
	"time"

	"gorm.io/gorm"
)

const reactionSampleCap = 10

type ReactionSample struct {
	OwnerID  int       `gorm:"column:owner_id"`
	Reaction string    `gorm:"column:reaction"`
	UserID   int       `gorm:"column:user_id"`
	Count    int       `gorm:"column:cnt"`
	FirstAt  time.Time `gorm:"column:first_at"`
}

func (r *TopicRepository) SampleTopicReactions(topicIDs []int) ([]ReactionSample, error) {
	return sampleReactions(r.db, "topic_reaction", "topic_id", topicIDs)
}

func (r *ReplyRepository) SampleReplyReactions(replyIDs []int) ([]ReactionSample, error) {
	return sampleReactions(r.db, "topic_reply_reaction", "topic_reply_id", replyIDs)
}

func sampleReactions(db *gorm.DB, table, ownerCol string, ownerIDs []int) ([]ReactionSample, error) {
	out := []ReactionSample{}
	if len(ownerIDs) == 0 {
		return out, nil
	}
	err := db.Raw(`
		SELECT owner_id, reaction, user_id, cnt, first_at FROM (
			SELECT `+ownerCol+` AS owner_id, reaction, user_id,
				row_number() OVER (PARTITION BY `+ownerCol+`, reaction ORDER BY created, user_id) AS rn,
				count(*) OVER (PARTITION BY `+ownerCol+`, reaction) AS cnt,
				min(created) OVER (PARTITION BY `+ownerCol+`, reaction) AS first_at
			FROM `+table+` WHERE `+ownerCol+` IN ?
		) t WHERE rn <= ? ORDER BY owner_id, first_at, reaction, rn`,
		ownerIDs, reactionSampleCap).Scan(&out).Error
	return out, err
}
