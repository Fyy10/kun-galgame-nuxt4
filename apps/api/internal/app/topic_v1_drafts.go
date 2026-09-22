package app

import (
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
)

func (a *App) newTopicV1Drafts() *topicapiv1.Drafts {
	var drafts *topicRepo.TopicDraftRepository
	if a.DB != nil {
		drafts = topicRepo.NewTopicDraftRepository(a.DB)
	}
	return topicapiv1.NewDrafts(drafts)
}
