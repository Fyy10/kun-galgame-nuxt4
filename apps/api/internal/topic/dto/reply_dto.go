package dto

import "time"

type ListRepliesRequest struct {
	TopicID   int    `query:"topic_id" validate:"required,min=1"`
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=30"`
	SortOrder string `query:"sort_order" validate:"required,oneof=asc desc"`
}

type ReplyLocateResponse struct {
	Page      int `json:"page"`
	Floor     int `json:"floor"`
	ReplyID   int `json:"reply_id"`
	CommentID int `json:"comment_id"`
}

type CreateReplyRequest struct {
	TopicID int    `json:"topic_id" validate:"required,min=1"`
	Content string `json:"content" validate:"required,max=10007"`
}

type UpdateReplyRequest struct {
	ReplyID int    `json:"reply_id" validate:"required,min=1"`
	Content string `json:"content" validate:"required,max=10007"`
}

type ReplyInteractionRequest struct {
	ReplyID int `json:"reply_id" validate:"required,min=1"`
}

type ReactionRequest struct {
	Reaction string `json:"reaction" validate:"required"`
}

type ReplyReactionRequest struct {
	ReplyID  int    `json:"reply_id" validate:"required,min=1"`
	Reaction string `json:"reaction" validate:"required"`
}

type BestAnswerRequest struct {
	TopicID int `json:"topic_id" validate:"required,min=1"`
	ReplyID int `json:"reply_id" validate:"required,min=1"`
}

type PinReplyRequest struct {
	TopicID int `json:"topic_id" validate:"required,min=1"`
	ReplyID int `json:"reply_id" validate:"required,min=1"`
}

type TopicReplyResponse struct {
	ID              int                    `json:"id"`
	TopicID         int                    `json:"topic_id"`
	Floor           int                    `json:"floor"`
	User            KunUserWithMoemoepoint `json:"user"`
	Edited          *time.Time             `json:"edited"`
	ContentMarkdown string                 `json:"content_markdown"`
	ContentHtml     string                 `json:"content_html"`
	LikeCount       int                    `json:"like_count"`
	IsLiked         bool                   `json:"is_liked"`
	DislikeCount    int                    `json:"dislike_count"`
	IsDisliked      bool                   `json:"is_disliked"`
	Reactions       []ReactionSummary      `json:"reactions"`
	Comments        []TopicCommentResponse `json:"comment"`
	IsPinned        bool                   `json:"is_pinned"`
	IsBestAnswer    bool                   `json:"is_best_answer"`
	Created         time.Time              `json:"created"`
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
