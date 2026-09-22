package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type ReplyRepository struct {
	db *gorm.DB
}

func NewReplyRepository(db *gorm.DB) *ReplyRepository {
	return &ReplyRepository{db: db}
}

func (r *ReplyRepository) DB() *gorm.DB {
	return r.db
}

func (r *ReplyRepository) FindByID(id int) (*model.TopicReply, error) {
	var reply model.TopicReply
	err := r.db.First(&reply, id).Error
	return &reply, err
}

type ReplyRow struct {
	model.TopicReply
	UserName        string
	UserAvatar      string
	UserMoemoepoint int
}

func (r *ReplyRepository) FindRepliesPaginated(
	topicID int,
	excludeIDs []int,
	page, limit int,
	sortOrder string,
) ([]ReplyRow, error) {
	var rows []ReplyRow
	query := r.db.Table("topic_reply").
		Select(`topic_reply.*`).
		Where("topic_reply.topic_id = ?", topicID).
		Where("topic_reply.status = ?", 0)

	if len(excludeIDs) > 0 {
		query = query.Where("topic_reply.id NOT IN ?", excludeIDs)
	}

	err := query.
		Order("topic_reply.floor " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *ReplyRepository) LocateReplyPageByFloor(topicID, floor, limit int) (int, error) {
	if limit <= 0 {
		limit = 30
	}
	var count int64
	err := r.db.Model(&model.TopicReply{}).
		Where("topic_id = ? AND floor <= ?", topicID, floor).
		Count(&count).Error
	if err != nil {
		return 1, err
	}
	if count <= 0 {
		return 1, nil
	}
	return int((count-1)/int64(limit)) + 1, nil
}

func (r *ReplyRepository) FindReplyFloorByCommentID(topicID, commentID int) (floor int, replyID int, ok bool, err error) {
	var row struct {
		Floor int
		ID    int
	}
	e := r.db.Table("topic_comment c").
		Select("r.floor AS floor, r.id AS id").
		Joins("JOIN topic_reply r ON r.id = c.topic_reply_id").
		Where("c.id = ? AND c.topic_id = ?", commentID, topicID).
		Scan(&row).Error
	if e != nil {
		return 0, 0, false, e
	}
	if row.ID == 0 {
		return 0, 0, false, nil
	}
	return row.Floor, row.ID, true, nil
}

func (r *ReplyRepository) FindRepliesByIDs(ids []int) ([]ReplyRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []ReplyRow
	err := r.db.Table("topic_reply").
		Select(`topic_reply.*`).
		Where("topic_reply.id IN ?", ids).
		Where("topic_reply.status = ?", 0).
		Find(&rows).Error
	return rows, err
}

func (r *ReplyRepository) FindReplyLikeStatus(userID int, replyIDs []int) (map[int]bool, error) {
	return findReactionStatus(r.db, "topic_reply_reaction", "topic_reply_id", "like", userID, replyIDs)
}

func (r *ReplyRepository) FindReplyDislikeStatus(userID int, replyIDs []int) (map[int]bool, error) {
	return findReactionStatus(r.db, "topic_reply_reaction", "topic_reply_id", "dislike", userID, replyIDs)
}

func findInteractionStatus(db *gorm.DB, table, fkCol string, userID int, ids []int) (map[int]bool, error) {
	if len(ids) == 0 || userID == 0 {
		return make(map[int]bool), nil
	}
	var foundIDs []int
	err := db.Table(table).
		Where("user_id = ? AND "+fkCol+" IN ?", userID, ids).
		Pluck(fkCol, &foundIDs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(foundIDs))
	for _, id := range foundIDs {
		result[id] = true
	}
	return result, nil
}

func (r *ReplyRepository) DeleteRepliesByIDs(tx *gorm.DB, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Exec(
		"DELETE FROM topic_comment_like WHERE topic_comment_id IN (SELECT id FROM topic_comment WHERE topic_reply_id IN ?)",
		ids,
	).Error; err != nil {
		return err
	}
	for _, m := range []any{
		&model.TopicComment{}, &model.TopicReplyLike{}, &model.TopicReplyDislike{},
	} {
		if err := tx.Where("topic_reply_id IN ?", ids).Delete(m).Error; err != nil {
			return err
		}
	}

	return tx.Where("id IN ?", ids).Delete(&model.TopicReply{}).Error
}

func (r *ReplyRepository) SetStatus(id, status int) error {
	return r.db.Model(&model.TopicReply{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ReplyRepository) CountReplyRelated(replyID int) (commentCount, likeCount int64, err error) {
	r.db.Model(&model.TopicComment{}).Where("topic_reply_id = ?", replyID).Count(&commentCount)
	r.db.Model(&model.TopicReplyLike{}).Where("topic_reply_id = ?", replyID).Count(&likeCount)
	return
}

func (r *ReplyRepository) FindByIDTx(tx *gorm.DB, replyID int) (*model.TopicReply, error) {
	var reply model.TopicReply
	err := tx.First(&reply, replyID).Error
	return &reply, err
}

func (r *ReplyRepository) AdjustReplyLikeCount(tx *gorm.DB, replyID, delta int) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

func (r *ReplyRepository) AdjustReplyDislikeCount(tx *gorm.DB, replyID, delta int) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).
		Update("dislike_count", gorm.Expr("dislike_count + ?", delta)).Error
}

func (r *ReplyRepository) CreateReply(tx *gorm.DB, reply *model.TopicReply) error {
	return tx.Create(reply).Error
}

func (r *ReplyRepository) UpdateReplyContent(tx *gorm.DB, replyID int, fields map[string]any) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).Updates(fields).Error
}
