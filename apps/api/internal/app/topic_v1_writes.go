package app

import topicapiv1 "kun-galgame-api/internal/topic/apiv1"

func (a *App) newTopicV1Writes(reads *topicapiv1.Service) *topicapiv1.Writes {
	return topicapiv1.NewWrites(reads)
}
