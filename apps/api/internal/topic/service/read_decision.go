package service

import (
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/access"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
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

func topicDetailGrants(topic *topicModel.Topic, user *middleware.UserInfo, grants []topicModel.TopicAccessGrant) *dto.TopicAccessGrants {
	if user == nil || (user.ID != topic.UserID && !user.Can(perm.TopicEditAny)) {
		return nil
	}
	out := &dto.TopicAccessGrants{Roles: []string{}, UserIDs: []int{}}
	for _, grant := range grants {
		switch grant.SubjectType {
		case "role":
			out.Roles = append(out.Roles, grant.SubjectValue)
		case "user":
			if id, err := strconv.Atoi(grant.SubjectValue); err == nil {
				out.UserIDs = append(out.UserIDs, id)
			}
		}
	}
	return out
}
