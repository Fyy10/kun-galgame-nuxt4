package dto

import "time"

type ReplyLocateResponse struct {
	Page      int `json:"page"`
	Floor     int `json:"floor"`
	ReplyID   int `json:"reply_id"`
	CommentID int `json:"comment_id"`
}

type CreateCommentRequest struct {
	TopicID         int    `json:"topic_id" validate:"required,min=1"`
	ReplyID         int    `json:"reply_id" validate:"required,min=1"`
	TargetUserID    int    `json:"target_user_id" validate:"omitempty,min=1"` // ignored; the server derives the target
	Content         string `json:"content" validate:"required,min=1,max=1007"`
	ParentCommentID *int   `json:"parent_comment_id" validate:"omitempty,min=1"`
}

type CommentInteractionRequest struct {
	CommentID int `json:"comment_id" validate:"required,min=1"`
}

type UpdateCommentRequest struct {
	CommentID int    `json:"comment_id" validate:"required,min=1"`
	Content   string `json:"content" validate:"required,min=1,max=1007"`
}

type TopicCommentResponse struct {
	ID              int        `json:"id"`
	ReplyID         int        `json:"reply_id"`
	TopicID         int        `json:"topic_id"`
	User            KunUser    `json:"user"`
	TargetUser      KunUser    `json:"target_user"`
	ParentCommentID *int       `json:"parent_comment_id"`
	Content         string     `json:"content"`
	IsLiked         bool       `json:"is_liked"`
	LikeCount       int        `json:"like_count"`
	Created         time.Time  `json:"created"`
	Edited          *time.Time `json:"edited"`
}
