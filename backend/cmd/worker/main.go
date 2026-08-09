// Command worker runs the background jobs.
//
// The same jobs also run inside cmd/api when RUN_JOBS_IN_API is set, which is
// the default in development so `docker compose up` needs no extra service. In
// production the API scales horizontally, so the worker runs separately; the
// jobs take Postgres advisory locks either way, making a double run harmless.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mokchan/webnovel-backend/internal/config"
	"github.com/mokchan/webnovel-backend/internal/db"
	"github.com/mokchan/webnovel-backend/internal/jobs"
	libraryrepo "github.com/mokchan/webnovel-backend/internal/repository/library"
	notificationrepo "github.com/mokchan/webnovel-backend/internal/repository/notification"
	walletrepo "github.com/mokchan/webnovel-backend/internal/repository/wallet"
	notificationsvc "github.com/mokchan/webnovel-backend/internal/service/notification"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

func main() {
	once := flag.Bool("once", false, "run every job a single time and exit")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gormDB, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db open", "err", err)
		os.Exit(1)
	}
	defer db.Close(gormDB)

	wallet := walletsvc.New(walletrepo.New(gormDB), cfg.BonusTTL, cfg.PlatformFeePercent, time.Now)
	notifier := notificationsvc.New(notificationrepo.New(gormDB), libraryrepo.New(gormDB), time.Now)
	scheduler := jobs.BuildScheduler(gormDB, wallet, notifier, time.Now, slog.Default())

	if *once {
		slog.Info("running every job once")
		scheduler.RunNow(ctx)
		return
	}

	slog.Info("worker started")
	scheduler.Run(ctx)
	slog.Info("worker stopped")
}
