package service

import (
	"time"

	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"

	"gorm.io/gorm"
)

type ArticleService struct {
	articleRepo  *repository.ArticleRepository
	categoryRepo *repository.CategoryRepository
	cdnBase      string
}

func NewArticleService(
	articleRepo *repository.ArticleRepository,
	categoryRepo *repository.CategoryRepository,
	cdnBase string,
) *ArticleService {
	return &ArticleService{articleRepo: articleRepo, categoryRepo: categoryRepo, cdnBase: cdnBase}
}

func (s *ArticleService) resolveBannerURL(hash, legacy string) string {
	if hash != "" {
		return imageclient.MainURL(s.cdnBase, hash, "webp")
	}
	return legacy
}

type ArticleListResult struct {
	Items []dto.ArticleSummary
	Total int64
}

var orderByColumn = map[string]string{
	"publishedTime": "published_time",
	"created":       "created",
	"view":          "view",
	"updated":       "updated",
	"order":         "sort_order",
}

func (s *ArticleService) GetList(req *dto.GetArticlesRequest) *ArticleListResult {
	if req.OrderBy == "" {
		req.OrderBy = "order"
	}
	col, ok := orderByColumn[req.OrderBy]
	if !ok {
		col = "sort_order"
	}
	if req.SortOrder == "" {
		if col == "sort_order" {
			req.SortOrder = "asc"
		} else {
			req.SortOrder = "desc"
		}
	}
	req.OrderBy = col

	items, total := s.articleRepo.FindPaginated(req)

	categoryByID := s.loadCategoriesFor(items)
	summaries := make([]dto.ArticleSummary, 0, len(items))
	for _, a := range items {
		sm := toArticleSummary(a, categoryByID[a.CategoryID])
		sm.BannerURL = s.resolveBannerURL(a.BannerImageHash, a.Banner)
		summaries = append(summaries, sm)
	}
	return &ArticleListResult{Items: summaries, Total: total}
}

func (s *ArticleService) Reorder(ids []int) *errors.AppError {
	if err := s.articleRepo.Reorder(ids); err != nil {
		return errors.ErrInternal("调整文档顺序失败")
	}
	return nil
}

func (s *ArticleService) SetPin(id int, isPin bool) *errors.AppError {
	if err := s.articleRepo.SetPin(id, isPin); err != nil {
		return errors.ErrInternal("更新置顶状态失败")
	}
	return nil
}

func (s *ArticleService) loadCategoriesFor(articles []model.DocArticle) map[int]dto.ArticleCategoryBrief {
	out := map[int]dto.ArticleCategoryBrief{}
	if len(articles) == 0 {
		return out
	}
	idSet := map[int]struct{}{}
	ids := make([]int, 0, len(articles))
	for _, a := range articles {
		if _, ok := idSet[a.CategoryID]; ok {
			continue
		}
		idSet[a.CategoryID] = struct{}{}
		ids = append(ids, a.CategoryID)
	}
	var cats []model.DocCategory
	s.categoryRepo.DB().Where("id IN ?", ids).Find(&cats)
	for _, c := range cats {
		out[c.ID] = dto.ArticleCategoryBrief{ID: c.ID, Slug: c.Slug, Title: c.Title}
	}
	return out
}

func toArticleSummary(a model.DocArticle, cat dto.ArticleCategoryBrief) dto.ArticleSummary {
	if cat.ID == 0 {
		cat = dto.ArticleCategoryBrief{ID: a.CategoryID}
	}
	return dto.ArticleSummary{
		ID:              a.ID,
		Title:           a.Title,
		Slug:            a.Slug,
		Path:            a.Path,
		Description:     a.Description,
		Banner:          a.Banner,
		BannerImageHash: a.BannerImageHash,
		Status:          a.Status,
		IsPin:           a.IsPin,
		View:            a.View,
		SortOrder:       a.SortOrder,
		PublishedTime:   a.PublishedTime,
		EditedTime:      a.EditedTime,
		CategoryID:      a.CategoryID,
		AuthorID:        a.AuthorID,
		Category:        cat,
		Created:         a.CreatedAt,
		Updated:         a.UpdatedAt,
	}
}

