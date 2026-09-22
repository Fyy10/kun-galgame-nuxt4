package service

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/access"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"
)

func requireTopicRead(repo *repository.TopicRepository, topic *topicModel.Topic, user *middleware.UserInfo) ([]topicModel.TopicAccessGrant, *errors.AppError) {
	viewer := access.Snapshot(user)
	if topic.Status == 1 && !(viewer.Authenticated && (viewer.ID == topic.UserID || viewer.ViewHidden)) {
		return nil, errors.ErrNotFound("未找到该话题")
	}
	var grants []topicModel.TopicAccessGrant
	if access.NeedsGrants(topic) {
		var err error
		grants, err = repo.FindAccessGrants(topic.ID)
		if err != nil {
			return nil, errors.ErrInternal("获取话题权限失败")
		}
	}
	if !access.Allowed(topic, viewer, grants) {
		return nil, errors.ErrNotFound("未找到该话题")
	}
	return grants, nil
}
