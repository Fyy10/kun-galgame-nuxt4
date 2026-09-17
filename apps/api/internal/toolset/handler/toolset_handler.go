package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ToolsetHandler struct {
	toolsetService *service.ToolsetService
}

func NewToolsetHandler(toolsetService *service.ToolsetService) *ToolsetHandler {
	return &ToolsetHandler{toolsetService: toolsetService}
}

func (h *ToolsetHandler) GetList(c fiber.Ctx) error {
	var req dto.ToolsetListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 24
	}

	items, total := h.toolsetService.GetList(c.Context(), &req)
	return response.Paginated(c, items, total)
}

func (h *ToolsetHandler) GetUserToolsets(c fiber.Ctx) error {
	userID := fiber.Params[int](c, "id")
	if userID <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.ToolsetListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 24
	}
	req.UserID = userID

	items, total := h.toolsetService.GetList(c.Context(), &req)
	return response.Paginated(c, items, total)
}

func (h *ToolsetHandler) Create(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateToolsetRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	toolset, appErr := h.toolsetService.Create(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, toolset)
}

func (h *ToolsetHandler) GetDetail(c fiber.Ctx) error {
	id := fiber.Params[int](c, "id")
	if id <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	detail, appErr := h.toolsetService.GetDetail(c.Context(), id)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

func (h *ToolsetHandler) Update(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id := fiber.Params[int](c, "id")
	if id <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.UpdateToolsetRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.toolsetService.Update(c.Context(), user.ID, user.Can(perm.ToolsetEditAny), id, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "工具更新成功")
}

func (h *ToolsetHandler) Delete(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id := fiber.Params[int](c, "id")
	if id <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	if appErr := h.toolsetService.Delete(user.ID, user.Can(perm.ToolsetDeleteAny), id); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "工具已删除")
}
