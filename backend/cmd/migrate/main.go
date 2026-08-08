package main

import (
	"database/sql"
	"flag"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/mokchan/webnovel-backend/internal/config"
	"github.com/mokchan/webnovel-backend/migrations"
)

func main() {
	cmd := flag.String("cmd", "up", "goose command: up, down, status, reset, version")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("db open", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("dialect", "err", err)
		os.Exit(1)
	}
	if err := goose.Run(*cmd, sqlDB, "."); err != nil {
		slog.Error("goose", "err", err)
		os.Exit(1)
	}
}
