// Command migrate chạy database migration: `migrate up` | `migrate down`.
package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"kg-cdl/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// Đường dẫn tới thư mục migration (tính từ thư mục gốc dự án).
	sourceURL := "file://../db/migrations"
	if v := os.Getenv("MIGRATIONS_PATH"); v != "" {
		sourceURL = "file://" + v
	}

	m, err := migrate.New(sourceURL, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		log.Fatalf("unknown command %q (use 'up' or 'down')", cmd)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate %s: %v", cmd, err)
	}
	log.Printf("migrate %s: done", cmd)
}
