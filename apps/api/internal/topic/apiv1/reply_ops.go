package apiv1

import (
	"context"
	"errors"
	"strconv"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func (s *Service) listTopicReplies(ctx context.Context, in *listTopicRepliesInput) (*listTopicRepliesOutput, error) {
	topic, user, p := s.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	dir, ok := replySortDir(in.Sort)
	if !ok {
		return nil, problem.New(
			problem.CodeUnknownSort,
			"The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "use a sort token declared by this operation", nil),
		)
	}
	fp := collect.Fingerprint(strconv.Itoa(topic.ID), strconv.Itoa(in.FromFloor))
	keys, curErr := collect.DecodeCursor(in.Cursor, string(in.Sort), fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseReplyPos(keys)
	if posErr != nil {
		return nil, posErr
	}

	query := repository.ReplyKeysetQuery{
		TopicID: topic.ID, Direction: dir, FromFloor: in.FromFloor, Pos: pos,
	}
	pg, p := s.gatherReplyPage(ctx, query, in.Limit)
	if p != nil {
		return nil, p
	}
	selected := pg.rows

	extra, p := s.loadReplyExtras(selected, user)
	if p != nil {
		return nil, p
	}
	users, p := s.lookupUsers(ctx, collectReplyUserIDs(selected, extra))
	if p != nil {
		return nil, p
	}
	moeIDs := make([]int, len(selected))
	for i, r := range selected {
		moeIDs[i] = r.UserID
	}
	moe, err := s.topics.LookupMoemoepoints(moeIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	docs, p := s.convertBodies(ctx, replyBodies(selected))
	if p != nil {
		return nil, p
	}
	mapped := s.mapReplies(topic, selected, docs, extra, users, moe, user)
	items := make([]Reply, 0, len(mapped))
	for _, item := range mapped {
		if item != nil {
			items = append(items, *item)
		}
	}

	var next *string
	if pg.hasMore {
		cur := collect.EncodeCursor(string(in.Sort), fp, strconv.Itoa(pg.last.Floor), strconv.Itoa(pg.last.ID))
		next = &cur
	}
	return &listTopicRepliesOutput{Body: repr.NewList(items, next)}, nil
}

type replyPage struct {
	rows    []model.TopicReply
	last    model.TopicReply
	hasMore bool
}

// Replies by banned authors are skipped, so a page may need further windows,
// but only then: the first draft read all five windows whenever more rows
// existed and loaded comments and reactions for 150 replies to show 30.
func (s *Service) gatherReplyPage(ctx context.Context, query repository.ReplyKeysetQuery, limit int) (replyPage, *problem.Problem) {
	var pg replyPage
	q := query
	for range maxWindows {
		q.Limit = limit
		rows, err := s.replies.FindKeyset(q)
		if err != nil {
			return pg, problem.Internal(err)
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		if len(rows) == 0 {
			return pg, nil
		}
		authors, p := s.lookupUsers(ctx, replyAuthorIDs(rows))
		if p != nil {
			return pg, p
		}
		for i, row := range rows {
			pg.last = row
			if u, ok := authors[row.UserID]; ok && !userclient.IsRenderable(u) {
				continue
			}
			pg.rows = append(pg.rows, row)
			if len(pg.rows) == limit {
				pg.hasMore = more || i < len(rows)-1
				return pg, nil
			}
		}
		if !more {
			return pg, nil
		}
		q.Pos = &repository.ReplyKeysetPos{Floor: pg.last.Floor, ID: pg.last.ID}
	}
	pg.hasMore = true
	return pg, nil
}

func replyAuthorIDs(rows []model.TopicReply) []int {
	return userclient.CollectIDs(rows, func(r model.TopicReply) int { return r.UserID })
}

func (s *Service) getReply(ctx context.Context, in *getReplyInput) (*getReplyOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	id, ok := parsePositiveID(in.ReplyID)
	if !ok {
		return nil, notFound()
	}
	row, err := s.replies.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	if row.Status != 0 {
		return nil, notFound()
	}
	topic, user, p := s.visibleTopic(ctx, strconv.Itoa(row.TopicID))
	if p != nil {
		return nil, p
	}
	if row.TopicID != topic.ID {
		return nil, notFound()
	}
	mapped, p := s.buildOneReply(ctx, topic, *row, user)
	if p != nil {
		return nil, p
	}
	if mapped == nil {
		return nil, notFound()
	}
	return &getReplyOutput{Body: *mapped}, nil
}

func (s *Service) buildOneReply(ctx context.Context, topic *model.Topic, row model.TopicReply, viewer *middleware.UserInfo) (*Reply, *problem.Problem) {
	replies := []model.TopicReply{row}
	extra, p := s.loadReplyExtras(replies, viewer)
	if p != nil {
		return nil, p
	}
	users, p := s.lookupUsers(ctx, collectReplyUserIDs(replies, extra))
	if p != nil {
		return nil, p
	}
	if u, ok := users[row.UserID]; ok && !userclient.IsRenderable(u) {
		return nil, notFound()
	}
	moe, err := s.topics.LookupMoemoepoints([]int{row.UserID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	docs, p := s.convertBodies(ctx, []string{row.Content})
	if p != nil {
		return nil, p
	}
	out := s.mapReplies(topic, replies, docs, extra, users, moe, viewer)
	if len(out) == 0 {
		return nil, nil
	}
	return out[0], nil
}

func replySortDir(token ReplySortToken) (string, bool) {
	switch token {
	case "", "floor_asc":
		return "asc", true
	case "floor_desc":
		return "desc", true
	default:
		return "", false
	}
}

func parseReplyPos(keys []string) (*repository.ReplyKeysetPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	floor, err1 := strconv.Atoi(keys[0])
	id, err2 := strconv.Atoi(keys[1])
	if err1 != nil || err2 != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.ReplyKeysetPos{Floor: floor, ID: id}, nil
}
