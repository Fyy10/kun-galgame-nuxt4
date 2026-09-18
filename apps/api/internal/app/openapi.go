package app

import (
	"kun-galgame-api/pkg/config"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

// V1Spec mounts the real route table with no backing services, so the committed
// spec is the one the server registers rather than a second list that can drift.
func V1Spec() huma.API {
	cfg := &config.Config{}
	cfg.CORS.AllowOrigins = "https://www.kungal.com"
	a := &App{Fiber: fiber.New(), Config: cfg}
	a.setupRoutes()
	return a.APIv1
}
