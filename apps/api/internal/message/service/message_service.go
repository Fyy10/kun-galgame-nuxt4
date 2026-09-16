package service

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type MessageService struct {
	messageRepo *repository.MessageRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	community   *communityclient.Client
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
	community *communityclient.Client,
) *MessageService {
	return &MessageService{messageRepo: messageRepo, stateRepo: stateRepo, userClient: userClient, community: community}
}

func (s *MessageService) localMutedFor(userID int) []string {
	state, err := s.stateRepo.FindByID(userID)
	if err != nil || state == nil {
		return nil
	}
	local, _ := SplitMuted(state.MutedNotificationTypes)
	return local
}

func (s *MessageService) hydrateMessageRows(ctx context.Context, rows []repository.MessageRow) []dto.MessageResponse {
	uids := userclient.CollectIDs(rows, func(r repository.MessageRow) int { return r.SenderID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.MessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.SenderID]
		if !userclient.IsRenderable(u) {
			continue
		}
		itemCount, actorCount := r.ItemCount, r.ActorCount
		if itemCount < 1 {
			itemCount = 1
		}
		if actorCount < 1 {
			actorCount = 1
		}
		messages = append(messages, dto.MessageResponse{
			ID:         r.ID,
			Sender:     dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ReceiverID: r.ReceiverID,
			Link:       r.Link,
			Content:    r.Content,
			Status:     r.Status,
			Type:       r.Type,
			Created:    r.CreatedAt,
			ItemCount:  itemCount,
			ActorCount: actorCount,
			Community:  r.Community,
		})
	}
	return messages
}

func (s *MessageService) GetMessages(
	ctx context.Context,
	userID int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	rows, total, err := s.messageRepo.FindMessages(
		userID, s.localMutedFor(userID), nil, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}
	return &dto.MessageListResponse{Messages: s.hydrateMessageRows(ctx, rows), Total: total}, nil
}

func (s *MessageService) GetMutedMessages(
	ctx context.Context,
	userID int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	empty := &dto.MessageListResponse{Messages: []dto.MessageResponse{}, Total: 0}

	localMuted := s.localMutedFor(userID)
	if len(localMuted) == 0 {
		return empty, nil
	}

	onlyTypes := localMuted
	if req.Type != "" {
		if !slices.Contains(localMuted, req.Type) {
			return empty, nil
		}
		onlyTypes = []string{req.Type}
	}

	rows, total, err := s.messageRepo.FindMessages(
		userID, nil, onlyTypes, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}
	return &dto.MessageListResponse{Messages: s.hydrateMessageRows(ctx, rows), Total: total}, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, userID, messageID int) *errors.AppError {
	row, err := s.messageRepo.DeleteByIDAndReceiver(messageID, userID)
	if err != nil {
		return errors.ErrInternal("删除消息失败")
	}
	if row != nil && row.Status == "unread" && row.CommunityNotificationID != nil && *row.CommunityNotificationID > 0 {
		s.forwardRead(userID, []int64{*row.CommunityNotificationID})
	}
	return nil
}

func (s *MessageService) MarkAllRead(ctx context.Context, userID int) *errors.AppError {
	ids, err := s.messageRepo.MarkAllRead(userID)
	if err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	s.forwardRead(userID, ids)
	return nil
}

func (s *MessageService) forwardRead(userID int, ids []int64) {
	if s.community == nil || !s.community.Configured() || len(ids) == 0 {
		return
	}
	copied := append([]int64(nil), ids...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		// Forward by id, never {all:true}: all would also mark rows dispatched
		// upstream but not mirrored yet, so the user would never see them unread.
		for i := 0; i < len(copied); i += 100 {
			end := i + 100
			if end > len(copied) {
				end = len(copied)
			}
			if _, err := s.community.MarkNotificationsRead(ctx, int64(userID), copied[i:end]); err != nil {
				slog.Warn("community notification read forward failed", "user_id", userID, "error", err)
				return
			}
		}
	}()
}

func (s *MessageService) GetSystemMessages(ctx context.Context, userID int) ([]dto.SystemMessageResponse, *errors.AppError) {
	rows, err := s.messageRepo.FindSystemMessages()
	if err != nil {
		return nil, errors.ErrInternal("获取系统消息失败")
	}

	cursor, _ := s.messageRepo.GetSystemReadCursor(userID)

	uids := userclient.CollectIDs(rows, func(r repository.SystemMessageRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.SystemMessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		messages = append(messages, dto.SystemMessageResponse{
			ID:      r.ID,
			IsRead:  int64(r.ID) <= cursor,
			Content: r.Content,
			Admin:   dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.CreatedAt,
		})
	}
	return messages, nil
}

func (s *MessageService) MarkAllSystemRead(ctx context.Context, userID int) *errors.AppError {
	maxID, err := s.messageRepo.GetMaxSystemMessageID()
	if err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	if err := s.messageRepo.UpsertSystemReadCursorForward(userID, maxID); err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	return nil
}

func (s *MessageService) GetNavSummary(ctx context.Context, userID int) ([]map[string]any, *errors.AppError) {
	result, err := s.messageRepo.GetNavSummary(userID, s.localMutedFor(userID))
	if err != nil {
		return nil, errors.ErrInternal("获取消息概要失败")
	}
	return result, nil
}
