package repository

import "time"

type CommentListRow struct {
	ID              int
	TopicReplyID    int
	TopicID         int
	Content         string
	UserID          int
	TargetUserID    int
	ParentCommentID *int
	LikeCount       int
	CreatedAt       time.Time
	Edited          *time.Time
}

func (r *CommentRepository) ListByReplyIDs(replyIDs []int) ([]CommentListRow, error) {
	if len(replyIDs) == 0 {
		return []CommentListRow{}, nil
	}
	var rows []CommentListRow
	err := r.db.Table("topic_comment tc").
		Select(`tc.id, tc.topic_reply_id, tc.topic_id, tc.content,
			tc.user_id, tc.target_user_id, tc.parent_comment_id,
			(SELECT COUNT(*) FROM topic_comment_like WHERE topic_comment_id = tc.id) AS like_count,
			tc.created AS created_at, tc.edited`).
		Where("tc.topic_reply_id IN ?", replyIDs).
		Where("tc.status = ?", 0).
		Order("tc.created ASC, tc.id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *TopicRepository) LookupMoemoepoints(userIDs []int) (map[int]int, error) {
	out := map[int]int{}
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID      int `gorm:"column:user_id"`
		Moemoepoint int `gorm:"column:moemoepoint"`
	}
	err := r.db.Table("kungal_user_state").
		Select("user_id, moemoepoint").
		Where("user_id IN ?", userIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserID] = row.Moemoepoint
	}
	return out, nil
}
