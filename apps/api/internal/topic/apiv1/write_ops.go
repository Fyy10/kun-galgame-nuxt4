package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/pkg/problem"
)

type Writes struct {
	reads *Service
}

func NewWrites(reads *Service) *Writes {
	return &Writes{reads: reads}
}

type createTopicInput struct {
	Body TopicCreate
}

type createTopicOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new topic, such as /api/v1/topics/4121."`
	Body     Topic
}

type updateTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    TopicPatch
}

type updateTopicOutput struct {
	Body Topic
}

type getTopicSourceInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type getTopicSourceOutput struct {
	Body TopicSource
}

type createReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    ReplyCreate
}

type createReplyOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new reply, such as /api/v1/replies/16335."`
	Body     Reply
}

type updateReplyInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	Body    ReplyPatch
}

type updateReplyOutput struct {
	Body Reply
}

type deleteReplyInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
}

type getReplySourceInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
}

type getReplySourceOutput struct {
	Body ReplySource
}

var errWritesNotImplemented = errors.New("apiv1 topic writes: not implemented")

func (w *Writes) createTopic(ctx context.Context, in *createTopicInput) (*createTopicOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) updateTopic(ctx context.Context, in *updateTopicInput) (*updateTopicOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) getTopicSource(ctx context.Context, in *getTopicSourceInput) (*getTopicSourceOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) createReply(ctx context.Context, in *createReplyInput) (*createReplyOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) updateReply(ctx context.Context, in *updateReplyInput) (*updateReplyOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) deleteReply(ctx context.Context, in *deleteReplyInput) (*struct{}, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}

func (w *Writes) getReplySource(ctx context.Context, in *getReplySourceInput) (*getReplySourceOutput, error) {
	return nil, problem.Internal(errWritesNotImplemented)
}
