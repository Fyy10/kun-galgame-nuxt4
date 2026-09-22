package apiv1

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
)

func topicViewer(topic *model.Topic, viewer *middleware.UserInfo, mine map[string]struct{}, favorited, upvoted bool) *TopicViewer {
	if viewer == nil {
		return nil
	}
	caps := capsForTopic(topic, viewer)
	return &TopicViewer{
		HasLiked:         hasToken(mine, "like"),
		HasDisliked:      hasToken(mine, "dislike"),
		HasFavorited:     favorited,
		HasUpvoted:       upvoted,
		CanEdit:          caps.Edit,
		CanHide:          caps.Hide,
		CanUnhide:        caps.Unhide,
		CanLike:          caps.Like,
		CanUpvote:        caps.Upvote,
		CanSetBestAnswer: caps.SetBestAnswer,
		CanPinReply:      caps.PinReply,
	}
}

func replyViewer(topic *model.Topic, reply *model.TopicReply, viewer *middleware.UserInfo, mine map[string]struct{}) *ReplyViewer {
	if viewer == nil {
		return nil
	}
	caps := capsForReply(topic, reply, viewer)
	return &ReplyViewer{
		HasLiked:    hasToken(mine, "like"),
		HasDisliked: hasToken(mine, "dislike"),
		CanEdit:     caps.Edit,
		CanDelete:   caps.Delete,
		CanLike:     caps.Like,
	}
}

func (s *Service) loadTopicViewer(topic *model.Topic, viewer *middleware.UserInfo) (*TopicViewer, map[string]struct{}, *problem.Problem) {
	if viewer == nil {
		return nil, nil, nil
	}
	if s.topics == nil {
		return nil, nil, problem.Internal(errUnconfigured)
	}
	toks, err := s.topics.GetUserTopicReactions(topic.ID, viewer.ID)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	mine := tokenSet(toks)
	fav, err := s.topics.HasUserFavorited(viewer.ID, topic.ID)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	up, err := s.topics.HasUserUpvoted(viewer.ID, topic.ID)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	return topicViewer(topic, viewer, mine, fav, up), mine, nil
}
