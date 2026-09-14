package service

import (
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

const MaxDraftsPerUser = 30

type DraftService struct {
	draftRepo *repository.TopicDraftRepository
}

func NewDraftService(draftRepo *repository.TopicDraftRepository) *DraftService {
	return &DraftService{draftRepo: draftRepo}
}

func (s *DraftService) Save(userID int, req *dto.SaveTopicDraftRequest) (int, *errors.AppError) {
	req.Content = markdown.NormalizeStoredContent(req.Content)
	if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Content) == "" {
		return 0, errors.ErrBadRequest("草稿的标题和正文不能都为空")
	}

	count, err := s.draftRepo.CountByUser(userID)
	if err != nil {
		return 0, errors.ErrInternal("读取草稿数量失败")
	}
	if count >= MaxDraftsPerUser {
		return 0, errors.ErrBadRequest("草稿数量已达上限，请先删除一些草稿再保存")
	}

	draft := &model.TopicDraft{
		UserID:      userID,
		Title:       req.Title,
		Content:     req.Content,
		Category:    req.Category,
		Sections:    model.StringSlice(req.Sections),
		CoverImages: model.ImageTokens(req.CoverImages),
		IsNSFW:      req.IsNSFW,
	}
	if err := s.draftRepo.Create(draft); err != nil {
		return 0, errors.ErrInternal("保存草稿失败")
	}
	return draft.ID, nil
}

func (s *DraftService) List(userID int) ([]dto.TopicDraftListItem, *errors.AppError) {
	rows, err := s.draftRepo.ListByUser(userID)
	if err != nil {
		return nil, errors.ErrInternal("读取草稿列表失败")
	}
	items := make([]dto.TopicDraftListItem, len(rows))
	for i, r := range rows {
		items[i] = dto.TopicDraftListItem{
			ID:      r.ID,
			Title:   r.Title,
			Summary: r.Summary,
			Updated: r.Updated,
		}
	}
	return items, nil
}

func (s *DraftService) Get(userID, id int) (*dto.TopicDraftDetail, *errors.AppError) {
	d, err := s.draftRepo.GetByIDForUser(id, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound("草稿不存在")
		}
		return nil, errors.ErrInternal("读取草稿失败")
	}
	return &dto.TopicDraftDetail{
		ID:          d.ID,
		Title:       d.Title,
		Content:     d.Content,
		Category:    d.Category,
		Sections:    []string(d.Sections),
		IsNSFW:      d.IsNSFW,
		CoverImages: []string(d.CoverImages),
		Updated:     d.UpdatedAt,
	}, nil
}

func (s *DraftService) Delete(userID, id int) *errors.AppError {
	n, err := s.draftRepo.DeleteForUser(id, userID)
	if err != nil {
		return errors.ErrInternal("删除草稿失败")
	}
	if n == 0 {
		return errors.ErrNotFound("草稿不存在")
	}
	return nil
}
