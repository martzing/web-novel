// Package config loads runtime settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every runtime setting the API and worker binaries need.
type Config struct {
	Env         string
	Port        int
	DatabaseURL string
	CORSOrigins []string

	// Auth
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Password hashing (argon2id)
	Argon2Memory      uint32
	Argon2Time        uint32
	Argon2Parallelism uint8

	// Abuse control
	RateLimitEnabled bool
	AuthRatePerMin   int
	BodyFetchPerMin  int
	TrustedProxies   []string

	// Coins
	PaymentsMockEnabled bool
	BonusTTL            time.Duration
	PlatformFeePercent  int

	// Writer uploads
	UploadDir        string
	PublicUploadBase string

	// Background jobs
	RunJobsInAPI bool
}

// Load reads configuration from .env (when present) and the process environment.
func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := envInt("PORT", 8080)
	if err != nil {
		return nil, err
	}
	authRate, err := envInt("AUTH_RATE_PER_MIN", 60)
	if err != nil {
		return nil, err
	}
	bodyFetch, err := envInt("BODY_FETCH_PER_MIN", 20)
	if err != nil {
		return nil, err
	}
	feePercent, err := envInt("PLATFORM_FEE_PERCENT", 30)
	if err != nil {
		return nil, err
	}
	if feePercent < 0 || feePercent > 100 {
		return nil, fmt.Errorf("PLATFORM_FEE_PERCENT must be between 0 and 100, got %d", feePercent)
	}
	argonMemory, err := envInt("ARGON2_MEMORY_KIB", 64*1024)
	if err != nil {
		return nil, err
	}
	argonTime, err := envInt("ARGON2_TIME", 3)
	if err != nil {
		return nil, err
	}
	argonPar, err := envInt("ARGON2_PARALLELISM", 2)
	if err != nil {
		return nil, err
	}
	accessTTL, err := envDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := envDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return nil, err
	}
	bonusTTL, err := envDuration("BONUS_TTL", 30*24*time.Hour)
	if err != nil {
		return nil, err
	}

	env := getenv("ENV", "development")

	cfg := &Config{
		Env:         env,
		Port:        port,
		DatabaseURL: mustEnv("DATABASE_URL"),
		CORSOrigins: splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),

		JWTSecret:       mustEnv("JWT_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,

		Argon2Memory:      uint32(argonMemory),
		Argon2Time:        uint32(argonTime),
		Argon2Parallelism: uint8(argonPar),

		RateLimitEnabled: envBool("RATE_LIMIT_ENABLED", true),
		AuthRatePerMin:   authRate,
		BodyFetchPerMin:  bodyFetch,
		TrustedProxies:   splitCSV(getenv("TRUSTED_PROXIES", "")),

		PaymentsMockEnabled: envBool("PAYMENTS_MOCK_ENABLED", true),
		BonusTTL:            bonusTTL,
		PlatformFeePercent:  feePercent,

		UploadDir:        getenv("UPLOAD_DIR", "./uploads"),
		PublicUploadBase: getenv("PUBLIC_UPLOAD_BASE", "/uploads"),

		RunJobsInAPI: envBool("RUN_JOBS_IN_API", env != "production"),
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing env " + k)
	}
	return v
}

func envInt(k string, def int) (int, error) {
	raw := getenv(k, "")
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", k, err)
	}
	return v, nil
}

func envDuration(k string, def time.Duration) (time.Duration, error) {
	raw := getenv(k, "")
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", k, err)
	}
	return v, nil
}

func envBool(k string, def bool) bool {
	raw := getenv(k, "")
	if raw == "" {
		return def
	}
	return raw == "true" || raw == "1"
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
