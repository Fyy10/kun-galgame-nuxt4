package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (s *Service) getComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	topic, _, comment, user, p := s.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	return s.commentOut(ctx, topic, comment.ID, user)
}

func (s *Service) commentOut(ctx context.Context, topic *model.Topic, commentID int, viewer *middleware.UserInfo) (*commentOutput, error) {
	mapped, p := s.buildOneComment(ctx, topic, commentID, viewer)
	if p != nil {
		return nil, p
	}
	return &commentOutput{Body: *mapped}, nil
}

func (s *Service) buildOneComment(ctx context.Context, topic *model.Topic, commentID int, viewer *middleware.UserInfo) (*Comment, *problem.Problem) {
	if s == nil || s.comments == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	row, err := s.comments.FindVisibleCommentRow(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	docs, p := s.convertCommentBodies(ctx, []string{row.Content})
	if p != nil {
		return nil, p
	}
	users, p := s.lookupUsers(ctx, []int{row.UserID, row.TargetUserID})
	if p != nil {
		return nil, p
	}
	liked := map[int]bool{}
	if viewer != nil {
		liked, err = s.comments.FindCommentLikeStatus(viewer.ID, []int{row.ID})
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	pack := &replyPack{
		docs:   map[int]content.ContentDocument{row.ID: docs[0]},
		likedC: liked,
		users:  users,
	}
	mapped, ok := pack.mapComment(s.cdn, topic, *row, viewer)
	if !ok {
		return nil, notFound()
	}
	return &mapped, nil
}
