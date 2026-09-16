package model

import "time"

type Message struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string `gorm:"type:varchar(233);default:''" json:"content"`
	Link    string `gorm:"type:varchar(100);default:''" json:"link"`
	Status  string `gorm:"default:'unread'" json:"status"`
	Type    string `gorm:"not null" json:"type"`

	SenderID   int `gorm:"column:sender_id;not null" json:"sender_id"`
	ReceiverID int `gorm:"column:receiver_id;not null" json:"receiver_id"`

	CommunityNotificationID *int64 `gorm:"column:community_notification_id" json:"community_notification_id"`
	CommunitySeq            *int64 `gorm:"column:community_seq" json:"community_seq"`
	CommunityThreadID       *int64 `gorm:"column:community_thread_id" json:"community_thread_id"`
	CommunityPostNumber     *int   `gorm:"column:community_post_number" json:"community_post_number"`
	ItemCount               int    `gorm:"column:item_count;default:1;not null" json:"item_count"`
	ActorCount              int    `gorm:"column:actor_count;default:1;not null" json:"actor_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Message) TableName() string { return "message" }

type SystemMessage struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string `gorm:"column:content;type:text;default:''" json:"content"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (SystemMessage) TableName() string { return "system_message" }

type SystemMessageReadState struct {
	UserID            int       `gorm:"column:user_id;primaryKey" json:"user_id"`
	LastReadMessageID int64     `gorm:"column:last_read_message_id;default:0" json:"last_read_message_id"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (SystemMessageReadState) TableName() string { return "system_message_read_state" }

type ChatRoom struct {
	ID                    int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                  string     `gorm:"uniqueIndex;default:''" json:"name"`
	Avatar                string     `gorm:"default:''" json:"avatar"`
	Type                  string     `gorm:"not null" json:"type"`
	LastMessageContent    string     `gorm:"column:last_message_content;default:''" json:"last_message_content"`
	LastMessageTime       *time.Time `gorm:"column:last_message_time" json:"last_message_time"`
	LastMessageSenderID   *int       `gorm:"column:last_message_sender_id" json:"last_message_sender_id"`
	LastMessageSenderName string     `gorm:"column:last_message_sender_name;default:''" json:"last_message_sender_name"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatRoom) TableName() string { return "chat_room" }

type ChatRoomParticipant struct {
	ID         int `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID int `gorm:"column:chat_room_id;not null;uniqueIndex:idx_room_participant" json:"chat_room_id"`
	UserID     int `gorm:"column:user_id;not null;uniqueIndex:idx_room_participant" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatRoomParticipant) TableName() string { return "chat_room_participant" }

type ChatRoomAdmin struct {
	ID         int `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID int `gorm:"column:chat_room_id;not null;uniqueIndex:idx_room_admin" json:"chat_room_id"`
	UserID     int `gorm:"column:user_id;not null;uniqueIndex:idx_room_admin" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatRoomAdmin) TableName() string { return "chat_room_admin" }

type ChatMessage struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatroomName string     `gorm:"column:chatroom_name;not null" json:"chatroom_name"`
	Content      string     `gorm:"type:varchar(1000);not null" json:"content"`
	IsRecall     bool       `gorm:"column:is_recall;default:false" json:"is_recall"`
	RecallTime   *time.Time `gorm:"column:recall_time" json:"recall_time"`
	EditTime     *time.Time `gorm:"column:edit_time" json:"edit_time"`

	ChatRoomID int  `gorm:"column:chat_room_id;not null" json:"chat_room_id"`
	SenderID   int  `gorm:"column:sender_id;not null" json:"sender_id"`
	ReceiverID *int `gorm:"column:receiver_id" json:"receiver_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatMessage) TableName() string { return "chat_message" }

type ChatMessageReadBy struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ReadTime      time.Time `gorm:"column:read_time;autoCreateTime" json:"read_time"`
	ChatMessageID int       `gorm:"column:chat_message_id;not null;uniqueIndex:idx_msg_read" json:"chat_message_id"`
	UserID        int       `gorm:"column:user_id;not null;uniqueIndex:idx_msg_read" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatMessageReadBy) TableName() string { return "chat_message_read_by" }

type ChatMessageReaction struct {
	ID            int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Reaction      string `gorm:"not null" json:"reaction"`
	ChatMessageID int    `gorm:"column:chat_message_id;not null;uniqueIndex:idx_msg_reaction" json:"chat_message_id"`
	UserID        int    `gorm:"column:user_id;not null;uniqueIndex:idx_msg_reaction" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (ChatMessageReaction) TableName() string { return "chat_message_reaction" }
