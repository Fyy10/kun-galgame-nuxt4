package handler

import (
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ResourceCommentHandler struct {
	service *service.ResourceCommentService
}

func NewResourceCommentHandler(svc *service.ResourceCommentService) *ResourceCommentHandler {
	return &ResourceCommentHandler{service: svc}
}

func (h *ResourceCommentHandler) RegisterReads(optAuth fiber.Router) {
	optAuth.Get("/galgame-rating/:id/comments", h.RatingList)
	optAuth.Get("/website/:domain/comments", h.WebsiteList)
	optAuth.Get("/toolset/:id/comments", h.ToolsetList)
	optAuth.Get("/galgame-resource/:id/comments", h.ResourceList)
	optAuth.Get("/galgame-quiz/:id/comments", h.QuizList)
}

func (h *ResourceCommentHandler) RegisterWrites(authed fiber.Router) {
	authed.Post("/galgame-rating/:id/comments", h.RatingCreate)
	authed.Delete("/galgame-rating/:id/comments/:postId", h.RatingDelete)
	authed.Post("/website/:domain/comments", h.WebsiteCreate)
	authed.Delete("/website/:domain/comments/:postId", h.WebsiteDelete)
	authed.Post("/toolset/:id/comments", h.ToolsetCreate)
	authed.Delete("/toolset/:id/comments/:postId", h.ToolsetDelete)
	authed.Post("/galgame-resource/:id/comments", h.ResourceCreate)
	authed.Delete("/galgame-resource/:id/comments/:postId", h.ResourceDelete)
	authed.Post("/galgame-quiz/:id/comments", h.QuizCreate)
	authed.Delete("/galgame-quiz/:id/comments/:postId", h.QuizDelete)
}

func (h *ResourceCommentHandler) list(c fiber.Ctx, src service.CommentSource, resourceID int) error {
	var req struct {
		Cursor string `query:"cursor"`
		Limit  int    `query:"limit" validate:"omitempty,min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.service.GetComments(c.Context(), src, resourceID, optionalUID(c), req.Cursor, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *ResourceCommentHandler) remove(c fiber.Ctx, src service.CommentSource, resourceID int, modPerm perm.Permission) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	postID, ok := parsePositive64(c.Params("postId"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}
	if appErr := h.service.DeleteComment(c.Context(), src, resourceID, user.ID, user.Can(modPerm), postID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论已删除")
}

func (h *ResourceCommentHandler) RatingList(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评分 ID"))
	}
	return h.list(c, service.SourceRating(), id)
}

func (h *ResourceCommentHandler) RatingCreate(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评分 ID"))
	}
	var req struct {
		Content      string `json:"content" validate:"required,min=1,max=1314"`
		TargetUserID int64  `json:"target_user_id" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	target := req.TargetUserID
	item, appErr := h.service.CreateComment(c.Context(), service.SourceRating(), id, user.ID, req.Content, nil, &target)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

func (h *ResourceCommentHandler) RatingDelete(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评分 ID"))
	}
	return h.remove(c, service.SourceRating(), id, perm.CommentRatingDelete)
}

func (h *ResourceCommentHandler) WebsiteList(c fiber.Ctx) error {
	id, ok := websiteID(c)
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的网站 ID"))
	}
	return h.list(c, service.SourceWebsite(), id)
}

func (h *ResourceCommentHandler) WebsiteCreate(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req struct {
		Content       string `json:"content" validate:"required,min=1,max=1007"`
		WebsiteID     int    `json:"website_id" validate:"required,min=1"`
		ReplyToPostID *int64 `json:"reply_to_post_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.CreateComment(c.Context(), service.SourceWebsite(), req.WebsiteID, user.ID, req.Content, req.ReplyToPostID, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

func (h *ResourceCommentHandler) WebsiteDelete(c fiber.Ctx) error {
	id, ok := websiteID(c)
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的网站 ID"))
	}
	return h.remove(c, service.SourceWebsite(), id, perm.CommentWebsiteDelete)
}

func websiteID(c fiber.Ctx) (int, bool) {
	return parsePositive(c.Query("website_id"))
}

func (h *ResourceCommentHandler) ToolsetList(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的工具 ID"))
	}
	return h.list(c, service.SourceToolset(), id)
}

func (h *ResourceCommentHandler) ToolsetCreate(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的工具 ID"))
	}
	var req struct {
		Content       string `json:"content" validate:"required,min=1,max=1007"`
		ReplyToPostID *int64 `json:"reply_to_post_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.CreateComment(c.Context(), service.SourceToolset(), id, user.ID, req.Content, req.ReplyToPostID, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

func (h *ResourceCommentHandler) ToolsetDelete(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的工具 ID"))
	}
	return h.remove(c, service.SourceToolset(), id, perm.CommentToolsetDelete)
}

func (h *ResourceCommentHandler) ResourceList(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的资源 ID"))
	}
	return h.list(c, service.SourceResource(), id)
}

func (h *ResourceCommentHandler) ResourceCreate(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的资源 ID"))
	}
	var req struct {
		Content       string `json:"content" validate:"required,min=1,max=1007"`
		ReplyToPostID *int64 `json:"reply_to_post_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.CreateComment(c.Context(), service.SourceResource(), id, user.ID, req.Content, req.ReplyToPostID, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

func (h *ResourceCommentHandler) ResourceDelete(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的资源 ID"))
	}
	return h.remove(c, service.SourceResource(), id, perm.CommentResourceDelete)
}

func (h *ResourceCommentHandler) QuizList(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的题目 ID"))
	}
	return h.list(c, service.SourceQuiz(), id)
}

func (h *ResourceCommentHandler) QuizCreate(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的题目 ID"))
	}
	var req struct {
		Content       string `json:"content" validate:"required,min=1,max=1007"`
		ReplyToPostID *int64 `json:"reply_to_post_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.CreateComment(c.Context(), service.SourceQuiz(), id, user.ID, req.Content, req.ReplyToPostID, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

func (h *ResourceCommentHandler) QuizDelete(c fiber.Ctx) error {
	id, ok := parsePositive(c.Params("id"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的题目 ID"))
	}
	return h.remove(c, service.SourceQuiz(), id, perm.CommentQuizDelete)
}
