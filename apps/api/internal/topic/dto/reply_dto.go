package dto

type ReplyLocateResponse struct {
	Page      int `json:"page"`
	Floor     int `json:"floor"`
	ReplyID   int `json:"reply_id"`
	CommentID int `json:"comment_id"`
}
