package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                 string
	Port                int
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	CORSOrigins         []string
	PaymentsMockEnabled bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := strconv.Atoi(getenv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	cfg := &Config{
		Env:                 getenv("ENV", "development"),
		Port:                port,
		DatabaseURL:         mustEnv("DATABASE_URL"),
		RedisURL:            getenv("REDIS_URL", ""),
		JWTSecret:           mustEnv("JWT_SECRET"),
		CORSOrigins:         splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		PaymentsMockEnabled: getenv("PAYMENTS_MOCK_ENABLED", "true") == "true",
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
