package handler

import (
	"kun-galgame-api/internal/apprelease/dto"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type ReleaseHandler struct {
	version dto.AppVersionResponse
}

func NewReleaseHandler(cfg config.AppReleaseConfig) *ReleaseHandler {
	return &ReleaseHandler{version: dto.AppVersionResponse{
		MinVersion:    cfg.MinVersion,
		LatestVersion: cfg.LatestVersion,
		Notes:         cfg.Notes,
		Downloads: dto.AppDownloads{
			Android: cfg.Downloads.Android,
			IOS:     cfg.Downloads.IOS,
			Windows: cfg.Downloads.Windows,
			Linux:   cfg.Downloads.Linux,
		},
	}}
}

func (h *ReleaseHandler) GetVersion(c fiber.Ctx) error {
	return response.OK(c, h.version)
}
