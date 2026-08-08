package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/mokchan/webnovel-backend/internal/config"
	cataloghandler "github.com/mokchan/webnovel-backend/internal/handler/catalog"
	catalogrepo "github.com/mokchan/webnovel-backend/internal/repository/catalog"
	catalogsvc "github.com/mokchan/webnovel-backend/internal/service/catalog"
)

func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
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

	catalogRepo := catalogrepo.New(db)
	catalogService := catalogsvc.New(catalogRepo)

	v1 := r.Group("/api/v1")
	cataloghandler.New(catalogService).Register(v1)

	return r
}
