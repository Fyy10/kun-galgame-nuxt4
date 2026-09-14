package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type ChatService struct {
	chatRepo   *repository.ChatRepository
	userClient *userclient.Client
}

func NewChatService(
	chatRepo *repository.ChatRepository,
	userClient *userclient.Client,
) *ChatService {
	return &ChatService{chatRepo: chatRepo, userClient: userClient}
}

func (s *ChatService) GetNavContact(ctx context.Context, userID int) ([]dto.NavContactItem, *errors.AppError) {
	rooms, err := s.chatRepo.FindRoomsForUser(userID)
	if err != nil {
		return nil, errors.ErrInternal("查询聊天室失败")
	}
	if len(rooms) == 0 {
		return []dto.NavContactItem{}, nil
	}

	roomIDs := make([]int, len(rooms))
	for i, r := range rooms {
		roomIDs[i] = r.ID
	}

	participants := s.chatRepo.FindParticipantsByRoomIDs(roomIDs)
	roomParts := make(map[int][]repository.ParticipantRow)
	for _, p := range participants {
		roomParts[p.ChatRoomID] = append(roomParts[p.ChatRoomID], p)
	}

	pids := userclient.CollectIDs(participants, func(p repository.ParticipantRow) int { return p.UserID })
	userMap := s.userClient.Hydrate(ctx, pids)

	unreadMap := make(map[int]int)
	for _, u := range s.chatRepo.CountUnreadByRoomIDs(roomIDs, userID) {
		unreadMap[u.ChatRoomID] = u.Count
	}
	totalMap := make(map[int]int)
	for _, t := range s.chatRepo.CountTotalByRoomIDs(roomIDs) {
		totalMap[t.ChatRoomID] = t.Count
	}

	items := make([]dto.NavContactItem, len(rooms))
	for i, r := range rooms {
		title, avatar, route := r.Name, r.Avatar, r.Name
		if r.Type == "private" {
			for _, p := range roomParts[r.ID] {
				if p.UserID != userID {
					u := userMap[p.UserID]
					title = u.Name
					avatar = u.Avatar
					route = strconv.Itoa(p.UserID)
					break
				}
			}
		}
		items[i] = dto.NavContactItem{
			ChatroomName:    r.Name,
			Content:         r.LastMessageContent,
			LastMessageTime: r.LastMessageTime,
			Count:           totalMap[r.ID],
			UnreadCount:     unreadMap[r.ID],
			Route:           route,
			Title:           title,
			Avatar:          avatar,
		}
	}
	return items, nil
}

func (s *ChatService) GetChatHistory(
	ctx context.Context,
	userID int,
	req *dto.GetChatHistoryRequest,
) ([]dto.ChatMessageItem, *errors.AppError) {
	if req.ReceiverID == userID {
		return nil, errors.ErrBadRequest("不能给自己发送消息")
	}

	roomID, roomName, err := s.findOrCreatePrivateRoom(userID, req.ReceiverID)
	if err != nil {
		return nil, errors.ErrInternal("查询聊天室失败")
	}
	if roomID == 0 {
		return []dto.ChatMessageItem{}, nil
	}

	rows := s.chatRepo.FindMessagesByRoom(roomID, roomName, req.Page, req.Limit)

	if len(rows) > 0 {
		msgIDs := make([]int, 0, len(rows))
		for _, m := range rows {
			if m.SenderID != userID {
				msgIDs = append(msgIDs, m.ID)
			}
		}
		if err := s.chatRepo.MarkMessagesRead(msgIDs, userID); err != nil {
			slog.Warn("标记聊天消息已读失败", "user_id", userID, "room_id", roomID, "error", err)
		}
	}

	uids := userclient.CollectIDs(rows, func(m repository.ChatMessageRow) int { return m.SenderID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.ChatMessageItem, len(rows))
	for i, m := range rows {
		u := userMap[m.SenderID]

		content := m.Content
		contentHTML := markdown.RenderInline(m.Content)
		if m.IsRecall {
			content = ""
			contentHTML = ""
		}

		items[len(rows)-1-i] = dto.ChatMessageItem{
			ID:           m.ID,
			ChatroomName: m.ChatroomName,
			Sender:       dto.ChatSender{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ReceiverID:   m.ReceiverID,
			Content:      content,
			ContentHtml:  contentHTML,
			IsRecall:     m.IsRecall,
			Created:      m.Created,
			RecallTime:   m.RecallTime,
			EditTime:     m.EditTime,
			ReadBy:       []dto.ChatSender{},
		}
	}
	return items, nil
}

func (s *ChatService) SendChatMessage(
	ctx context.Context,
	senderUserID int,
	senderName string,
	req *dto.SendChatMessageRequest,
) *errors.AppError {
	req.Content = markdown.NormalizeStoredContent(req.Content)
	if req.ReceiverID == senderUserID {
		return errors.ErrBadRequest("不能给自己发送消息")
	}

	roomID, roomName, err := s.findOrCreatePrivateRoom(senderUserID, req.ReceiverID)
	if err != nil || roomID == 0 {
		return errors.ErrInternal("创建聊天室失败")
	}

	now := time.Now()
	txErr := s.chatRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.chatRepo.InsertChatMessage(tx, roomID, roomName, senderUserID, req.ReceiverID, req.Content, now); err != nil {
			return err
		}
		return s.chatRepo.UpdateRoomLastMessage(tx, roomID, req.Content, senderUserID, senderName, now)
	})
	if txErr != nil {
		return errors.ErrInternal("发送消息失败")
	}
	return nil
}

func (s *ChatService) RecallMessage(
	ctx context.Context,
	userID int,
	messageID int,
) *errors.AppError {
	header, ok := s.chatRepo.FindMessageHeader(messageID)
	if !ok {
		return errors.ErrNotFound("消息不存在或已被删除")
	}
	if header.SenderID != userID {
		return errors.ErrForbidden("您只能撤回自己发送的消息")
	}
	if header.IsRecall {
		return errors.ErrBadRequest("该消息已被撤回")
	}

	now := time.Now()
	if err := s.chatRepo.MarkMessageRecalled(messageID, now); err != nil {
		return errors.ErrInternal("撤回消息失败")
	}

	if s.chatRepo.IsLatestMessageInRoom(header.ChatRoomID, messageID) {
		senderName := ""
		if u, _, err := s.userClient.User(ctx, header.SenderID); err == nil {
			senderName = u.Name
		}
		preview := fmt.Sprintf("%s撤回了一条消息", senderName)
		if err := s.chatRepo.UpdateRoomLastMessage(s.chatRepo.DB(), header.ChatRoomID, preview, userID, senderName, now); err != nil {
			slog.Warn("撤回后刷新聊天室预览失败", "room_id", header.ChatRoomID, "error", err)
		}
	}
	return nil
}

func (s *ChatService) findOrCreatePrivateRoom(uid1, uid2 int) (int, string, error) {
	room := s.chatRepo.FindPrivateRoomBetween(uid1, uid2)
	if room.ID > 0 {
		return room.ID, room.Name, nil
	}
	newRoom, err := s.chatRepo.CreatePrivateRoom(generateRoomID(uid1, uid2), uid1, uid2)
	if err != nil {
		return 0, "", err
	}
	return newRoom.ID, newRoom.Name, nil
}

func generateRoomID(uid1, uid2 int) string {
	if uid1 < uid2 {
		return fmt.Sprintf("%d-%d", uid1, uid2)
	}
	return fmt.Sprintf("%d-%d", uid2, uid1)
}
