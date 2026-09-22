package dto

import (
	"time"

	"kun-galgame-api/pkg/imageclient"
)

type KunUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type ListTopicsRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=50"`
	SortField string `query:"sort_field"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Category  string `query:"category"`
}

type TopicCard struct {
	ID               int                              `json:"id"`
	Title            string                           `json:"title"`
	View             int                              `json:"view"`
	Sections         []string                         `json:"section"`
	CoverImages      []string                         `json:"cover_images"`
	CoverImageMeta   map[string]imageclient.ImageMeta `json:"cover_image_meta,omitempty"`
	User             KunUser                          `json:"user"`
	Status           int                              `json:"status"`
	HasBestAnswer    bool                             `json:"has_best_answer"`
	MiniApps         []string                         `json:"mini_apps"`
	IsNSFW           bool                             `json:"is_nsfw_topic"`
	LikeCount        int                              `json:"like_count"`
	ReplyCount       int                              `json:"reply_count"`
	CommentCount     int                              `json:"comment_count"`
	StatusUpdateTime time.Time                        `json:"status_update_time"`
	Created          time.Time                        `json:"created"`
	UpvoteTime       *time.Time                       `json:"upvote_time"`
}

type MyTopicInteractions struct {
	Favorited []int            `json:"favorited"`
	Reactions map[int][]string `json:"reactions"`
}
