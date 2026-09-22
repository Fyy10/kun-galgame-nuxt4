package apiv1

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const historySort = "created_desc"

func historyFingerprint(kind string, id int) string {
	return collect.Fingerprint(kind, strconv.Itoa(id))
}

func decodeHistoryCursor(cursor, fingerprint string) (*repository.EngageHistoryPos, *problem.Problem) {
	keys, err := collect.DecodeCursor(cursor, historySort, fingerprint)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	created, e1 := time.Parse(time.RFC3339Nano, keys[0])
	id, e2 := strconv.Atoi(keys[1])
	if e1 != nil || e2 != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.EngageHistoryPos{Created: created, ID: id}, nil
}

func encodeHistoryCursor(fingerprint string, row repository.EngageHistoryRow) string {
	return collect.EncodeCursor(historySort, fingerprint, row.Created.UTC().Format(time.RFC3339Nano), strconv.Itoa(row.ID))
}

func historyLimit(n int) int {
	if n <= 0 {
		return collect.DefaultLimit
	}
	return n
}

type historyPage struct {
	rows    []repository.EngageHistoryRow
	last    repository.EngageHistoryRow
	hasMore bool
}

func (x *Interactions) gatherHistory(
	ctx context.Context,
	limit int,
	fetch func(limit int, pos *repository.EngageHistoryPos) ([]repository.EngageHistoryRow, error),
) (historyPage, *problem.Problem) {
	var pg historyPage
	var pos *repository.EngageHistoryPos
	for range maxWindows {
		rows, err := fetch(limit, pos)
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
		authors, p := x.reads.lookupUsers(ctx, historyUserIDs(rows))
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
		pos = &repository.EngageHistoryPos{Created: pg.last.Created, ID: pg.last.ID}
	}
	pg.hasMore = true
	return pg, nil
}

func historyUserIDs(rows []repository.EngageHistoryRow) []int {
	return userclient.CollectIDs(rows, func(r repository.EngageHistoryRow) int { return r.UserID })
}

func (x *Interactions) historyUserMap(ctx context.Context, rows []repository.EngageHistoryRow) (map[int]userclient.User, *problem.Problem) {
	return x.reads.lookupUsers(ctx, historyUserIDs(rows))
}

func (x *Interactions) userRefOf(users map[int]userclient.User, id int) repr.UserRef {
	if u, ok := users[id]; ok {
		return repr.NewUserRef(x.reads.cdn, u)
	}
	return repr.DeletedUserRef(id)
}

func (x *Interactions) listTopicUpvotes(ctx context.Context, in *listTopicUpvotesInput) (*listTopicUpvotesOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, _, p := x.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	fp := historyFingerprint("topic_upvotes", topic.ID)
	pos, p := decodeHistoryCursor(in.Cursor, fp)
	if p != nil {
		return nil, p
	}
	limit := historyLimit(in.Limit)
	pg, p := x.gatherHistory(ctx, limit, func(limit int, window *repository.EngageHistoryPos) ([]repository.EngageHistoryRow, error) {
		start := pos
		if window != nil {
			start = window
		}
		return x.reads.topics.ListTopicUpvotesKeyset(topic.ID, limit, start)
	})
	if p != nil {
		return nil, p
	}
	users, p := x.historyUserMap(ctx, pg.rows)
	if p != nil {
		return nil, p
	}
	items := make([]TopicUpvote, 0, len(pg.rows))
	for _, row := range pg.rows {
		items = append(items, TopicUpvote{
			Object:    "topic_upvote",
			ID:        repr.ID(row.ID),
			TopicID:   repr.ID(topic.ID),
			Upvoter:   x.userRefOf(users, row.UserID),
			Note:      notePtr(row.Description),
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	var next *string
	if pg.hasMore {
		cur := encodeHistoryCursor(fp, pg.last)
		next = &cur
	}
	return &listTopicUpvotesOutput{Body: repr.NewList(items, next)}, nil
}

func (x *Interactions) listTopicReactions(ctx context.Context, in *listTopicReactionsInput) (*listReactionsOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, _, p := x.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	fp := historyFingerprint("topic_reactions", topic.ID)
	pos, p := decodeHistoryCursor(in.Cursor, fp)
	if p != nil {
		return nil, p
	}
	limit := historyLimit(in.Limit)
	pg, p := x.gatherHistory(ctx, limit, func(limit int, window *repository.EngageHistoryPos) ([]repository.EngageHistoryRow, error) {
		start := pos
		if window != nil {
			start = window
		}
		return x.reads.topics.ListTopicReactionsKeyset(topic.ID, limit, start)
	})
	if p != nil {
		return nil, p
	}
	return x.reactionsPage(ctx, fp, pg)
}

func (x *Interactions) listReplyReactions(ctx context.Context, in *listReplyReactionsInput) (*listReactionsOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	_, reply, _, p := x.reads.visibleReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	fp := historyFingerprint("reply_reactions", reply.ID)
	pos, p := decodeHistoryCursor(in.Cursor, fp)
	if p != nil {
		return nil, p
	}
	limit := historyLimit(in.Limit)
	pg, p := x.gatherHistory(ctx, limit, func(limit int, window *repository.EngageHistoryPos) ([]repository.EngageHistoryRow, error) {
		start := pos
		if window != nil {
			start = window
		}
		return x.reads.replies.ListReplyReactionsKeyset(reply.ID, limit, start)
	})
	if p != nil {
		return nil, p
	}
	return x.reactionsPage(ctx, fp, pg)
}

func (x *Interactions) reactionsPage(ctx context.Context, fp string, pg historyPage) (*listReactionsOutput, error) {
	users, p := x.historyUserMap(ctx, pg.rows)
	if p != nil {
		return nil, p
	}
	items := make([]Reaction, 0, len(pg.rows))
	for _, row := range pg.rows {
		items = append(items, Reaction{
			Object:    "reaction",
			ID:        repr.ID(row.ID),
			Reaction:  ReactionToken(row.Reaction),
			Reactor:   x.userRefOf(users, row.UserID),
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	var next *string
	if pg.hasMore {
		cur := encodeHistoryCursor(fp, pg.last)
		next = &cur
	}
	return &listReactionsOutput{Body: repr.NewList(items, next)}, nil
}
