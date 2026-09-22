package service

import (
	"kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type CommentService struct {
	replyRepo   *repository.ReplyRepository
	commentRepo *repository.CommentRepository
}

func NewCommentService(
	replyRepo *repository.ReplyRepository,
	commentRepo *repository.CommentRepository,
) *CommentService {
	return &CommentService{
		replyRepo: replyRepo, commentRepo: commentRepo,
	}
}

func (s *CommentService) ModerationRemove(commentID int) error {
	comment, err := s.commentRepo.FindCommentByID(commentID)
	if err != nil {
		return nil
	}
	return s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.commentRepo.DeleteCommentLikesForComment(tx, commentID); err != nil {
			return err
		}
		if err := s.commentRepo.DeleteCommentByID(tx, commentID); err != nil {
			return err
		}
		return recomputeTopicCounts(tx, comment.TopicID)
	})
}
