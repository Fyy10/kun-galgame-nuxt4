package service

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type createCtx struct {
	galgameID int
	ownerID   int
}

func (s *ResourceCommentService) CreateComment(ctx context.Context, src CommentSource, resourceID, userID int, content string, replyToPostID, targetUserID *int64) (*CommunityPostItem, *errors.AppError) {
	cc, appErr := s.resolveCreateCtx(src, resourceID)
	if appErr != nil {
		return nil, appErr
	}

	if s.commentAreaLocked(ctx, src, resourceID, userID) {
		return nil, errors.ErrForbidden("作答后才能参与这道题目的讨论")
	}

	content = markdown.NormalizeStoredContent(content)
	req := communityclient.CommentRequest{
		AnchorKind: communityclient.AnchorSiteResource, AnchorID: src.anchorID(resourceID),
		ContentRating: communityclient.RatingAll, AuthorID: int64(userID), Body: content,
	}
	if replyToPostID != nil {
		req.ReplyToPostID = *replyToPostID
	}
	if targetUserID != nil {
		req.TargetUserID = *targetUserID
	}
	res, err := s.community.CommentOnAnchor(ctx, req)
	if err != nil {
		return nil, mapCommunityError(err)
	}
	post := &res.Post

	s.afterCreate(ctx, src, resourceID, cc, userID, content, post)

	items := s.renderPosts(ctx, userID, []communityclient.PostView{*post})
	if len(items) == 0 {
		return buildCommunityItem(*post, 0, s.userClient.Hydrate(ctx, []int{userID})[userID], 0, false), nil
	}
	return items[0], nil
}

func (s *ResourceCommentService) resolveCreateCtx(src CommentSource, resourceID int) (createCtx, *errors.AppError) {
	switch src.key {
	case sourceRating.key:
		gid := s.ratingGalgameID(resourceID)
		if gid == 0 {
			return createCtx{}, errors.ErrNotFound("未找到这个评分")
		}
		return createCtx{galgameID: gid}, nil
	case sourceToolset.key:
		owner := s.toolsetOwner(resourceID)
		if owner == 0 {
			return createCtx{}, errors.ErrNotFound("未找到该工具")
		}
		return createCtx{ownerID: owner}, nil
	case sourceResource.key:
		owner := s.galgameResourceOwner(resourceID)
		if owner == 0 {
			return createCtx{}, errors.ErrNotFound("未找到该资源")
		}
		return createCtx{ownerID: owner}, nil
	case sourceQuiz.key:
		author := s.quizAuthor(resourceID)
		if author == 0 {
			return createCtx{}, errors.ErrNotFound("未找到该题目")
		}
		return createCtx{ownerID: author}, nil
	default:
		return createCtx{}, nil
	}
}

func shouldNotifyOwner(src CommentSource, senderID, ownerID int, isReply, hasPref, lookupFailed bool) bool {
	switch src.key {
	case sourceToolset.key, sourceResource.key, sourceQuiz.key:
	default:
		return false
	}
	if isReply || ownerID <= 0 || ownerID == senderID {
		return false
	}
	if lookupFailed {
		return true
	}
	return !hasPref
}

func (s *ResourceCommentService) ownerHasPreference(ctx context.Context, ownerID int, threadID int64, src CommentSource, resourceID int) (has bool, failed bool) {
	if threadID > 0 {
		st, err := s.community.ThreadStates(ctx, int64(ownerID), []int64{threadID})
		if err != nil {
			return false, true
		}
		if len(st.States) > 0 {
			return true, false
		}
	}
	as, err := s.community.AnchorStates(ctx, int64(ownerID), []communityclient.AnchorRef{{
		AnchorKind: communityclient.AnchorSiteResource,
		AnchorID:   src.anchorID(resourceID),
	}})
	if err != nil {
		return false, true
	}
	return len(as.States) > 0, false
}

