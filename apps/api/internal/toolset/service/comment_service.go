package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/toolset/dto"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"
)

type CommentService struct {
	userClient *userclient.Client
	community  *communityclient.Client
}

func NewCommentService(
	userClient *userclient.Client,
	community *communityclient.Client,
) *CommentService {
	return &CommentService{userClient: userClient, community: community}
}

func (s *CommentService) GetLatestForDetail(ctx context.Context, toolsetID, limit int) []dto.CommentDetailItem {
	items := []dto.CommentDetailItem{}
	page, err := s.community.GetComments(ctx, communityclient.AnchorSiteResource, "toolset:"+strconv.Itoa(toolsetID), "", "")
	if err != nil {
		slog.Warn("toolset detail: community read failed (best-effort)", "toolset_id", toolsetID, "error", err)
		return items
	}

	uids := make([]int, 0, len(page.Posts))
	for _, p := range page.Posts {
		uids = append(uids, int(p.AuthorID))
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	for _, p := range page.Posts {
		if len(items) >= limit {
			break
		}
		if p.Status != communityclient.PostVisible {
			continue
		}
		u := userMap[int(p.AuthorID)]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, toolsetDetailItemFromPost(p, toolsetID, userBriefFromClient(u)))
	}
	return items
}

func toolsetDetailItemFromPost(p communityclient.PostView, toolsetID int, user userModel.UserBrief) dto.CommentDetailItem {
	var parentID *int
	if p.ReplyToPostID != 0 {
		pid := int(p.ReplyToPostID)
		parentID = &pid
	}
	created := parseCommunityTime(p.CreatedAt)
	updated := created
	if edited := parseCommunityTime(p.EditedAt); !edited.IsZero() {
		updated = edited
	}
	return dto.CommentDetailItem{
		ID:        int(p.ID),
		Content:   p.ContentRaw,
		UserID:    int(p.AuthorID),
		ToolsetID: toolsetID,
		ParentID:  parentID,
		Edited:    parseCommunityTimePtr(p.EditedAt),
		Created:   created,
		Updated:   updated,
		User:      user,
	}
}

func parseCommunityTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseCommunityTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}