func (s *ArticleService) GetBySlug(slug string) (*dto.ArticleDetailResponse, *errors.AppError) {
	article, err := s.articleRepo.FindBySlug(slug)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该文章")
	}

	go s.articleRepo.IncrementView(article.ID)

	tagIDs, err := s.articleRepo.FindTagIDsByArticleID(article.ID)
	if err != nil {
		return nil, errors.ErrInternal("获取文章标签失败")
	}

	html, toc := markdown.RenderWithTOC(article.ContentMarkdown)

	var cat dto.ArticleCategoryBrief
	var category model.DocCategory
	if err := s.categoryRepo.DB().
		Where("id = ?", article.CategoryID).First(&category).Error; err == nil {
		cat = dto.ArticleCategoryBrief{ID: category.ID, Slug: category.Slug, Title: category.Title}
	} else {
		cat = dto.ArticleCategoryBrief{ID: article.CategoryID}
	}

	return &dto.ArticleDetailResponse{
		ID:              article.ID,
		Title:           article.Title,
		Slug:            article.Slug,
		Path:            article.Path,
		Description:     article.Description,
		Banner:          article.Banner,
		BannerImageHash: article.BannerImageHash,
		BannerURL:       s.resolveBannerURL(article.BannerImageHash, article.Banner),
		Status:          article.Status,
		IsPin:           article.IsPin,
		View:            article.View,
		PublishedTime:   article.PublishedTime,
		EditedTime:      article.EditedTime,
		ContentMarkdown: article.ContentMarkdown,
		ContentHTML:     html,
		Toc:             toc,
		CategoryID:      article.CategoryID,
		AuthorID:        article.AuthorID,
		Category:        cat,
		TagIDs:          tagIDs,
		Created:         article.CreatedAt,
		Updated:         article.UpdatedAt,
	}, nil
}

func (s *ArticleService) Create(userID int, req *dto.CreateArticleRequest) (*model.DocArticle, *errors.AppError) {
	req.ContentMarkdown = markdown.NormalizeStoredContent(req.ContentMarkdown)
	now := time.Now()
	article := model.DocArticle{
		Title:           req.Title,
		Slug:            req.Slug,
		Path:            "/doc/" + req.Slug,
		Description:     req.Description,
		Banner:          req.Banner,
		BannerImageHash: req.BannerImageHash,
		Status:          req.Status,
		IsPin:           req.IsPin,
		ContentMarkdown: req.ContentMarkdown,
		CategoryID:      req.CategoryID,
		AuthorID:        userID,
		PublishedTime:   now,
	}

	txErr := s.articleRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articleRepo.Create(tx, &article); err != nil {
			return err
		}
		return s.articleRepo.InsertTagRelations(tx, article.ID, req.TagIDs)
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建文章失败")
	}

	return &article, nil
}

func (s *ArticleService) Update(req *dto.UpdateArticleRequest) *errors.AppError {
	req.ContentMarkdown = markdown.NormalizeStoredContent(req.ContentMarkdown)
	now := time.Now()
	updates := map[string]any{
		"title":             req.Title,
		"slug":              req.Slug,
		"path":              "/doc/" + req.Slug,
		"description":       req.Description,
		"banner":            req.Banner,
		"banner_image_hash": req.BannerImageHash,
		"status":            req.Status,
		"is_pin":            req.IsPin,
		"content_markdown":  req.ContentMarkdown,
		"category_id":       req.CategoryID,
		"edited_time":       &now,
	}

	txErr := s.articleRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articleRepo.UpdateFields(tx, req.ArticleID, updates); err != nil {
			return err
		}
		return s.articleRepo.ReplaceTagRelations(tx, req.ArticleID, req.TagIDs)
	})
	if txErr != nil {
		return errors.ErrInternal("更新文章失败")
	}

	return nil
}

func (s *ArticleService) Delete(articleID int) *errors.AppError {
	if err := s.articleRepo.DeleteTagRelationsByArticleID(articleID); err != nil {
		return errors.ErrInternal("删除文章失败")
	}
	if err := s.articleRepo.DeleteByID(articleID); err != nil {
		return errors.ErrInternal("删除文章失败")
	}
	return nil
}
