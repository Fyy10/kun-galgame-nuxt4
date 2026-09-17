package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeResult struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

type LocateResult struct {
	PostID   int64 `json:"post_id"`
	ThreadID int64 `json:"thread_id"`
}

func (s *CommunityCommentService) CreateComment(ctx context.Context, userID, galgameID int, content string, replyToPostID *int64) (*CommunityPostItem, *errors.AppError) {
	content = markdown.NormalizeStoredContent(content)
	mentions, appErr := s.mentionIDsForCreate(ctx, userID, content)
	if appErr != nil {
		return nil, appErr
	}
	req := communityclient.CommentRequest{
		AnchorKind: communityclient.AnchorSiteGame, AnchorID: strconv.Itoa(galgameID),
		ContentRating: communityclient.RatingAll, AuthorID: int64(userID), Body: content,
		MentionUserIDs: mentions,
	}
	if replyToPostID != nil {
		req.ReplyToPostID = *replyToPostID
	}
	res, err := s.community.CommentOnAnchor(ctx, req)
	if err != nil {
		return nil, mapCommunityError(err)
	}
	post := &res.Post

	s.afterCreate(userID, galgameID, content, post)

	items := s.renderPosts(ctx, galgameID, userID, []communityclient.PostView{*post})
	if len(items) == 0 {
		return buildCommunityItem(*post, galgameID, s.userClient.Hydrate(ctx, []int{userID})[userID], 0, false), nil
	}
	return items[0], nil
}

func (s *CommunityCommentService) afterCreate(userID, galgameID int, content string, post *communityclient.PostView) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameLocal{ID: galgameID}).Error; e != nil {
			return e
		}
		if e := tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("comment_count", gorm.Expr("comment_count + 1")).Error; e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		slog.Warn("community comment post-create bookkeeping failed (best-effort)", "galgame_id", galgameID, "post_id", post.ID, "error", err)
	}
	feedParityUpsert(s.db, feedTypeGalgameComment, post.ID, userID, galgameID,
		content, "/galgame/"+strconv.Itoa(galgameID), false, post.CreatedAt)
}

func (s *CommunityCommentService) UpdateComment(ctx context.Context, userID int, can func(perm.Permission) bool, postID int64, galgameID *int, content string) (*CommunityPostItem, *errors.AppError) {
	content = markdown.NormalizeStoredContent(content)
	canModerate, oldRaw, found := s.resolveModEdit(ctx, can, postID)

	post, err := s.community.EditPost(ctx, postID, communityclient.EditPostRequest{
		AuthorID: int64(userID), Body: content, AsModerator: canModerate,
	})
	if err != nil {
		return nil, mapCommunityError(err)
	}

	gid := 0
	if galgameID != nil {
		gid = *galgameID
		if found {
			s.refanMentions(userID, gid, content, post, newlyAddedMentionIDs(oldRaw, content, userID))
		}
	}

	author := s.userClient.Hydrate(ctx, []int{int(post.AuthorID)})[int(post.AuthorID)]
	lc := s.posts.CountLikes([]int64{post.ID})[post.ID]
	liked := s.posts.LikedSet(userID, []int64{post.ID})[post.ID]
	return buildCommunityItem(*post, gid, author, lc, liked), nil
}

func (s *CommunityCommentService) refanMentions(userID, galgameID int, content string, post *communityclient.PostView, ids []int) {
	if len(ids) == 0 {
		return
	}
	preview := truncate(markdown.StripReferenceTokens(content), 233)
	for _, mid := range ids {
		if err := s.helpers.CreateGalgameCommentMention(s.db, userID, mid, preview, galgameID, int(post.ID)); err != nil {
			slog.Warn("mention notification insert failed (best-effort)",
				"galgame_id", galgameID, "post_id", post.ID, "receiver_id", mid, "error", err)
		}
	}
}

func (s *CommunityCommentService) resolveModEdit(ctx context.Context, can func(perm.Permission) bool, postID int64) (canModerate bool, oldRaw string, found bool) {
	fallback := can(perm.CommentGalgameEdit) ||
		can(perm.CommentRatingEdit) ||
		can(perm.CommentWebsiteEdit) ||
		can(perm.CommentToolsetEdit) ||
		can(perm.CommentResourceEdit) ||
		can(perm.CommentQuizEdit)
	resolved, err := s.community.ResolvePosts(ctx, []int64{postID})
	if err != nil {
		return fallback, "", false
	}
	for _, ap := range resolved.Posts {
		if ap.Post.ID != postID {
			continue
		}
		found = true
		oldRaw = ap.Post.ContentRaw
		if p, ok := commentEditPermForAnchor(ap.Thread.AnchorKind, ap.Thread.AnchorID); ok {
			return can(p), oldRaw, true
		}
		break
	}
	return fallback, oldRaw, found
}

func commentEditPermForAnchor(anchorKind int32, anchorID string) (perm.Permission, bool) {
	switch anchorKind {
	case communityclient.AnchorSiteGame:
		return perm.CommentGalgameEdit, true
	case communityclient.AnchorSiteResource:
		switch {
		case strings.HasPrefix(anchorID, "rating:"):
			return perm.CommentRatingEdit, true
		case strings.HasPrefix(anchorID, "website:"):
			return perm.CommentWebsiteEdit, true
		case strings.HasPrefix(anchorID, "toolset:"):
			return perm.CommentToolsetEdit, true
		case strings.HasPrefix(anchorID, "resource:"):
			return perm.CommentResourceEdit, true
		case strings.HasPrefix(anchorID, "quiz:"):
			return perm.CommentQuizEdit, true
		}
	}
	return "", false
}

