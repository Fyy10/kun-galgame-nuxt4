package dto

type ListMessagesRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=30"`
	Type      string `query:"type"`
	SortOrder string `query:"sort_order" validate:"required,oneof=asc desc"`
}

type KunUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type MessageResponse struct {
	ID         int     `json:"id"`
	Sender     KunUser `json:"sender"`
	ReceiverID int     `json:"receiver_id"`
	Link       string  `json:"link"`
	Content    string  `json:"content"`
	Status     string  `json:"status"`
	Type       string  `json:"type"`
	Created    string  `json:"created"`
	ItemCount  int     `json:"item_count"`
	ActorCount int     `json:"actor_count"`
	Community  bool    `json:"community"`
}

type MessageListResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int64             `json:"total"`
}

type SystemMessageResponse struct {
	ID      int     `json:"id"`
	IsRead  bool    `json:"is_read"`
	Content string  `json:"content"`
	Admin   KunUser `json:"admin"`
	Created string  `json:"created"`
}
