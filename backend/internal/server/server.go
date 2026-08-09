// Package server is the composition root: it builds every repository, service
// and handler and mounts the route tree.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/mokchan/webnovel-backend/internal/auth"
	"github.com/mokchan/webnovel-backend/internal/config"
	"github.com/mokchan/webnovel-backend/internal/crypto/argon2id"
	"github.com/mokchan/webnovel-backend/internal/jobs"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	"github.com/mokchan/webnovel-backend/internal/ratelimit"

	cataloghandler "github.com/mokchan/webnovel-backend/internal/handler/catalog"
	identityhandler "github.com/mokchan/webnovel-backend/internal/handler/identity"
	libraryhandler "github.com/mokchan/webnovel-backend/internal/handler/library"
	notificationhandler "github.com/mokchan/webnovel-backend/internal/handler/notification"
	readinghandler "github.com/mokchan/webnovel-backend/internal/handler/reading"
	socialhandler "github.com/mokchan/webnovel-backend/internal/handler/social"
	wallethandler "github.com/mokchan/webnovel-backend/internal/handler/wallet"
	writerhandler "github.com/mokchan/webnovel-backend/internal/handler/writer"

	catalogrepo "github.com/mokchan/webnovel-backend/internal/repository/catalog"
	identityrepo "github.com/mokchan/webnovel-backend/internal/repository/identity"
	libraryrepo "github.com/mokchan/webnovel-backend/internal/repository/library"
	notificationrepo "github.com/mokchan/webnovel-backend/internal/repository/notification"
	readingrepo "github.com/mokchan/webnovel-backend/internal/repository/reading"
	socialrepo "github.com/mokchan/webnovel-backend/internal/repository/social"
	walletrepo "github.com/mokchan/webnovel-backend/internal/repository/wallet"
	writerrepo "github.com/mokchan/webnovel-backend/internal/repository/writer"

	catalogsvc "github.com/mokchan/webnovel-backend/internal/service/catalog"
	identitysvc "github.com/mokchan/webnovel-backend/internal/service/identity"
	librarysvc "github.com/mokchan/webnovel-backend/internal/service/library"
	notificationsvc "github.com/mokchan/webnovel-backend/internal/service/notification"
	readingsvc "github.com/mokchan/webnovel-backend/internal/service/reading"
	socialsvc "github.com/mokchan/webnovel-backend/internal/service/social"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
	writersvc "github.com/mokchan/webnovel-backend/internal/service/writer"
	"github.com/mokchan/webnovel-backend/internal/storage"
)

// Deps are the collaborators the composition root builds once and shares.
type Deps struct {
	Config *config.Config
	DB     *gorm.DB
	Issuer *auth.Issuer
	Hasher *argon2id.Hasher
	Now    func() time.Time
}

// New builds the HTTP engine.
func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	engine, err := Build(cfg, db)
	if err != nil {
		// Only a misconfigured JWT secret can fail here, and the process cannot
		// serve authenticated traffic without one.
		panic(err)
	}
	return engine
}

