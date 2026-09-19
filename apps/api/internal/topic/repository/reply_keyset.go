package repository

import (
	"fmt"

	"kun-galgame-api/internal/topic/model"
)

type ReplyKeysetPos struct {
	Floor int
	ID    int
}

type ReplyKeysetQuery struct {
	TopicID   int
	Direction string
	FromFloor int
	Limit     int
	Pos       *ReplyKeysetPos
}

func (r *ReplyRepository) FindKeyset(q ReplyKeysetQuery) ([]model.TopicReply, error) {
	dir := q.Direction
	if dir != "asc" && dir != "desc" {
		return nil, fmt.Errorf("unsupported reply sort direction %q", dir)
	}
	query := r.db.Model(&model.TopicReply{}).
		Where("topic_id = ? AND status = 0", q.TopicID)
	if sql, arg, ok := FloorBound(dir, q.FromFloor); ok {
		query = query.Where(sql, arg)
	}
	if q.Pos != nil {
		cmp := ">"
		if dir == "desc" {
			cmp = "<"
		}
		query = query.Where("(floor, id) "+cmp+" (?, ?)", q.Pos.Floor, q.Pos.ID)
	}
	var rows []model.TopicReply
	err := query.Order("floor " + dir + ", id " + dir).Limit(q.Limit + 1).Find(&rows).Error
	return rows, err
}

func FloorBound(direction string, fromFloor int) (sql string, arg int, ok bool) {
	if fromFloor <= 0 {
		return "", 0, false
	}
	if direction == "desc" {
		return "floor <= ?", fromFloor, true
	}
	return "floor >= ?", fromFloor, true
}