func (s *CommunityCommentService) DeleteComment(ctx context.Context, userID int, canModerate bool, postID int64, galgameID *int) *errors.AppError {
	if err := s.community.DeletePost(ctx, postID, int64(userID), canModerate); err != nil {
		return mapCommunityError(err)
	}
	if galgameID != nil {
		s.bumpCommentCount(*galgameID, -1)
	}
	feedParityDelete(s.db, feedTypeGalgameComment, postID)
	feedParityDeleteLegacyGalgame(s.db, postID)
	return nil
}

func (s *CommunityCommentService) bumpCommentCount(galgameID, delta int) {
	if err := s.db.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
		slog.Warn("community comment counter adjust failed (best-effort)", "galgame_id", galgameID, "delta", delta, "error", err)
	}
}

func (s *CommunityCommentService) ToggleLike(ctx context.Context, userID int, postID int64) (*LikeResult, *errors.AppError) {
	res, err := s.community.ToggleReaction(ctx, postID, communityclient.ReactionToggleRequest{
		UserID: int64(userID), Kind: communityclient.ReactionLike,
	})
	if err != nil {
		return nil, mapCommunityError(err)
	}

	localPresent, awardDelta := likeEffects(res.Added, res.AuthorID, userID)

	if localPresent {
		if e := s.posts.EnsureLike(postID, userID); e != nil {
			slog.Warn("galgame_post_like insert failed", "post_id", postID, "user_id", userID, "error", e)
		}
	} else {
		if e := s.posts.RemoveLike(postID, userID); e != nil {
			slog.Warn("galgame_post_like delete failed", "post_id", postID, "user_id", userID, "error", e)
		}
	}

	if awardDelta != 0 {
		ref := moemoepoint.Ref("galgame_post", int(postID))
		moemoepoint.Award(int(res.AuthorID), awardDelta, moemoepoint.ReasonLiked, ref, moemoepoint.KeyNonce(moemoepoint.ReasonLiked, ref))
	}

	lc := s.posts.CountLikes([]int64{postID})[postID]
	return &LikeResult{Liked: res.Added, LikeCount: lc}, nil
}

func likeEffects(added bool, authorID int64, userID int) (localPresent bool, awardDelta int) {
	self := authorID == int64(userID)
	if added {
		if self {
			return true, 0
		}
		return true, 1
	}
	if self {
		return false, 0
	}
	return false, -1
}

func (s *CommunityCommentService) mentionIDsForCreate(ctx context.Context, authorID int, content string) ([]int64, *errors.AppError) {
	ids := markdown.ExtractMentionIDs(content)
	if len(ids) == 0 {
		return nil, nil
	}
	known, err := s.userClient.Users(ctx, ids)
	return prepareMentionIDs(ids, authorID, known, err)
}

func prepareMentionIDs(ids []int, authorID int, known map[int]userclient.User, lookupErr error) ([]int64, *errors.AppError) {
	seen := make(map[int]bool, len(ids))
	kept := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || id == authorID || seen[id] {
			continue
		}
		seen[id] = true
		kept = append(kept, id)
	}
	if lookupErr == nil {
		filtered := kept[:0]
		for _, id := range kept {
			if _, ok := known[id]; ok {
				filtered = append(filtered, id)
			}
		}
		kept = filtered
	}
	if len(kept) > 20 {
		return nil, errors.ErrValidation("一条评论最多 @ 20 位用户")
	}
	out := make([]int64, len(kept))
	for i, id := range kept {
		out[i] = int64(id)
	}
	return out, nil
}

func newlyAddedMentionIDs(oldContent, newContent string, editorID int) []int {
	old := make(map[int]bool)
	for _, id := range markdown.ExtractMentionIDs(oldContent) {
		old[id] = true
	}
	var added []int
	seen := make(map[int]bool)
	for _, id := range markdown.ExtractMentionIDs(newContent) {
		if id <= 0 || id == editorID || old[id] || seen[id] {
			continue
		}
		seen[id] = true
		added = append(added, id)
	}
	return added
}

func (s *CommunityCommentService) FlagComment(ctx context.Context, userID int, postID int64, reason int, note string) *errors.AppError {
	if reason < communityclient.FlagReasonSpam || reason > communityclient.FlagReasonNsfwMislabel {
		return errors.ErrValidation("非法的举报理由")
	}
	if err := s.community.SubmitFlag(ctx, postID, communityclient.FlagRequest{
		FlaggerID: int64(userID), Reason: int32(reason), Note: note,
	}); err != nil {
		return mapCommunityError(err)
	}
	return nil
}

func (s *CommunityCommentService) Locate(legacyID int) (*LocateResult, *errors.AppError) {
	row, err := s.posts.FindMapByLegacyID(legacyID)
	if err != nil {
		return nil, errors.ErrInternal("查询失败")
	}
	if row == nil {
		return nil, errors.ErrNotFound("未找到对应评论")
	}
	return &LocateResult{PostID: row.PostID, ThreadID: row.ThreadID}, nil
}
