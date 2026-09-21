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
	onlyEmpty := flag.Bool("only-empty", true, "Skip rows that already have platforms or runtimes")
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
		Platform  string
		Note      string
		Size      string
		Platforms []byte `gorm:"column:platforms"`
		Runtimes  []byte `gorm:"column:runtimes"`
	}

	processed, updated, skipped := 0, 0, 0
	lastID := 0
	for {
		q := db.Table("galgame_resource").
			Select("id, platform, note, size, platforms, runtimes").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(*batchSize)
		var rows []row
		if err := q.Scan(&rows).Error; err != nil {
			slog.Error("拉取失败", "error", err)
			os.Exit(1)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			processed++
			lastID = r.ID
			if *onlyEmpty {
				hasP := len(r.Platforms) > 0 && string(r.Platforms) != "[]"
				hasR := len(r.Runtimes) > 0 && string(r.Runtimes) != "[]"
				if hasP || hasR {
					skipped++
					continue
				}
			}
			g := resourcevocab.GuessFromText(r.Note, r.Size, r.Platform)
			if len(g.Platforms) == 0 && len(g.Runtimes) == 0 {
				skipped++
				continue
			}
			compat := resourcevocab.CompatPlatform(g.Platforms, g.Runtimes)
			if *dryRun {
				slog.Info("将更新",
					"id", r.ID,
					"legacy", r.Platform,
					"platforms", []string(g.Platforms),
					"runtimes", []string(g.Runtimes),
					"compat", compat,
				)
			} else {
				fields := map[string]any{
					"platforms": g.Platforms,
					"runtimes":  g.Runtimes,
				}
				if compat != "" {
					fields["platform"] = compat
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
