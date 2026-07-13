package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/handlers"
	"github.com/tscommunication/ts-cloud/internal/config"
)

func Register(router *gin.Engine, cfg *config.Config) {

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"app":     cfg.AppName,
			"status":  "running",
			"version": "0.1.0",
			"env":     cfg.AppEnv,
		})
	})

	router.GET("/health", handlers.Health)

	router.POST("/api/v1/auth/login", handlers.Login)
}
