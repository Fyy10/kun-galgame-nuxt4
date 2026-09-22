package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type Interactions struct {
	reads *Service
}

func NewInteractions(reads *Service) *Interactions {
	return &Interactions{reads: reads}
}

type topicReactionInput struct {
	TopicID  string        `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Reaction ReactionInput `path:"reaction"`
}

type topicEngagementOutput struct {
	Body TopicEngagement
}

type topicFavoriteInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type upvoteTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    UpvoteCreate
}

type upvoteTopicOutput struct {
	Body TopicUpvote
}

type listTopicUpvotesInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	collect.Page
}

type listTopicUpvotesOutput struct {
	Body repr.List[TopicUpvote]
}

type listTopicReactionsInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	collect.Page
}

type listReactionsOutput struct {
	Body repr.List[Reaction]
}

type setTopicReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    ReplyChoice
}

type clearTopicReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type topicOutput struct {
	Body Topic
}

type replyReactionInput struct {
	ReplyID  string        `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	Reaction ReactionInput `path:"reaction"`
}

type replyEngagementOutput struct {
	Body ReplyEngagement
}

type listReplyReactionsInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	collect.Page
}

var errInteractionsNotImplemented = errors.New("apiv1 topic interactions: not implemented")

func (x *Interactions) setTopicReaction(ctx context.Context, in *topicReactionInput) (*topicEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) removeTopicReaction(ctx context.Context, in *topicReactionInput) (*topicEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) favoriteTopic(ctx context.Context, in *topicFavoriteInput) (*topicEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) unfavoriteTopic(ctx context.Context, in *topicFavoriteInput) (*topicEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) upvoteTopic(ctx context.Context, in *upvoteTopicInput) (*upvoteTopicOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) listTopicUpvotes(ctx context.Context, in *listTopicUpvotesInput) (*listTopicUpvotesOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) listTopicReactions(ctx context.Context, in *listTopicReactionsInput) (*listReactionsOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) setBestAnswer(ctx context.Context, in *setTopicReplyInput) (*topicOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) clearBestAnswer(ctx context.Context, in *clearTopicReplyInput) (*topicOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) pinReply(ctx context.Context, in *setTopicReplyInput) (*topicOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) unpinReply(ctx context.Context, in *clearTopicReplyInput) (*topicOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) setReplyReaction(ctx context.Context, in *replyReactionInput) (*replyEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) removeReplyReaction(ctx context.Context, in *replyReactionInput) (*replyEngagementOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}

func (x *Interactions) listReplyReactions(ctx context.Context, in *listReplyReactionsInput) (*listReactionsOutput, error) {
	return nil, problem.Internal(errInteractionsNotImplemented)
}
