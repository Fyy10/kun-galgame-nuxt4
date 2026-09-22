package app

import (
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"

	"gorm.io/gorm"
)

var engageTestAward topicapiv1.AwardFunc

func (a *App) newTopicV1Interactions(reads *topicapiv1.Service) *topicapiv1.Interactions {
	var (
		notify msgService.Notifier
		db     *gorm.DB
	)
	if a != nil {
		db = a.DB
		notify = a.Notifier
		if notify == nil && db != nil {
			notify = msgService.NewNotifier(msgRepo.NewMessageRepository(db))
		}
	}
	award := moemoepoint.Award
	if engageTestAward != nil {
		award = engageTestAward
	}
	return topicapiv1.NewInteractions(reads, db, notify, award)
}
