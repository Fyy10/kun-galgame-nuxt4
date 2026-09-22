package app

import (
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"

	"gorm.io/gorm"
)

func (a *App) newTopicV1Interactions(reads *topicapiv1.Service) *topicapiv1.Interactions {
	var (
		notify msgService.Notifier
		db     *gorm.DB
		award  topicapiv1.AwardFunc
	)
	if a != nil {
		db = a.DB
		notify = a.Notifier
		award = a.TopicAward
		if notify == nil && db != nil {
			notify = msgService.NewNotifier(msgRepo.NewMessageRepository(db))
		}
	}
	if award == nil {
		award = moemoepoint.Award
	}
	return topicapiv1.NewInteractions(reads, db, notify, award)
}
