package repository

import "kun-galgame-api/internal/topic/model"

func (r *TopicRepository) FindByIDs(ids []int) ([]model.Topic, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.Topic
	err := r.db.Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *TopicRepository) FindAccessGrantsForTopics(ids []int) (map[int][]model.TopicAccessGrant, error) {
	out := map[int][]model.TopicAccessGrant{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.TopicAccessGrant
	if err := r.db.Where("topic_id IN ?", ids).
		Order("topic_id, subject_type, subject_value").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TopicID] = append(out[row.TopicID], row)
	}
	return out, nil
}

func (r *TopicRepository) UserFavoritedTopicIDs(userID int, ids []int) (map[int]bool, error) {
	out := map[int]bool{}
	if userID <= 0 || len(ids) == 0 {
		return out, nil
	}
	var found []int
	if err := r.db.Model(&model.TopicFavorite{}).
		Where("user_id = ? AND topic_id IN ?", userID, ids).
		Pluck("topic_id", &found).Error; err != nil {
		return nil, err
	}
	for _, id := range found {
		out[id] = true
	}
	return out, nil
}

// Ordered by id so the tokens come back in the order the caller reacted,
// which is the order every other reaction face in v1 uses.
func (r *TopicRepository) UserTopicReactions(userID int, ids []int) (map[int][]string, error) {
	out := map[int][]string{}
	if userID <= 0 || len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		TopicID  int    `gorm:"column:topic_id"`
		Reaction string `gorm:"column:reaction"`
	}
	if err := r.db.Model(&model.TopicReaction{}).
		Select("topic_id, reaction").
		Where("user_id = ? AND topic_id IN ?", userID, ids).
		Order("id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TopicID] = append(out[row.TopicID], row.Reaction)
	}
	return out, nil
}
