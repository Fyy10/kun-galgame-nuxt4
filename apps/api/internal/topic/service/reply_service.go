package service

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/trust/gate"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ReplyService struct {
	replyRepo   *repository.ReplyRepository
	topicRepo   *repository.TopicRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	rdb         *redis.Client
	check       *gate.CheckService
	scan        *gate.ScanService
	helpers     InteractionHelpers
}

func NewReplyService(
	replyRepo *repository.ReplyRepository,
	topicRepo *repository.TopicRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
	rdb *redis.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *ReplyService {
	return &ReplyService{
		replyRepo:   replyRepo,
		topicRepo:   topicRepo,
		stateRepo:   stateRepo,
		userClient:  userClient,
		rdb:         rdb,
		check:       check,
		scan:        scan,
	}
}

func (s *ReplyService) LocateReply(topicID, floor, commentID, limit int, userInfo *middleware.UserInfo) (*dto.ReplyLocateResponse, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该话题")
	}
	if _, appErr := requireTopicRead(s.topicRepo, topic, userInfo); appErr != nil {
		return nil, appErr
	}
	replyID := 0
	if commentID > 0 {
		f, rid, ok, err := s.replyRepo.FindReplyFloorByCommentID(topicID, commentID)
		if err != nil {
			return nil, errors.ErrInternal("定位评论失败")
		}
		if !ok {
			return nil, errors.ErrNotFound("评论不存在或已删除")
		}
		floor, replyID = f, rid
	}
	if floor <= 0 {
		return nil, errors.ErrBadRequest("缺少 reply 或 comment 参数")
	}
	page, err := s.replyRepo.LocateReplyPageByFloor(topicID, floor, limit)
	if err != nil {
		return nil, errors.ErrInternal("定位回复失败")
	}
	return &dto.ReplyLocateResponse{
		Page:      page,
		Floor:     floor,
		ReplyID:   replyID,
		CommentID: commentID,
	}, nil
}

func (s *ReplyService) ModerationRemove(replyID int) error {
	reply, err := s.replyRepo.FindByID(replyID)
	if err != nil {
		return nil
	}
	return s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.replyRepo.DeleteRepliesByIDs(tx, []int{replyID}); err != nil {
			return err
		}
		return recomputeTopicCounts(tx, reply.TopicID)
	})
}
