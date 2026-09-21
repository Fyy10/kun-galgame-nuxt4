package main

import (
	"flag"
	"log/slog"
	"os"

	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Report planned changes but do not write")
	batchSize := flag.Int("batch", 500, "Rows per batch")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	type row struct {
		ID        int
		Type      string
		Platform  string
		Note      string
		Size      string
		Platforms []byte `gorm:"column:platforms"`
		Runtimes  []byte `gorm:"column:runtimes"`
	}

	processed, updated, skipped := 0, 0, 0
	lastID := 0
	for {
		var rows []row
		err := db.Table("galgame_resource").
			Select("id, type, platform, note, size, platforms, runtimes").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(*batchSize).
			Scan(&rows).Error
		if err != nil {
			slog.Error("拉取失败", "error", err)
			os.Exit(1)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			processed++
			lastID = r.ID
			patch, ok := planAxes(axesRow{
				Type: r.Type, Legacy: r.Platform, Note: r.Note, Size: r.Size,
				Platforms: scanKeys(r.Platforms), Runtimes: scanKeys(r.Runtimes),
			})
			if !ok {
				skipped++
				continue
			}
			if *dryRun {
				slog.Info("将更新",
					"id", r.ID,
					"set_platforms", patch.SetP, "platforms", []string(patch.Platforms),
					"set_runtimes", patch.SetR, "runtimes", []string(patch.Runtimes),
				)
			} else {
				fields := map[string]any{}
				if patch.SetP {
					fields["platforms"] = patch.Platforms
				}
				if patch.SetR {
					fields["runtimes"] = patch.Runtimes
				}
				if err := db.Table("galgame_resource").Where("id = ?", r.ID).
					Updates(fields).Error; err != nil {
					slog.Error("更新失败", "id", r.ID, "error", err)
					os.Exit(1)
				}
			}
			updated++
		}
	}
	slog.Info("完成", "processed", processed, "updated", updated, "skipped", skipped, "dry_run", *dryRun)
}

func scanKeys(raw []byte) resourcevocab.Keys {
	var k resourcevocab.Keys
	_ = k.Scan(raw)
	return k
}
