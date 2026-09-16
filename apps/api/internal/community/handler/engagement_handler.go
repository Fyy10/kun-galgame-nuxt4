package handler

import (
	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/community/engagement"
	"kun-galgame-api/internal/middleware"
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

func (h *EngagementHandler) WallRead(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req struct {
		AnchorKind int32  `json:"anchor_kind" validate:"min=0,max=4"`
		AnchorID   string `json:"anchor_id" validate:"required,max=64"`
		ThreadID   int64  `json:"thread_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	state, appErr := h.service.WallRead(c.Context(), user.ID, anchor.Ref{Kind: req.AnchorKind, ID: req.AnchorID}, req.ThreadID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *EngagementHandler) WallFollow(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req struct {
		AnchorKind int32  `json:"anchor_kind" validate:"min=0,max=4"`
		AnchorID   string `json:"anchor_id" validate:"required,max=64"`
		Following  bool   `json:"following"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	state, appErr := h.service.WallFollow(c.Context(), user.ID, anchor.Ref{Kind: req.AnchorKind, ID: req.AnchorID}, req.Following)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *EngagementHandler) Following(c fiber.Ctx) error {
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
	if req.Limit == 0 {
		req.Limit = 30
	}

	res, appErr := h.service.Following(c.Context(), user.ID, req.Cursor, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}
