package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type TopicHandler struct {
	topicService *service.TopicService
}

func NewTopicHandler(topicService *service.TopicService) *TopicHandler {
	return &TopicHandler{topicService: topicService}
}

func (h *TopicHandler) MyInteractions(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.topicService.GetMyInteractions(user.ID))
}

func (h *TopicHandler) GetResourceList(c fiber.Ctx) error {
	var req dto.ListTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if req.SortField == "" {
		req.SortField = "status_update_time"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	isNSFW := !utils.IsSFW(c)
	items, _, appErr := h.topicService.GetResourceList(c.Context(), &req, isNSFW, middleware.GetUser(c) != nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, items)
}
