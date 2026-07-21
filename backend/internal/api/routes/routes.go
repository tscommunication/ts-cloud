package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/handlers"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/middleware"
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

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())

	// Everyone (authenticated)
	api.GET("/me", handlers.Me)

	// SuperAdmin + Admin
	api.GET(
		"/users",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetUsers,
	)

	api.GET(
		"/users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetUser,
	)

	// SuperAdmin only
	api.POST(
		"/users",
		middleware.RequireRoles("superadmin"),
		handlers.CreateUser,
	)

	// SuperAdmin + Admin
	api.PUT(
		"/users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateUser,
	)

	// SuperAdmin only
	api.DELETE(
		"/users/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteUser,
	)
	// Customer APIs

	api.GET(
        "/customers",
        middleware.RequireRoles("superadmin", "admin"),
        handlers.GetCustomers,
	)

	api.POST(
        "/customers",
        middleware.RequireRoles("superadmin", "admin"),
        handlers.CreateCustomer,
	)
}
