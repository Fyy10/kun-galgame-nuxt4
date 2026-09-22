package app

import (
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/trust/gate"
)

func (a *App) newTopicV1Polls(reads *topicapiv1.Service) *topicapiv1.Polls {
	check := a.TrustCheck
	scan := a.TrustScan
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	var polls *topicRepo.PollRepository
	if a.DB != nil {
		polls = topicRepo.NewPollRepository(a.DB)
	}
	return topicapiv1.NewPolls(reads, polls, check, scan)
}
