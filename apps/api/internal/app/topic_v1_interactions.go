package app

import topicapiv1 "kun-galgame-api/internal/topic/apiv1"

func (a *App) newTopicV1Interactions(reads *topicapiv1.Service) *topicapiv1.Interactions {
	return topicapiv1.NewInteractions(reads)
}