// Build assembles the engine, returning configuration errors instead of
// panicking so tests can assert on them.
func Build(cfg *config.Config, db *gorm.DB) (*gin.Engine, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	issuer, err := auth.NewIssuer(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	deps := Deps{
		Config: cfg,
		DB:     db,
		Issuer: issuer,
		Hasher: argon2id.New(argon2id.Params{
			Memory:      cfg.Argon2Memory,
			Time:        cfg.Argon2Time,
			Parallelism: cfg.Argon2Parallelism,
		}),
		Now: time.Now,
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	// Gin's ClientIP trusts X-Forwarded-For by default. Setting the allowlist
	// explicitly (empty = trust nothing) is what keeps the per-IP rate limit
	// from being defeated by a rotating header.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
		AllowCredentials: true,
		MaxAge:           5 * time.Minute,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Limiters are per-engine, never package globals: handler integration tests
	// each build their own engine and must not share throttle state.
	var authLimiter ratelimit.Limiter
	var bodyCounter *ratelimit.DistinctCounter
	if cfg.RateLimitEnabled {
		authLimiter = ratelimit.NewTokenBucket(cfg.AuthRatePerMin, time.Minute, deps.Now)
		bodyCounter = ratelimit.NewDistinctCounter(cfg.BodyFetchPerMin, time.Minute, deps.Now)
	}

	clock := middleware.Clock(deps.Now)
	optionalAuth := middleware.OptionalAuth(issuer, clock)
	requireAuth := middleware.RequireAuth(issuer, clock)
	bodyLimit := middleware.LimitDistinctChapterBodies(bodyCounter, "id")
	authThrottle := middleware.RateLimitByIP(authLimiter)

	v1 := r.Group("/api/v1")

	// --- identity ----------------------------------------------------------
	identityRepo := identityrepo.New(db)
	sessionStore := identityrepo.NewSessionStore(db)
	identityService := identitysvc.New(
		identityRepo, sessionStore, deps.Hasher, issuer, cfg.RefreshTokenTTL, deps.Now,
	)
	identityhandler.New(identityService).Register(v1, requireAuth, authThrottle)

	// --- wallet ------------------------------------------------------------
	// Built before reading and catalog: both consume its entitlement lookups
	// through narrow ports, so neither imports the wallet package.
	walletRepo := walletrepo.New(db)
	walletService := walletsvc.New(walletRepo, cfg.BonusTTL, cfg.PlatformFeePercent, deps.Now)
	wallethandler.New(walletService, cfg.PaymentsMockEnabled).Register(v1, requireAuth)

	// --- catalog repository -------------------------------------------------
	// Built here rather than with the rest of catalog because library consumes
	// it through the narrow library.SeriesBooks port: ติดตามทั้งชุด has to
	// resolve a series to its books, and the library context must not import
	// catalog to do it.
	catalogRepo := catalogrepo.New(db)

	// --- library -----------------------------------------------------------
	libraryRepo := libraryrepo.New(db)
	libraryService := librarysvc.New(libraryRepo, catalogRepo, deps.Now)
	libraryhandler.New(libraryService).Register(v1, requireAuth)

	// --- notifications -----------------------------------------------------
	// Built before writer and social: both notify through a one-method port, so
	// neither depends on the notification package and they stay independent of
	// each other.
	notificationRepo := notificationrepo.New(db)
	notificationService := notificationsvc.New(notificationRepo, libraryRepo, deps.Now)
	notificationhandler.New(notificationService).Register(v1, requireAuth)

	// --- reading -----------------------------------------------------------
	readingRepo := readingrepo.New(db)
	// The wallet repository satisfies both narrow ports, so reading still never
	// imports the wallet package.
	readingService := readingsvc.New(readingRepo, walletRepo, walletRepo, deps.Now)
	readinghandler.New(readingService).Register(v1, optionalAuth, requireAuth, bodyLimit)

	// --- writer ------------------------------------------------------------
	writerRepo := writerrepo.New(db)
	writerService := writersvc.New(
		writerRepo,
		storage.NewLocalFS(cfg.UploadDir, cfg.PublicUploadBase),
		notificationService,
		deps.Now,
	)
	writerhandler.New(writerService).Register(v1, requireAuth)

	// Uploaded covers are served straight from the upload directory. An empty
	// or root base would register a catch-all `/*filepath` that conflicts with
	// every API route, so serve only when a real prefix is configured.
	if base := strings.TrimRight(cfg.PublicUploadBase, "/"); base != "" && cfg.UploadDir != "" {
		r.Static(base, cfg.UploadDir)
	}

	// --- social ------------------------------------------------------------
	socialRepo := socialrepo.New(db)
	socialService := socialsvc.New(socialRepo, notificationService, deps.Now)
	socialhandler.New(socialService).Register(v1, optionalAuth, requireAuth)

	// --- catalog -----------------------------------------------------------
	// catalogRepo is built above, where library needs it.
	catalogService := catalogsvc.New(catalogRepo, walletRepo)
	cataloghandler.New(catalogService).Register(v1, optionalAuth)

	slog.Debug("routes registered", "count", len(r.Routes()))
	return r, nil
}

// StartJobs runs the background scheduler in-process when RUN_JOBS_IN_API is
// set, so `docker compose up` needs no separate worker in development. The jobs
// take advisory locks, so running here and in cmd/worker at once is harmless.
func StartJobs(ctx context.Context, cfg *config.Config, db *gorm.DB) {
	if !cfg.RunJobsInAPI {
		return
	}
	wallet := walletsvc.New(walletrepo.New(db), cfg.BonusTTL, cfg.PlatformFeePercent, time.Now)
	notifier := notificationsvc.New(notificationrepo.New(db), libraryrepo.New(db), time.Now)
	scheduler := jobs.BuildScheduler(db, wallet, notifier, time.Now, slog.Default())

	go func() {
		slog.Info("background jobs started in-process")
		scheduler.Run(ctx)
	}()
}
