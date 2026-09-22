package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) FindCommentLikeStatus(userID int, commentIDs []int) (map[int]bool, error) {
	return findInteractionStatus(r.db, "topic_comment_like", "topic_comment_id", userID, commentIDs)
}

func (r *CommentRepository) FindCommentByID(id int) (*model.TopicComment, error) {
	var comment model.TopicComment
	err := r.db.First(&comment, id).Error
	return &comment, err
}

func (r *CommentRepository) CountCommentLikes(commentID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicCommentLike{}).Where("topic_comment_id = ?", commentID).Count(&count).Error
	return count, err
}

func (r *CommentRepository) DeleteCommentLikesForComment(tx *gorm.DB, commentID int) error {
	return tx.Where("topic_comment_id = ?", commentID).Delete(&model.TopicCommentLike{}).Error
}

func (r *CommentRepository) DeleteCommentByID(tx *gorm.DB, commentID int) error {
	return tx.Delete(&model.TopicComment{}, commentID).Error
}

func (r *CommentRepository) SetStatus(id, status int) error {
	return r.db.Model(&model.TopicComment{}).Where("id = ?", id).Update("status", status).Error
}
