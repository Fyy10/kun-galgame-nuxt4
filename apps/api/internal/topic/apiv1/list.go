package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type Service struct {
	list     *repository.TopicListRepository
	topics   *repository.TopicRepository
	taxonomy *repository.TopicTaxonomyRepository
	users    *userclient.Client
	cdn      string
}

func New(
	list *repository.TopicListRepository,
	topics *repository.TopicRepository,
	taxonomy *repository.TopicTaxonomyRepository,
	users *userclient.Client,
	cdn string,
) *Service {
	return &Service{list: list, topics: topics, taxonomy: taxonomy, users: users, cdn: cdn}
}

type listTopicsInput struct {
	collect.Page
	Sort        SortToken `query:"sort" default:"bumped_desc"`
	Category    string    `query:"category" enum:"galgame,technique,others" maxLength:"9" doc:"When set, only this category. Omitted means every category. There is no all token."`
	IncludeNSFW bool      `query:"include_nsfw" default:"false" doc:"When true, NSFW topics are included. Default false."`
}

type listTopicsOutput struct {
	Body repr.List[TopicSummary]
}

func (s *Service) listTopics(ctx context.Context, in *listTopicsInput) (*listTopicsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	token := string(in.Sort)
	spec, ok := lookupSort(token)
	if !ok {
		return nil, problem.New(
			problem.CodeUnknownSort,
			"The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "use a sort token declared by this operation", nil),
		)
	}
	authenticated := v1.User(ctx) != nil
	fp := listFingerprint(spec.Token, in.Category, in.IncludeNSFW, authenticated)
	keys, curErr := collect.DecodeCursor(in.Cursor, spec.Token, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseKeysetPos(keys, spec)
	if posErr != nil {
		return nil, posErr
	}

	rows, err := s.list.FindKeyset(repository.KeysetQuery{
		SortKey:       spec.Key,
		Direction:     spec.Direction,
		Category:      in.Category,
		IncludeNSFW:   in.IncludeNSFW,
		Authenticated: authenticated,
		Limit:         in.Limit,
		Pos:           pos,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}

	hasMore := len(rows) > in.Limit
	page := rows
	if hasMore {
		page = rows[:in.Limit]
	}

	ids := make([]int, len(page))
	for i, r := range page {
		ids[i] = r.ID
	}
	sectionMap := map[int][]string{}
	if len(ids) > 0 {
		sectionMap, err = s.taxonomy.FindSectionNamesByTopicIDs(ids)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	miniApps, err := s.topics.LookupMiniApps(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}

	userIDs := userclient.CollectIDs(page, func(r repository.TopicKeysetRow) int { return r.UserID })
	users, err := s.users.Users(ctx, userIDs)
	if err != nil {
		return nil, problem.Unavailable(err)
	}

	items := make([]TopicSummary, 0, len(page))
	for _, row := range page {
		author := repr.DeletedUserRef(row.UserID)
		if u, ok := users[row.UserID]; ok {
			if !userclient.IsRenderable(u) {
				continue
			}
			author = repr.NewUserRef(s.cdn, u)
		}
		item, mapErr := mapSummary(s.cdn, row, author, sectionMap[row.ID], miniApps[row.ID])
		if mapErr != nil {
			return nil, problem.Internal(mapErr)
		}
		items = append(items, item)
	}

	var next *string
	if hasMore && len(page) > 0 {
		cur := collect.EncodeCursor(spec.Token, fp, encodeKeys(page[len(page)-1], spec)...)
		next = &cur
	}
	return &listTopicsOutput{Body: repr.NewList(items, next)}, nil
}

var errUnconfigured = errors.New("apiv1 topics: service is not configured")

func parseKeysetPos(keys []string, spec sortSpec) (*repository.KeysetPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	pos := &repository.KeysetPos{ID: id, TimeSort: spec.Kind == sortKindTime}
	if spec.Kind == sortKindTime {
		t, err := time.Parse(time.RFC3339Nano, keys[0])
		if err != nil {
			return nil, invalidCursor()
		}
		pos.SortTime = t
		return pos, nil
	}
	n, err := strconv.ParseInt(keys[0], 10, 64)
	if err != nil {
		return nil, invalidCursor()
	}
	pos.SortInt = n
	return pos, nil
}

func encodeKeys(row repository.TopicKeysetRow, spec sortSpec) []string {
	id := strconv.Itoa(row.ID)
	if spec.Kind == sortKindTime {
		return []string{row.SortTime.UTC().Format(time.RFC3339Nano), id}
	}
	return []string{strconv.FormatInt(row.SortInt, 10), id}
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}
