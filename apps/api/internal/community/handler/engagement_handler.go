package handler

import (
	"strconv"

	"kun-galgame-api/internal/community/engagement"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type EngagementHandler struct {
	service *engagement.Service
}

func NewEngagementHandler(service *engagement.Service) *EngagementHandler {
	return &EngagementHandler{service: service}
}

func (h *EngagementHandler) MarkRead(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	threadID, appErr := threadIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	state, appErr := h.service.MarkRead(c.Context(), user.ID, threadID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *EngagementHandler) SetNotification(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	threadID, appErr := threadIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req struct {
		Level int32 `json:"level" validate:"min=0,max=3"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	state, appErr := h.service.SetLevel(c.Context(), user.ID, threadID, req.Level)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *EngagementHandler) Unread(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req struct {
		Cursor string `query:"cursor" validate:"omitempty,max=256"`
		Limit  int    `query:"limit" validate:"omitempty,min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.service.Unread(c.Context(), user.ID, req.Cursor, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}

func (h *EngagementHandler) UnreadCount(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"total": h.service.Count(c.Context(), user.ID)})
}

func threadIDParam(c fiber.Ctx) (int64, *errors.AppError) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.ErrBadRequest("评论区 ID 不正确")
	}
	return id, nil
}
