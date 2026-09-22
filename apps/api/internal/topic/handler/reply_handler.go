package handler

import (
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type ReplyHandler struct {
	replyService *service.ReplyService
}

func NewReplyHandler(replyService *service.ReplyService) *ReplyHandler {
	return &ReplyHandler{replyService: replyService}
}

func (h *ReplyHandler) GetReplyLocate(c fiber.Ctx) error {
	topicID, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的话题 ID"))
	}
	floor, _ := strconv.Atoi(c.Query("reply"))
	commentID, _ := strconv.Atoi(c.Query("comment"))
	if floor <= 0 && commentID <= 0 {
		return response.Error(c, errors.ErrBadRequest("缺少 reply 或 comment 参数"))
	}

	res, appErr := h.replyService.LocateReply(topicID, floor, commentID, 30, middleware.GetUser(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, res)
}
