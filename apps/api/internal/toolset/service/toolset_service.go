package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/internal/trust/gate"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type ToolsetService struct {
	toolsetRepo      *repository.ToolsetRepository
	resourceRepo     *repository.ResourceRepository
	practicalityRepo *repository.PracticalityRepository
	s3               *storage.S3Client
	userClient       *userclient.Client
	check            *gate.CheckService
	scan             *gate.ScanService

	practicalitySvc *PracticalityService
	commentSvc      *CommentService
}

func NewToolsetService(
	toolsetRepo *repository.ToolsetRepository,
	resourceRepo *repository.ResourceRepository,
	practicalityRepo *repository.PracticalityRepository,
	s3 *storage.S3Client,
	userClient *userclient.Client,
	practicalitySvc *PracticalityService,
	commentSvc *CommentService,
	check *gate.CheckService,
	scan *gate.ScanService,
) *ToolsetService {
	return &ToolsetService{
		toolsetRepo:      toolsetRepo,
		resourceRepo:     resourceRepo,
		practicalityRepo: practicalityRepo,
		s3:               s3,
		userClient:       userClient,
		check:            check,
		scan:             scan,
		practicalitySvc:  practicalitySvc,
		commentSvc:       commentSvc,
	}
}

func toolsetModerationText(name, description string, aliases []string, version string) string {
	parts := make([]string, 0, 3+len(aliases))
	parts = append(parts, name, description)
	parts = append(parts, aliases...)
	parts = append(parts, version)
	return gate.ComposeText(parts...)
}

func userBriefFromClient(u userclient.User) userModel.UserBrief {
	return userModel.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

func (s *ToolsetService) GetList(ctx context.Context, req *dto.ToolsetListRequest) ([]dto.ToolsetCard, int64) {
	filters := repository.ListFilters{
		Type:     req.Type,
		Language: req.Language,
		Platform: req.Platform,
		Version:  req.Version,
		UserID:   req.UserID,
		Query:    req.Query,
	}
	total := s.toolsetRepo.CountFiltered(filters)

	opts := repository.ListOptions{
		SortField: allowedSortField(req.SortField),
		SortOrder: sortOrder(req.SortOrder),
		Offset:    (req.Page - 1) * req.Limit,
		Limit:     req.Limit,
	}
	toolsets := s.toolsetRepo.ListFiltered(filters, opts)

	toolsetIDs := make([]int, len(toolsets))
	userIDs := make([]int, len(toolsets))
	for i, t := range toolsets {
		toolsetIDs[i] = t.ID
		userIDs[i] = t.UserID
	}

	avgMap := s.practicalityRepo.AveragesForToolsets(toolsetIDs)
	dlMap := s.resourceRepo.DownloadSumsForToolsets(toolsetIDs)
	ccMap := make(map[int]int, len(toolsets))
	for _, t := range toolsets {
		ccMap[t.ID] = t.CommentCount
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.ToolsetCard, 0, len(toolsets))
	for _, t := range toolsets {
		if !userclient.IsRenderable(userMap[t.UserID]) {
			continue
		}
		cards = append(cards, toolsetCardFromRow(t, userMap, avgMap, dlMap, ccMap))
	}

	return cards, total
}

func (s *ToolsetService) Create(
	ctx context.Context,
	userID int,
	req *dto.CreateToolsetRequest,
) (*dto.CreatedToolsetResponse, *errors.AppError) {
	req.Description = markdown.NormalizeStoredContent(req.Description)
	moderationText := toolsetModerationText(req.Name, req.Description, req.Aliases, req.Version)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return nil, gate.ErrContentBlocked()
	}

	homepageJSON, _ := json.Marshal(req.Homepage)

	var toolset model.GalgameToolset
	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		toolset = model.GalgameToolset{
			Name:        req.Name,
			Description: req.Description,
			Type:        req.Type,
			Language:    req.Language,
			Platform:    req.Platform,
			Homepage:    homepageJSON,
			Version:     req.Version,
			UserID:      userID,
		}
		if err := s.toolsetRepo.Create(tx, &toolset); err != nil {
			return err
		}

		if err := s.toolsetRepo.ReplaceAliases(tx, toolset.ID, trimNonEmpty(req.Aliases)); err != nil {
			return err
		}

		if err := s.toolsetRepo.AddContributor(tx, toolset.ID, userID); err != nil {
			return err
		}

		adjustMoemoepoint(tx, userID, 3,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("toolset", toolset.ID),
			moemoepoint.Key("toolset_create", strconv.Itoa(toolset.ID)))

		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建工具失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindToolset, "subject_id", toolset.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindToolset, strconv.Itoa(toolset.ID), moderationText, int64(userID))

	return &toolset, nil
}