func (s *ResourceCommentService) afterCreate(ctx context.Context, src CommentSource, resourceID int, cc createCtx, userID int, content string, post *communityclient.PostView) {
	switch src.key {
	case sourceRating.key:
		s.feedUpsert(src.feedType, post.ID, userID, content, src.pageLink(resourceID), false, post.CreatedAt)

	case sourceWebsite.key:
		s.bumpWebsiteCommentCount(resourceID, 1)
		slug, nsfw := s.websiteMeta(resourceID)
		s.feedUpsert(src.feedType, post.ID, userID, content, "/website/"+slug, nsfw, post.CreatedAt)

	case sourceToolset.key:
		s.bumpToolsetCommentCount(resourceID, 1)
		s.notifyOwnerIfNeeded(ctx, src, resourceID, cc.ownerID, userID, content, post)
		s.feedUpsert(src.feedType, post.ID, userID, content, src.pageLink(resourceID), false, post.CreatedAt)

	case sourceResource.key:
		s.bumpCountColumn("galgame_resource", resourceID, 1)
		s.notifyOwnerIfNeeded(ctx, src, resourceID, cc.ownerID, userID, content, post)
		s.feedUpsert(src.feedType, post.ID, userID, content, src.pageLink(resourceID), false, post.CreatedAt)

	case sourceQuiz.key:
		s.bumpCountColumn("galgame_quiz", resourceID, 1)
		s.notifyOwnerIfNeeded(ctx, src, resourceID, cc.ownerID, userID, content, post)
		s.feedUpsert(src.feedType, post.ID, userID, content, src.pageLink(resourceID), false, post.CreatedAt)
	}
}

func (s *ResourceCommentService) notifyOwnerIfNeeded(ctx context.Context, src CommentSource, resourceID, ownerID, userID int, content string, post *communityclient.PostView) {
	isReply := post.ReplyToPostID != 0
	if !shouldNotifyOwner(src, userID, ownerID, isReply, false, false) {
		return
	}
	hasPref, failed := s.ownerHasPreference(ctx, ownerID, post.ThreadID, src, resourceID)
	if !shouldNotifyOwner(src, userID, ownerID, isReply, hasPref, failed) {
		return
	}
	s.notifyDeduped(userID, ownerID, "commented", content, src.pageLink(resourceID))
}

func (s *ResourceCommentService) DeleteComment(ctx context.Context, src CommentSource, resourceID, userID int, canModerate bool, postID int64) *errors.AppError {
	elevated := canModerate
	if !elevated {
		if owner := s.resourceOwner(src, resourceID); owner != 0 && owner == userID {
			ok, verr := s.postInResourceThread(ctx, src, resourceID, postID)
			if verr != nil {
				return mapCommunityError(verr)
			}
			if !ok {
				return errors.ErrForbidden("您没有权限删除此评论")
			}
			elevated = true
		}
	}

	if err := s.community.DeletePost(ctx, postID, int64(userID), elevated); err != nil {
		return mapCommunityError(err)
	}

	switch src.key {
	case sourceWebsite.key:
		s.bumpWebsiteCommentCount(resourceID, -1)
	case sourceToolset.key:
		s.bumpToolsetCommentCount(resourceID, -1)
	case sourceResource.key:
		s.bumpCountColumn("galgame_resource", resourceID, -1)
	case sourceQuiz.key:
		s.bumpCountColumn("galgame_quiz", resourceID, -1)
	}
	s.feedDelete(src, postID)
	return nil
}

func (s *ResourceCommentService) resourceOwner(src CommentSource, resourceID int) int {
	switch src.key {
	case sourceRating.key:
		gid := s.ratingGalgameID(resourceID)
		if gid == 0 {
			return 0
		}
		return s.galgameOwner(gid)
	case sourceToolset.key:
		return s.toolsetOwner(resourceID)
	case sourceResource.key:
		return s.galgameResourceOwner(resourceID)
	case sourceQuiz.key:
		return s.quizAuthor(resourceID)
	default:
		return 0
	}
}

func (s *ResourceCommentService) postInResourceThread(ctx context.Context, src CommentSource, resourceID int, postID int64) (bool, error) {
	cursor := ""
	for {
		page, err := s.community.GetComments(ctx, communityclient.AnchorSiteResource, src.anchorID(resourceID), cursor, "50")
		if err != nil {
			return false, err
		}
		if page.Thread == nil {
			return false, nil
		}
		if containsPost(page.Posts, postID) {
			return true, nil
		}
		if page.NextCursor == "" {
			return false, nil
		}
		cursor = page.NextCursor
	}
}

