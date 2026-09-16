package repository

import (
	"database/sql"
	"errors"

	"kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) DB() *gorm.DB {
	return r.db
}

type MessageRow struct {
	ID         int
	SenderID   int
	ReceiverID int
	Link       string
	Content    string
	Status     string
	Type       string
	CreatedAt  string
	ItemCount  int
	ActorCount int
	Community  bool
}

func (r *MessageRepository) FindMessages(
	receiverID int,
	excludeTypes, onlyTypes []string,
	sortOrder string,
	page, limit int,
) ([]MessageRow, int64, error) {
	var rows []MessageRow
	var total int64

	query := r.db.Table("message m").
		Select(`m.id, m.sender_id,
			m.receiver_id, m.link, m.content, m.status, m.type, m.created AS created_at,
			m.item_count, m.actor_count, (m.community_notification_id IS NOT NULL) AS community`).
		Where("m.receiver_id = ?", receiverID)

	if len(onlyTypes) > 0 {
		query = query.Where("m.type IN ?", onlyTypes)
	}
	if len(excludeTypes) > 0 {
		query = query.Where("m.type NOT IN ?", excludeTypes)
	}

	query.Count(&total)

	err := query.
		Order("m.created " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error

	return rows, total, err
}

func (r *MessageRepository) DeleteByIDAndReceiver(id, receiverID int) (*model.Message, error) {
	var row model.Message
	err := r.db.Where("id = ? AND receiver_id = ?", id, receiverID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.db.Delete(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Rows, not Scan: gorm's Scan into []*int64 fails after the UPDATE has already
// committed, so the request reported 标记已读失败 and the ids were never
// forwarded, leaving the upstream fold open.
func (r *MessageRepository) MarkAllRead(receiverID int) ([]int64, error) {
	rows, err := r.db.Raw(`
		UPDATE message
		SET status = 'read', updated = now()
		WHERE receiver_id = ? AND status = 'unread'
		RETURNING community_notification_id
	`, receiverID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []int64{}
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid && id.Int64 > 0 {
			out = append(out, id.Int64)
		}
	}
	return out, rows.Err()
}

func (r *MessageRepository) MarkCommunityThreadRead(receiverID int, threadID int64, lastRead int32) error {
	return r.db.Exec(`
		UPDATE message
		SET status = 'read', updated = now()
		WHERE receiver_id = ?
		  AND community_thread_id = ?
		  AND status = 'unread'
		  AND type IN ('replied', 'commented', 'mentioned', 'followed')
		  AND community_post_number <= ?
	`, receiverID, threadID, lastRead).Error
}

type SystemMessageRow struct {
	ID        int
	Content   string
	UserID    int
	CreatedAt string
}

func (r *MessageRepository) FindSystemMessages() ([]SystemMessageRow, error) {
	var rows []SystemMessageRow
	err := r.db.Table("system_message sm").
		Select(`sm.id, sm.content, sm.user_id, sm.created AS created_at`).
		Order("sm.created DESC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) GetMaxSystemMessageID() (int64, error) {
	var maxID *int64
	err := r.db.Table("system_message").Select("MAX(id)").Scan(&maxID).Error
	if err != nil || maxID == nil {
		return 0, err
	}
	return *maxID, nil
}

func (r *MessageRepository) GetSystemReadCursor(userID int) (int64, error) {
	var row model.SystemMessageReadState
	err := r.db.First(&row, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return row.LastReadMessageID, err
}

func (r *MessageRepository) UpsertSystemReadCursorForward(userID int, lastReadID int64) error {
	return r.db.Exec(`
		INSERT INTO system_message_read_state (user_id, last_read_message_id, updated_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET last_read_message_id = GREATEST(
		        system_message_read_state.last_read_message_id,
		        EXCLUDED.last_read_message_id
		    ),
		    updated_at = NOW()
	`, userID, lastReadID).Error
}

func (r *MessageRepository) GetNavSummary(userID int, localMuted []string) ([]map[string]any, error) {
	noticeBase := func() *gorm.DB {
		q := r.db.Model(&model.Message{}).Where("receiver_id = ?", userID)
		if len(localMuted) > 0 {
			q = q.Not("type IN ?", localMuted)
		}
		return q
	}
	var noticeTotal, noticeUnread int64
	noticeBase().Count(&noticeTotal)
	noticeBase().Where("status = 'unread'").Count(&noticeUnread)

	var latestNotice model.Message
	noticeResult := noticeBase().Order("created DESC").First(&latestNotice)

	var sysTotal, sysUnread int64
	r.db.Model(&model.SystemMessage{}).Count(&sysTotal)
	cursor, _ := r.GetSystemReadCursor(userID)
	r.db.Model(&model.SystemMessage{}).Where("id > ?", cursor).Count(&sysUnread)

	var latestSys model.SystemMessage
	sysResult := r.db.Order("created DESC").First(&latestSys)

	noticeContent := ""
	if latestNotice.Content != "" {
		if len(latestNotice.Content) > 100 {
			noticeContent = latestNotice.Content[:100]
		} else {
			noticeContent = latestNotice.Content
		}
	}

	var noticeTime any = ""
	if noticeResult.RowsAffected > 0 {
		noticeTime = latestNotice.CreatedAt
	}
	var sysTime any = ""
	if sysResult.RowsAffected > 0 {
		sysTime = latestSys.CreatedAt
	}

	result := []map[string]any{
		{
			"chatroom_name":     "",
			"content":           noticeContent,
			"last_message_time": noticeTime,
			"count":             noticeTotal,
			"unread_count":      noticeUnread,
			"route":             "notice",
			"title":             "zako~",
			"avatar":            "",
		},
		{
			"chatroom_name":     "",
			"content":           "",
			"last_message_time": sysTime,
			"count":             sysTotal,
			"unread_count":      sysUnread,
			"route":             "system",
			"title":             "zako~",
			"avatar":            "",
		},
	}
	return result, nil
}
