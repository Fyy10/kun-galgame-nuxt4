package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type QuizHandler struct {
	quizService *service.QuizService
}

func NewQuizHandler(quizService *service.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

func (h *QuizHandler) GetAllQuizzes(c fiber.Ctx) error {
	var req dto.QuizListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.quizService.GetAllQuizzes(
		c.Context(), &req, utils.IsSFW(c), optionalUID(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *QuizHandler) GetMyAnswered(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.QuizListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.quizService.GetMyAnswered(c.Context(), user.ID, req.Page, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *QuizHandler) GetMyFavorites(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"favorited": h.quizService.GetMyFavorites(user.ID)})
}

func (h *QuizHandler) SearchGalgames(c fiber.Ctx) error {
	var req dto.QuizGalgameSearchRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	options := h.quizService.SearchGalgameOptions(
		c.Context(), req.Keywords, utils.IsSFW(c),
	)
	return response.OK(c, options)
}

func (h *QuizHandler) GetQuizPlay(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的题目 ID"))
	}
	detail, appErr := h.quizService.GetQuizPlay(c.Context(), id, optionalUID(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

func (h *QuizHandler) CreateQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateQuizRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	created, appErr := h.quizService.CreateQuiz(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, created)
}

func (h *QuizHandler) AnswerQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.AnswerQuizRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.quizService.AnswerQuiz(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

func (h *QuizHandler) RateQuizQuality(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.RateQuizQualityRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.quizService.RateQuizQuality(user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

func (h *QuizHandler) DeleteQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteQuizRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.quizService.DeleteQuiz(user.ID, user.Can(perm.QuizDeleteAny), req.QuizID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "题目已删除")
}

func (h *QuizHandler) ToggleQuizFavorite(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的题目 ID"))
	}
	if appErr := h.quizService.ToggleQuizFavorite(user.ID, id); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

func (h *QuizHandler) UpdateQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateQuizRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	regraded, appErr := h.quizService.UpdateQuiz(c.Context(), user.ID, user.Can(perm.QuizEditAny), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"regraded": regraded})
}

func (h *QuizHandler) GetQuizForEdit(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的题目 ID"))
	}
	data, appErr := h.quizService.GetQuizForEdit(
		c.Context(), user.ID, user.Can(perm.QuizEditAny), id,
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, data)
}

func (h *QuizHandler) GetQuizAnswers(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的题目 ID"))
	}
	viewerID := 0
	if u := middleware.GetUser(c); u != nil {
		viewerID = u.ID
	}
	return response.OK(c, h.quizService.GetQuizAnswers(c.Context(), id, viewerID))
}
