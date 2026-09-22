package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type PollRepository struct {
	db *gorm.DB
}

func NewPollRepository(db *gorm.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) DB() *gorm.DB {
	return r.db
}

func (r *PollRepository) FindByID(id int) (*model.TopicPoll, error) {
	var poll model.TopicPoll
	err := r.db.First(&poll, id).Error
	return &poll, err
}

func (r *PollRepository) CountByTopicID(topicID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicPoll{}).Where("topic_id = ?", topicID).Count(&count).Error
	return count, err
}

func (r *PollRepository) FindOptionsByPollID(pollID int) ([]model.TopicPollOption, error) {
	var options []model.TopicPollOption
	err := r.db.Where("poll_id = ?", pollID).Order("id ASC").Find(&options).Error
	return options, err
}

func (r *PollRepository) TouchTopicStatusUpdateTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).
		Where("id = ? AND created > ?", topicID, model.BumpCutoff(t)).
		Updates(map[string]any{"status_update_time": t}).Error
}

func (r *PollRepository) DeletePollCascade(tx *gorm.DB, pollID int) error {
	if err := tx.Where("poll_id = ?", pollID).Delete(&model.TopicPollVote{}).Error; err != nil {
		return err
	}
	if err := tx.Where("poll_id = ?", pollID).Delete(&model.TopicPollOption{}).Error; err != nil {
		return err
	}
	return tx.Delete(&model.TopicPoll{}, pollID).Error
}
