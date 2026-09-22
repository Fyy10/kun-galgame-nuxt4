package service

import (
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/repository"
)

func toTopicCard(r repository.TopicCardRow, sections []string, miniApps []string) dto.TopicCard {
	if sections == nil {
		sections = []string{}
	}
	covers := []string(r.CoverImages)
	if covers == nil {
		covers = []string{}
	}
	return dto.TopicCard{
		ID:             r.ID,
		Title:          r.Title,
		View:           r.View,
		Sections:       sections,
		CoverImages:    covers,
		CoverImageMeta: markdown.ResolveContentImageMeta(covers),
		User: dto.KunUser{
			ID:     r.UserID,
			Name:   r.UserName,
			Avatar: r.UserAvatar,
		},
		Status:           r.Status,
		HasBestAnswer:    r.BestAnswerID != nil,
		MiniApps:         miniApps,
		IsNSFW:           r.IsNSFW,
		LikeCount:        r.LikeCount,
		ReplyCount:       r.ReplyCount,
		CommentCount:     r.CommentCount,
		StatusUpdateTime: r.StatusUpdateTime,
		Created:          r.Created,
		UpvoteTime:       r.UpvoteTime,
	}
}