func (s *ToolsetService) GetDetail(ctx context.Context, id int) (*dto.ToolsetDetailResponse, *errors.AppError) {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	descriptionHTML := markdown.Render(toolset.Description)
	aliases := s.toolsetRepo.FindAliases(id)

	practicality := s.practicalitySvc.Summary(id)
	downloadSum := s.resourceRepo.DownloadSum(id)
	commentCount := int64(toolset.CommentCount)
	comments := s.commentSvc.GetLatestForDetail(ctx, id, 5)
	contributorIDs := s.toolsetRepo.FindContributorIDs(id)
	resources := s.resourceRepo.FindByToolset(id)

	allUIDs := append([]int{toolset.UserID}, contributorIDs...)
	userMap := s.userClient.Hydrate(ctx, allUIDs)
	if !userclient.IsRenderable(userMap[toolset.UserID]) {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	go s.toolsetRepo.IncrementView(id)

	user := userBriefFromClient(userMap[toolset.UserID])
	contributors := make([]userModel.UserBrief, 0, len(contributorIDs))
	for _, userID := range contributorIDs {
		contributors = append(contributors, userBriefFromClient(userMap[userID]))
	}

	homepage := []string{}
	if len(toolset.Homepage) > 0 {
		_ = json.Unmarshal(toolset.Homepage, &homepage)
		if homepage == nil {
			homepage = []string{}
		}
	}

	aliasNames := make([]string, len(aliases))
	for i, a := range aliases {
		aliasNames[i] = a.Name
	}

	resourceItems := make([]dto.ToolsetResourceItem, len(resources))
	for i, r := range resources {
		resourceItems[i] = dto.ToolsetResourceItem{
			ID: r.ID, Type: r.Type, Size: r.Size,
			Download: r.Download, Status: r.Status,
		}
	}

	var avg *float64
	practicalityCount := int64(0)
	for _, c := range practicality.Counts {
		practicalityCount += c
	}
	if practicalityCount > 0 {
		v := practicality.Avg
		avg = &v
	}

	return &dto.ToolsetDetailResponse{
		ID:                 toolset.ID,
		Name:               toolset.Name,
		ContentMarkdown:    toolset.Description,
		ContentHTML:        descriptionHTML,
		Type:               toolset.Type,
		Platform:           toolset.Platform,
		Language:           toolset.Language,
		Version:            toolset.Version,
		Homepage:           homepage,
		View:               toolset.View,
		Download:           downloadSum,
		User:               user,
		Aliases:            aliasNames,
		PracticalityAvg:    avg,
		PracticalityCount:  practicalityCount,
		RatingCounts:       practicality.Counts,
		ResourceUpdateTime: toolset.ResourceUpdateTime,
		Resource:           resourceItems,
		Edited:             toolset.Edited,
		Created:            toolset.CreatedAt,
		Updated:            toolset.UpdatedAt,
		CommentCount:       commentCount,
		CommentPreview:     comments,
		Contributors:       contributors,
	}, nil
}

func (s *ToolsetService) Update(
	ctx context.Context,
	userID int, canModerate bool, id int,
	req *dto.UpdateToolsetRequest,
) *errors.AppError {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return errors.ErrNotFound("未找到该工具")
	}
	if toolset.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限编辑此工具")
	}

	req.Description = markdown.NormalizeStoredContent(req.Description)
	moderationText := toolsetModerationText(req.Name, req.Description, req.Aliases, req.Version)
	authorID := int64(toolset.UserID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return gate.ErrContentBlocked()
	}

	homepageJSON, _ := json.Marshal(req.Homepage)
	now := time.Now()

	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.toolsetRepo.UpdateFields(tx, id, map[string]any{
			"name":        req.Name,
			"description": req.Description,
			"type":        req.Type,
			"language":    req.Language,
			"platform":    req.Platform,
			"homepage":    homepageJSON,
			"version":     req.Version,
			"edited":      now,
		}); err != nil {
			return err
		}
		return s.toolsetRepo.ReplaceAliases(tx, id, trimNonEmpty(req.Aliases))
	})
	if txErr != nil {
		return errors.ErrInternal("更新工具失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindToolset, "subject_id", id, "author_id", toolset.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindToolset, strconv.Itoa(id), moderationText, int64(toolset.UserID))
	return nil
}

func (s *ToolsetService) Delete(userID int, canModerate bool, id int) *errors.AppError {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return errors.ErrNotFound("未找到该工具")
	}
	if toolset.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除此工具")
	}

	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		if s.s3 != nil {
			for _, r := range s.resourceRepo.FindS3ByToolsetTx(tx, id) {
				if r.Code == "" {
					continue
				}
				if err := s.s3.Delete(context.Background(), r.Code); err != nil {
					slog.Warn("删除 S3 资源失败", "key", r.Code, "error", err)
				}
			}
		}

		if err := s.toolsetRepo.DeleteAllRelated(tx, id); err != nil {
			return err
		}
		if err := s.toolsetRepo.DeleteByID(tx, id); err != nil {
			return err
		}

		adjustMoemoepoint(tx, toolset.UserID, -3,
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("toolset", id),
			moemoepoint.Key("toolset_delete", strconv.Itoa(id)))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除工具失败")
	}
	return nil
}

func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
