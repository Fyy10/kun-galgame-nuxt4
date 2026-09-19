package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type ReplySortToken string

func (ReplySortToken) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum("floor_asc", "floor_desc")
	s.Description = "Sort order. floor_asc (default) reads from the first floor; floor_desc from the last."
	return s
}

type getTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type getTopicOutput struct {
	Body Topic
}

type listTopicRepliesInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	collect.Page
	Sort      ReplySortToken `query:"sort" default:"floor_asc"`
	FromFloor int            `query:"from_floor" minimum:"1" doc:"Start at this floor instead of the first one in sort order, inclusive: floor_asc reads floors greater than or equal to it, floor_desc floors less than or equal to it. The floor need not exist."`
}

type listTopicRepliesOutput struct {
	Body repr.List[Reply]
}

type getReplyInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
}

type getReplyOutput struct {
	Body Reply
}

type recordTopicViewInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

var errNotImplemented = errors.New("apiv1 topics: not implemented")

func (s *Service) getTopic(ctx context.Context, in *getTopicInput) (*getTopicOutput, error) {
	return nil, problem.Internal(errNotImplemented)
}

func (s *Service) listTopicReplies(ctx context.Context, in *listTopicRepliesInput) (*listTopicRepliesOutput, error) {
	return nil, problem.Internal(errNotImplemented)
}

func (s *Service) getReply(ctx context.Context, in *getReplyInput) (*getReplyOutput, error) {
	return nil, problem.Internal(errNotImplemented)
}

func (s *Service) recordTopicView(ctx context.Context, in *recordTopicViewInput) (*struct{}, error) {
	return nil, problem.Internal(errNotImplemented)
}
