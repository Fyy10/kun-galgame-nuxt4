package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

type CommentRow struct {
	ID              int
	TopicReplyID    int
	TopicID         int
	Content         string
	UserID          int
	UserName        string
	UserAvatar      string
	TargetUserID    int
	TargetUserName  string
	TargetAvatar    string
	ParentCommentID *int
	LikeCount       int
	CreatedAt       time.Time
	Edited          *time.Time
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

func (r *CommentRepository) CreateComment(tx *gorm.DB, c *model.TopicComment) error {
	return tx.Create(c).Error
}

func (r *CommentRepository) UpdateCommentContent(tx *gorm.DB, commentID int, fields map[string]any) error {
	return tx.Model(&model.TopicComment{}).Where("id = ?", commentID).Updates(fields).Error
}

func (r *CommentRepository) FindCommentByIDTx(tx *gorm.DB, commentID int) (*model.TopicComment, error) {
	var comment model.TopicComment
	err := tx.First(&comment, commentID).Error
	return &comment, err
}

func (r *CommentRepository) FindCommentLike(tx *gorm.DB, userID, commentID int) (*model.TopicCommentLike, error) {
	var existing model.TopicCommentLike
	err := tx.Where("user_id = ? AND topic_comment_id = ?", userID, commentID).First(&existing).Error
	return &existing, err
}

func (r *CommentRepository) CreateCommentLike(tx *gorm.DB, userID, commentID int) error {
	return tx.Create(&model.TopicCommentLike{UserID: userID, TopicCommentID: commentID}).Error
}

func (r *CommentRepository) DeleteCommentLike(tx *gorm.DB, like *model.TopicCommentLike) error {
	return tx.Delete(like).Error
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
