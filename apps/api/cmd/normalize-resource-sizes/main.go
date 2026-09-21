package main

import (
	"flag"
	"log/slog"
	"os"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Report planned changes but do not write")
	batchSize := flag.Int("batch", 500, "Number of resources to process per batch")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	total := int64(0)
	if err := db.Table("galgame_resource").Count(&total).Error; err != nil {
		slog.Error("统计资源总数失败", "error", err)
		os.Exit(1)
	}
	slog.Info("开始规范化资源体积", "total", total, "batch", *batchSize, "dry_run", *dryRun)

	type resRow struct {
		ID   int
		Size string
	}

	processed := 0
	updated := 0
	unchanged := 0
	skipped := 0
	lastID := 0
	for {
		var rows []resRow
		err := db.Table("galgame_resource").
			Select("id, size").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(*batchSize).
			Scan(&rows).Error
		if err != nil {
			slog.Error("拉取资源批次失败", "error", err, "lastID", lastID)
			os.Exit(1)
		}
		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			processed++
			lastID = r.ID
			next, ok := extract(r.Size)
			if !ok {
				skipped++
				slog.Warn("无法解析体积", "id", r.ID, "size", r.Size)
				continue
			}
			if next == r.Size {
				unchanged++
				continue
			}
			if *dryRun {
				slog.Info("将更新", "id", r.ID, "from", r.Size, "to", next)
			} else if err := db.Table("galgame_resource").Where("id = ?", r.ID).
				Update("size", next).Error; err != nil {
				slog.Error("更新失败", "id", r.ID, "error", err)
				os.Exit(1)
			}
			updated++
		}
	}

	slog.Info("完成",
		"processed", processed,
		"updated", updated,
		"unchanged", unchanged,
		"skipped", skipped,
		"dry_run", *dryRun,
	)
}
