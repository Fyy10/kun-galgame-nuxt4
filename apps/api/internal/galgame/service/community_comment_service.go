package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type CommunityCommentService struct {
	community  *communityclient.Client
	posts      *repository.CommunityPostRepository
	userClient *userclient.Client
	db         *gorm.DB
	helpers    InteractionHelpers
}

func NewCommunityCommentService(
	community *communityclient.Client,
	posts *repository.CommunityPostRepository,
	userClient *userclient.Client,
	db *gorm.DB,
) *CommunityCommentService {
	return &CommunityCommentService{community: community, posts: posts, userClient: userClient, db: db}
}

type UserObj struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type CommunityPostItem struct {
	ID int64 `json:"id"`
	// ThreadID is how the first comment on a wall tells the page the wall now
	// exists: until then there is no thread to follow or report a receipt for.
	ThreadID          int64    `json:"thread_id"`
	Content           string   `json:"content"`
	ContentHtml       string   `json:"content_html"`
	GalgameID         int      `json:"galgame_id"`
	User              UserObj  `json:"user"`
	ParentCommentID   *int64   `json:"parent_comment_id"`
	RootCommentID     *int64   `json:"root_comment_id"`
	TargetUser        *UserObj `json:"target_user,omitempty"`
	LikeCount         int      `json:"like_count"`
	IsLiked           bool     `json:"is_liked"`
	Created           string   `json:"created"`
	Edited            *string  `json:"edited"`
	EditedByModerator bool     `json:"edited_by_moderator"`
	Status            int32    `json:"status"`
	Deleted           bool     `json:"deleted"`
	Held              bool     `json:"held"`
}

type CommunityCommentPage struct {
	ThreadID   int64                `json:"thread_id"`
	Posts      []*CommunityPostItem `json:"posts"`
	NextCursor string               `json:"next_cursor"`
	Total      int                  `json:"total"`
	Locked     bool                 `json:"locked"`
}

func (s *CommunityCommentService) GetComments(ctx context.Context, galgameID, viewerID int, cursor string, limit int) (*CommunityCommentPage, *errors.AppError) {
	page, err := s.community.GetComments(ctx, communityclient.AnchorSiteGame, strconv.Itoa(galgameID), cursor, clampReadLimit(limit))
	if err != nil {
		if isCommunityDown(err) {
			return emptyCommentPage(), nil
		}
		return nil, mapCommunityError(err)
	}
	if page.Thread == nil {
		return emptyCommentPage(), nil
	}

	items := s.renderPosts(ctx, galgameID, viewerID, page.Posts)
	return &CommunityCommentPage{
		ThreadID: page.Thread.ID, Posts: items, NextCursor: page.NextCursor, Total: int(page.Thread.PostsCount),
	}, nil
}

func (s *CommunityCommentService) renderPosts(ctx context.Context, galgameID, viewerID int, posts []communityclient.PostView) []*CommunityPostItem {
	userMap := s.userClient.Hydrate(ctx, collectPostUserIDs(posts))

	postIDs := make([]int64, 0, len(posts))
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}
	likeCounts := s.posts.CountLikes(postIDs)
	likedSet := s.posts.LikedSet(viewerID, postIDs)

	out := make([]*CommunityPostItem, 0, len(posts))
	for _, p := range posts {
		author := userMap[int(p.AuthorID)]
		if !userclient.IsRenderable(author) {
			continue
		}
		if !visibleTo(p.Status, p.AuthorID, viewerID) {
			continue
		}
		item := buildCommunityItem(p, galgameID, author, likeCounts[p.ID], likedSet[p.ID])
		if p.TargetUserID != 0 {
			t := userMap[int(p.TargetUserID)]
			tu := UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
			item.TargetUser = &tu
		}
		out = append(out, item)
	}
	return out
}

func visibleTo(status int32, authorID int64, viewerID int) bool {
	if status == communityclient.PostHeld {
		return viewerID != 0 && int64(viewerID) == authorID
	}
	return true
}

func buildCommunityItem(p communityclient.PostView, galgameID int, author userclient.User, likeCount int, liked bool) *CommunityPostItem {
	deleted := p.Status == communityclient.PostDeleted
	raw := p.ContentRaw
	if deleted {
		raw = ""
	}
	item := &CommunityPostItem{
		ID:                p.ID,
		ThreadID:          p.ThreadID,
		Content:           raw,
		ContentHtml:       markdown.Render(raw),
		GalgameID:         galgameID,
		User:              UserObj{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
		ParentCommentID:   nzPtr64(p.ReplyToPostID),
		RootCommentID:     nzPtr64(p.RootPostID),
		LikeCount:         likeCount,
		IsLiked:           liked,
		Created:           p.CreatedAt,
		EditedByModerator: p.EditedByModerator,
		Status:            p.Status,
		Deleted:           deleted,
		Held:              p.Status == communityclient.PostHeld,
	}
	if p.EditedAt != "" {
		e := p.EditedAt
		item.Edited = &e
	}
	return item
}

func clampReadLimit(limit int) string {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	return strconv.Itoa(limit)
}

func collectPostUserIDs(posts []communityclient.PostView) []int {
	set := make(map[int]struct{}, len(posts))
	for _, p := range posts {
		set[int(p.AuthorID)] = struct{}{}
		if p.TargetUserID != 0 {
			set[int(p.TargetUserID)] = struct{}{}
		}
	}
	ids := make([]int, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

func nzPtr64(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func emptyCommentPage() *CommunityCommentPage {
	return &CommunityCommentPage{Posts: []*CommunityPostItem{}}
}

func lockedCommentPage() *CommunityCommentPage {
	return &CommunityCommentPage{Posts: []*CommunityPostItem{}, Locked: true}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func isCommunityDown(err error) bool {
	if stderrors.Is(err, communityclient.ErrNotConfigured) || stderrors.Is(err, communityclient.ErrForbidden) {
		return true
	}
	var apiErr *communityclient.APIError
	return err != nil && !stderrors.As(err, &apiErr) && !stderrors.Is(err, communityclient.ErrRateLimited)
}

func mapCommunityError(err error) *errors.AppError {
	switch {
	case stderrors.Is(err, communityclient.ErrRateLimited):
		return errors.New(errors.CodeBiz, "发表过于频繁，请稍后再试（新人限制）", 429)
	case stderrors.Is(err, communityclient.ErrForbidden):
		return errors.New(errors.CodeBiz, "你没有权限执行此操作", 403)
	case stderrors.Is(err, communityclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "评论服务暂不可用", 503)
	}
	var apiErr *communityclient.APIError
	if stderrors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		return errors.New(apiErr.Code, apiErr.Msg, 400)
	}
	slog.Error("community upstream error", "error", err)
	return errors.New(errors.CodeBiz, "评论服务暂不可用", 503)
}