func containsPost(posts []communityclient.PostView, postID int64) bool {
	for _, p := range posts {
		if p.ID == postID {
			return true
		}
	}
	return false
}

func (s *ResourceCommentService) ratingGalgameID(ratingID int) int {
	var gid int
	s.db.Table("galgame_rating").Select("galgame_id").Where("id = ?", ratingID).Scan(&gid)
	return gid
}

// galgame's owner column is creator_user_id — see RatingRepository.FindGalgameOwner.
func (s *ResourceCommentService) galgameOwner(galgameID int) int {
	var uid int
	s.db.Table("galgame").Select("creator_user_id").Where("id = ?", galgameID).Scan(&uid)
	return uid
}

func (s *ResourceCommentService) toolsetOwner(toolsetID int) int {
	var uid int
	s.db.Table("galgame_toolset").Select("user_id").Where("id = ?", toolsetID).Scan(&uid)
	return uid
}

func (s *ResourceCommentService) galgameResourceOwner(resourceID int) int {
	var uid int
	s.db.Table("galgame_resource").Select("user_id").Where("id = ?", resourceID).Scan(&uid)
	return uid
}

func (s *ResourceCommentService) quizAuthor(quizID int) int {
	var uid int
	s.db.Table("galgame_quiz").Select("user_id").Where("id = ?", quizID).Scan(&uid)
	return uid
}

func (s *ResourceCommentService) websiteMeta(websiteID int) (slug string, nsfw bool) {
	var row struct {
		URL      string `gorm:"column:url"`
		AgeLimit string `gorm:"column:age_limit"`
	}
	res := s.db.Table("galgame_website").Select("url, age_limit").Where("id = ?", websiteID).Limit(1).Find(&row)
	if res.Error != nil || res.RowsAffected == 0 {
		return "", false
	}
	return row.URL, row.AgeLimit != "all"
}

func (s *ResourceCommentService) bumpWebsiteCommentCount(websiteID, delta int) {
	if err := s.db.Table("galgame_website").Where("id = ?", websiteID).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
		slog.Warn("website comment counter adjust failed (best-effort)", "website_id", websiteID, "delta", delta, "error", err)
	}
}

func (s *ResourceCommentService) bumpToolsetCommentCount(toolsetID, delta int) {
	if err := s.db.Table("galgame_toolset").Where("id = ?", toolsetID).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
		slog.Warn("toolset comment counter adjust failed (best-effort)", "toolset_id", toolsetID, "delta", delta, "error", err)
	}
}

func (s *ResourceCommentService) bumpCountColumn(table string, resourceID, delta int) {
	if err := s.db.Table(table).Where("id = ?", resourceID).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
		slog.Warn("comment counter adjust failed (best-effort)",
			"table", table, "resource_id", resourceID, "delta", delta, "error", err)
	}
}

func (s *ResourceCommentService) notifyDeduped(senderID, receiverID int, msgType, content, link string) {
	preview := markdown.ToPlainText(content, constants.TextPreviewLength)
	var count int64
	s.db.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND content = ? AND link = ?",
			senderID, receiverID, msgType, preview, link).
		Count(&count)
	if count > 0 {
		return
	}
	s.notifyCreate(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: preview, Link: link, Status: "unread",
	})
}

func (s *ResourceCommentService) notifyCreate(msg *msgModel.Message) {
	// afterCreate runs once the comment is already committed upstream, so a lost
	// notification must not fail the request — but it must not be invisible either.
	if err := s.db.Create(msg).Error; err != nil {
		slog.Warn("notification insert failed (best-effort)",
			"type", msg.Type, "receiver_id", msg.ReceiverID, "link", msg.Link, "error", err)
	}
}

func (s *ResourceCommentService) feedUpsert(feedType string, postID int64, userID int, content, link string, nsfw bool, createdAt string) {
	feedParityUpsert(s.db, feedType, postID, userID, 0, content, link, nsfw, createdAt)
}

func (s *ResourceCommentService) feedDelete(src CommentSource, postID int64) {
	feedParityDelete(s.db, src.feedType, postID)
	feedParityDeleteLegacyResource(s.db, src, postID)
}
