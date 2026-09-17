package handler

import (
	"strconv"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type UserPermissionHandler struct {
	svc *service.UserPermissionService
}

func NewUserPermissionHandler(svc *service.UserPermissionService) *UserPermissionHandler {
	return &UserPermissionHandler{svc: svc}
}

func (h *UserPermissionHandler) GetView(c fiber.Ctx) error {
	uid, ok := parseUIDParam(c.Params("uid"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的用户 ID"))
	}
	view, appErr := h.svc.View(c.Context(), uid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, view)
}

func (h *UserPermissionHandler) Replace(c fiber.Ctx) error {
	operator, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	uid, ok := parseUIDParam(c.Params("uid"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的用户 ID"))
	}
	var req dto.ReplaceUserOverridesRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	view, appErr := h.svc.ReplaceOverrides(c.Context(), operator.ID, operator.Roles, uid, req.Overrides)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, view)
}

func (h *UserPermissionHandler) GetMine(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if user.ViaBearer() {
		return response.OK(c, dto.MyPermissionsResponse{Permissions: []string{}})
	}
	return response.OK(c, h.svc.MyPermissions(user.ID, user.Roles))
}

func parseUIDParam(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
