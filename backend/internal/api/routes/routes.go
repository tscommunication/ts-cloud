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

	// Authentication
	router.POST("/api/v1/auth/login", handlers.Login)

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())

	// =====================================================
	// Authenticated User
	// =====================================================

	api.GET("/me", handlers.Me)

	// =====================================================
	// User APIs
	// =====================================================

	api.GET("/users",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetUsers,
	)

	api.GET("/users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetUser,
	)

	api.POST("/users",
		middleware.RequireRoles("superadmin"),
		handlers.CreateUser,
	)

	api.PUT("/users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateUser,
	)

	api.DELETE("/users/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteUser,
	)

	// =====================================================
	// Customer APIs
	// =====================================================

	api.GET("/customers",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetCustomers,
	)

	api.POST("/customers",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateCustomer,
	)

	// =====================================================
	// Package APIs
	// =====================================================

	api.GET("/packages",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetPackages,
	)

	api.GET("/packages/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetPackage,
	)

	api.POST("/packages",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreatePackage,
	)

	api.PUT("/packages/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdatePackage,
	)

	api.DELETE("/packages/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeletePackage,
	)

	// =====================================================
	// Subscription APIs
	// =====================================================

	api.GET("/subscriptions",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetSubscriptions,
	)

	api.POST("/subscriptions",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateSubscription,
	)

	api.GET("/subscriptions/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetSubscription,
	)

	api.PUT("/subscriptions/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateSubscription,
	)

	api.DELETE("/subscriptions/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteSubscription,
	)

	// =====================================================
	// Invoice APIs
	// =====================================================

	api.GET("/invoices",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetInvoices,
	)

	api.GET("/invoices/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetInvoice,
	)

	api.POST("/invoices",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateInvoice,
	)

	api.PUT("/invoices/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateInvoice,
	)

	api.DELETE("/invoices/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteInvoice,
	)

	// =====================================================
	// Payment APIs
	// =====================================================

	api.GET("/payments",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetPayments,
	)

	api.GET("/payments/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetPayment,
	)

	api.POST("/payments",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreatePayment,
	)

	api.PUT("/payments/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdatePayment,
	)

	api.DELETE("/payments/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeletePayment,
	)

	// =====================================================
	// FTP Server APIs
	// =====================================================

	api.GET("/ftp-servers",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPServers,
	)

	api.GET("/ftp-servers/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPServerByID,
	)

	api.POST("/ftp-servers",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateFTPServer,
	)

	api.PUT("/ftp-servers/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateFTPServer,
	)

	api.DELETE("/ftp-servers/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteFTPServer,
	)
	// =====================================================
	// FTP User APIs
	// =====================================================

	api.GET("/ftp-users",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPUsers,
	)

	api.GET("/ftp-users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPUserByID,
	)

	api.POST("/ftp-users",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateFTPUser,
	)

	api.PUT("/ftp-users/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateFTPUser,
	)

	api.POST("/ftp-users/:id/suspend",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.SuspendFTPUser,
	)

	api.POST("/ftp-users/:id/enable",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.EnableFTPUser,
	)

	api.GET("/ftp-users/:id/stats",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPUserStats,
	)
	api.GET("/ftp-users/:id/login-history",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPLoginHistory,
	)

	api.DELETE("/ftp-users/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteFTPUser,
	)
}
