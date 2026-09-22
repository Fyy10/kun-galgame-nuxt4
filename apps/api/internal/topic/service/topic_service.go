package service

import (
	"context"

	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

type TopicService struct {
	topicRepo    *repository.TopicRepository
	listRepo     *repository.TopicListRepository
	taxonomyRepo *repository.TopicTaxonomyRepository
	rdb          *redis.Client
	userClient   *userclient.Client
	stateRepo    *userRepo.StateRepository
}

func NewTopicService(
	topicRepo *repository.TopicRepository,
	listRepo *repository.TopicListRepository,
	taxonomyRepo *repository.TopicTaxonomyRepository,
	rdb *redis.Client,
	userClient *userclient.Client,
	stateRepo *userRepo.StateRepository,
) *TopicService {
	return &TopicService{
		topicRepo:    topicRepo,
		listRepo:     listRepo,
		taxonomyRepo: taxonomyRepo,
		rdb:          rdb,
		userClient:   userClient,
		stateRepo:    stateRepo,
	}
}

func (s *TopicService) GetMyInteractions(userID int) dto.MyTopicInteractions {
	favorited, reactions, err := s.topicRepo.UserTopicInteractions(userID)
	if err != nil {
		return dto.MyTopicInteractions{Favorited: []int{}, Reactions: map[int][]string{}}
	}
	return dto.MyTopicInteractions{Favorited: favorited, Reactions: reactions}
}

const topicUpvoteRecordLimit = 50

const topicReactionHistoryLimit = 300

func (s *TopicService) GetResourceList(
	ctx context.Context,
	req *dto.ListTopicsRequest,
	isNSFW, authenticated bool,
) ([]dto.TopicCard, int64, *errors.AppError) {
	rows, total, err := s.listRepo.FindResourceList(
		req.Page, req.Limit,
		req.SortField, req.SortOrder, req.Category,
		isNSFW, authenticated,
	)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取资源话题列表失败")
	}

	return s.mapListRows(ctx, rows, total)
}

func (s *TopicService) mapListRows(ctx context.Context, rows []repository.TopicCardRow, total int64) ([]dto.TopicCard, int64, *errors.AppError) {
	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	sectionMap, _ := s.taxonomyRepo.FindSectionNamesByTopicIDs(topicIDs)
	miniApps := s.topicRepo.FindTopicMiniApps(topicIDs)

	uids := userclient.CollectIDs(rows, func(r repository.TopicCardRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)
	for i := range rows {
		u := userMap[rows[i].UserID]
		rows[i].UserName = u.Name
		rows[i].UserAvatar = u.Avatar
	}

	cards := make([]dto.TopicCard, 0, len(rows))
	for i, r := range rows {
		if u, ok := userMap[r.UserID]; ok && !userclient.IsRenderable(u) {
			continue
		}
		cards = append(cards, toTopicCard(r, sectionMap[r.ID], miniApps[r.ID]))
		_ = i
	}
	return cards, total, nil
}
