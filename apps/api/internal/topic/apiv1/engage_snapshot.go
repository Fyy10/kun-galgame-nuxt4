package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) buildTopicEngagement(ctx context.Context, topic *model.Topic, viewer *middleware.UserInfo) (*TopicEngagement, *problem.Problem) {
	if s.topics == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	samples, err := s.topics.SampleTopicReactions([]int{topic.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	userIDs := make([]int, 0, len(samples))
	for _, row := range samples {
		userIDs = append(userIDs, row.UserID)
	}
	users, p := s.lookupUsers(ctx, userIDs)
	if p != nil {
		return nil, p
	}
	tv, mine, p := s.loadTopicViewer(topic, viewer)
	if p != nil {
		return nil, p
	}
	if tv == nil {
		tv = &TopicViewer{}
	}
	react := reactionSummaries(samples, users, s.cdn, mine, viewer)
	if react == nil {
		react = []ReactionSummary{}
	}
	return &TopicEngagement{
		Object:        "topic_engagement",
		TopicID:       repr.ID(topic.ID),
		LikeCount:     topic.LikeCount,
		DislikeCount:  topic.DislikeCount,
		FavoriteCount: topic.FavoriteCount,
		UpvoteCount:   topic.UpvoteCount,
		UpvotedAt:     repr.TimestampPtr(topic.UpvoteTime),
		Reactions:     react,
		Viewer:        *tv,
	}, nil
}

func (s *Service) buildReplyEngagement(ctx context.Context, topic *model.Topic, row *model.TopicReply, viewer *middleware.UserInfo) (*ReplyEngagement, *problem.Problem) {
	if s.replies == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	samples, err := s.replies.SampleReplyReactions([]int{row.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	userIDs := make([]int, 0, len(samples))
	for _, sample := range samples {
		userIDs = append(userIDs, sample.UserID)
	}
	users, p := s.lookupUsers(ctx, userIDs)
	if p != nil {
		return nil, p
	}
	mine := map[string]struct{}{}
	if viewer != nil {
		raw, err := s.replies.GetUserRepliesReactions([]int{row.ID}, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		mine = tokenSet(raw[row.ID])
	}
	rv := replyViewer(topic, row, viewer, mine)
	if rv == nil {
		rv = &ReplyViewer{}
	}
	react := reactionSummaries(samples, users, s.cdn, mine, viewer)
	if react == nil {
		react = []ReactionSummary{}
	}
	return &ReplyEngagement{
		Object:       "reply_engagement",
		ReplyID:      repr.ID(row.ID),
		LikeCount:    row.LikeCount,
		DislikeCount: row.DislikeCount,
		Reactions:    react,
		Viewer:       *rv,
	}, nil
}

func (x *Interactions) topicEngagementOut(ctx context.Context, topicID int, viewer *middleware.UserInfo) (*topicEngagementOutput, error) {
	topic, err := x.reads.topics.FindByID(topicID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	body, p := x.reads.buildTopicEngagement(ctx, topic, viewer)
	if p != nil {
		return nil, p
	}
	return &topicEngagementOutput{Body: *body}, nil
}

func (x *Interactions) replyEngagementOut(ctx context.Context, topic *model.Topic, replyID int, viewer *middleware.UserInfo) (*replyEngagementOutput, error) {
	row, err := x.reads.replies.FindByID(replyID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	body, p := x.reads.buildReplyEngagement(ctx, topic, row, viewer)
	if p != nil {
		return nil, p
	}
	return &replyEngagementOutput{Body: *body}, nil
}

func (x *Interactions) topicOut(ctx context.Context, topicID int, viewer *middleware.UserInfo) (*topicOutput, error) {
	topic, err := x.reads.topics.FindByID(topicID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	body, p := x.reads.buildTopic(ctx, topic, viewer)
	if p != nil {
		return nil, p
	}
	return &topicOutput{Body: *body}, nil
}
