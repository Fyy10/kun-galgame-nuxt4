package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/internal/trust/gate"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type ResourceService struct {
	resourceRepo *repository.ResourceRepository
	toolsetRepo  *repository.ToolsetRepository
	s3           *storage.S3Client
	art          *artifactclient.Client
	userClient   *userclient.Client
	check        *gate.CheckService
	scan         *gate.ScanService
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	toolsetRepo *repository.ToolsetRepository,
	s3 *storage.S3Client,
	art *artifactclient.Client,
	userClient *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *ResourceService {
	return &ResourceService{
		resourceRepo: resourceRepo,
		toolsetRepo:  toolsetRepo,
		s3:           s3,
		art:          art,
		userClient:   userClient,
		check:        check,
		scan:         scan,
	}
}

func (s *ResourceService) GetResourceDetail(
	ctx context.Context,
	req *dto.ResourceDetailRequest,
) (*dto.ResourceDetailResponse, *errors.AppError) {
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	uc, _, _ := s.userClient.User(ctx, resource.UserID)
	if !userclient.IsRenderable(uc) {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	go s.resourceRepo.IncrementDownload(resource.ID)

	if resource.Type == "s3" && resource.ArtifactUUID != "" {
		if dl, derr := s.art.Download(ctx, resource.ArtifactUUID); derr == nil {
			resource.Content = dl.Url
		} else {
			slog.Warn("解析 artifact 下载链接失败", "uuid", resource.ArtifactUUID, "error", derr)
		}
	}

	user := userModel.UserBrief{ID: uc.ID, Name: uc.Name, Avatar: uc.Avatar}

	return &dto.ResourceDetailResponse{
		GalgameToolsetResource: *resource,
		User:                   user,
	}, nil
}

func (s *ResourceService) CreateResource(
	ctx context.Context,
	userID, toolsetID int,
	req *dto.CreateResourceRequest,
) (*dto.CreatedResourceResponse, *errors.AppError) {
	req.Content = markdown.NormalizeStoredContent(req.Content)
	req.Note = markdown.NormalizeStoredContent(req.Note)
	if _, err := s.toolsetRepo.FindByID(toolsetID); err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	moderationText := gate.ComposeText(req.Content, req.Note)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return nil, gate.ErrContentBlocked()
	}

	var resource model.GalgameToolsetResource
	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		resource = model.GalgameToolsetResource{
			Content:      req.Content,
			Type:         req.Type,
			ArtifactUUID: req.ArtifactUUID,
			Code:         req.Code,
			Password:     req.Password,
			Size:         req.Size,
			Note:         req.Note,
			ToolsetID:    toolsetID,
			UserID:       userID,
		}
		if err := s.resourceRepo.Create(tx, &resource); err != nil {
			return err
		}

		adjustMoemoepoint(tx, userID, 3,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("toolset", toolsetID),
			moemoepoint.Key("resource_create", strconv.Itoa(resource.ID)))

		if err := s.toolsetRepo.AddContributor(tx, toolsetID, userID); err != nil {
			return err
		}

		return s.toolsetRepo.UpdateResourceTime(tx, toolsetID, time.Now())
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建资源失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindToolsetResource, "subject_id", resource.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindToolsetResource, strconv.Itoa(resource.ID), moderationText, int64(userID))

	return &resource, nil
}

func (s *ResourceService) UpdateResource(
	ctx context.Context,
	userID int, canModerate bool,
	req *dto.UpdateResourceRequest,
) (*model.GalgameToolsetResource, *errors.AppError) {
	req.Content = markdown.NormalizeStoredContent(req.Content)
	req.Note = markdown.NormalizeStoredContent(req.Note)
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	if resource.UserID != userID && !canModerate {
		return nil, errors.ErrForbidden("您没有权限编辑此资源")
	}

	moderationText := gate.ComposeText(req.Content, req.Note)
	authorID := int64(resource.UserID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return nil, gate.ErrContentBlocked()
	}

	now := time.Now()
	updates := map[string]any{
		"password": req.Password,
		"note":     req.Note,
		"edited":   now,
	}

	if resource.Type == "user" {
		updates["content"] = req.Content
		updates["code"] = req.Code
		updates["size"] = req.Size
	}

	if err := s.resourceRepo.UpdateFields(resource, updates); err != nil {
		return nil, errors.ErrInternal("更新资源失败")
	}

	refreshed, refreshErr := s.resourceRepo.FindByID(resource.ID)
	if refreshErr != nil {
		return nil, errors.ErrInternal("读取更新后的资源失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindToolsetResource, "subject_id", req.ResourceID, "author_id", resource.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindToolsetResource, strconv.Itoa(req.ResourceID), moderationText, int64(resource.UserID))
	return refreshed, nil
}

func (s *ResourceService) DeleteResource(
	userID int, canModerate bool,
	req *dto.DeleteResourceRequest,
) *errors.AppError {
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return errors.ErrNotFound("未找到该资源")
	}

	if resource.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除此资源")
	}

	if resource.Type == "s3" {
		if resource.ArtifactUUID != "" {
			if err := s.art.Delete(context.Background(), resource.ArtifactUUID); err != nil {
				slog.Warn("删除 artifact 资源失败", "uuid", resource.ArtifactUUID, "error", err)
			}
		} else if resource.Code != "" && s.s3 != nil {
			if err := s.s3.Delete(context.Background(), resource.Code); err != nil {
				slog.Warn("删除 S3 资源失败", "key", resource.Code, "error", err)
			}
		}
	}

	if err := s.resourceRepo.Delete(resource); err != nil {
		return errors.ErrInternal("删除资源失败")
	}

	adjustMoemoepoint(s.resourceRepo.DB(), resource.UserID, -3,
		moemoepoint.ReasonContentRemoved, moemoepoint.Ref("toolset_resource", resource.ID),
		moemoepoint.Key("resource_delete", strconv.Itoa(resource.ID)))

	return nil
}

func adjustMoemoepoint(_ *gorm.DB, userID, delta int, reason, ref, idempotencyKey string) {
	moemoepoint.Award(userID, delta, reason, ref, idempotencyKey)
}
