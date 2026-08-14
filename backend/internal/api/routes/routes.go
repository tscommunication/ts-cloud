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
	api.POST("/me/password", handlers.ChangeMyPassword)

	// Organization and distribution hierarchy
	api.GET("/pops", middleware.RequireRoles("superadmin", "admin"), handlers.GetPOPs)
	api.GET("/pops/:id", middleware.RequireRoles("superadmin", "admin"), handlers.GetPOP)
	api.POST("/pops", middleware.RequireRoles("superadmin"), handlers.CreatePOP)
	api.PUT("/pops/:id", middleware.RequireRoles("superadmin"), handlers.UpdatePOP)
	api.PATCH("/pops/:id/status", middleware.RequireRoles("superadmin"), handlers.UpdatePOPStatus)
	api.GET("/agents", middleware.RequireRoles("superadmin", "admin"), handlers.GetAgents)
	api.GET("/agents/:id", middleware.RequireRoles("superadmin", "admin"), handlers.GetAgent)
	api.POST("/agents", middleware.RequireRoles("superadmin", "admin"), handlers.CreateAgent)
	api.PUT("/agents/:id", middleware.RequireRoles("superadmin", "admin"), handlers.UpdateAgent)
	api.PATCH("/agents/:id/status", middleware.RequireRoles("superadmin"), handlers.UpdateAgentStatus)
	api.GET("/agent-collections", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetAgentCollections)
	api.GET("/agent-dashboard", middleware.RequireRoles("agent"), handlers.GetAgentDashboard)
	api.GET("/agent-settlements", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetAgentSettlements)
	api.GET("/network/routers", middleware.RequireRoles("superadmin", "admin"), handlers.GetNetworkRouters)
	api.POST("/network/routers", middleware.RequireRoles("superadmin"), handlers.CreateNetworkRouter)
	api.PUT("/network/routers/:id", middleware.RequireRoles("superadmin"), handlers.UpdateNetworkRouter)
	api.POST("/network/routers/:id/test-connection", middleware.RequireRoles("superadmin"), handlers.TestNetworkRouterConnection)
	api.POST("/agent-settlements", middleware.RequireRoles("superadmin"), handlers.CreateAgentSettlement)
	api.POST("/agent-settlements/:id/void", middleware.RequireRoles("superadmin"), handlers.VoidAgentSettlement)

	// =====================================================
	// User APIs
	// =====================================================

	api.GET("/users",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetUsers,
	)

	api.GET("/users/:id",
		handlers.GetUser,
	)

	api.POST("/users",
		middleware.RequireRoles("superadmin"),
		handlers.CreateUser,
	)

	api.PUT("/users/:id",
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
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomers,
	)

	api.GET("/customers/:id",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerByID,
	)

	api.GET("/customers/:id/summary",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerSummary,
	)

	api.GET("/customers/:id/ledger",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerLedger,
	)

	api.POST("/customers",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateCustomer,
	)

	api.PUT("/customers/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomer,
	)

	api.PATCH("/customers/:id/status",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomerStatus,
	)

	api.POST("/customers/:id/archive",
		middleware.RequireRoles("superadmin"),
		handlers.ArchiveCustomer,
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

	api.POST("/subscriptions/:id/suspend",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.SuspendSubscription,
	)

	api.POST("/subscriptions/:id/activate",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.ActivateSubscription,
	)

	api.POST("/subscriptions/:id/renew",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.RenewSubscription,
	)

	api.POST("/subscriptions/:id/disconnect",
		middleware.RequireRoles("superadmin"),
		handlers.DisconnectSubscription,
	)

	// =====================================================
	// Invoice APIs
	// =====================================================

	api.GET("/invoices",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetInvoices,
	)

	api.GET("/invoices/:id",
		middleware.RequireRoles("superadmin", "admin", "agent"),
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

	api.POST("/invoices/:id/cancel",
		middleware.RequireRoles("superadmin"),
		handlers.CancelInvoice,
	)

	api.GET("/billing/summary",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetBillingSummary,
	)

	api.GET("/billing/runs",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetBillingRuns,
	)

	api.POST("/billing/run",
		middleware.RequireRoles("superadmin"),
		handlers.RunBilling,
	)

	// =====================================================
	// Payment APIs
	// =====================================================

	api.GET("/payments",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetPayments,
	)

	api.GET("/payments/:id",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetPayment,
	)

	api.POST("/payments",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.CreatePayment,
	)

	api.PUT("/payments/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdatePayment,
	)

	api.POST("/payments/:id/void",
		middleware.RequireRoles("superadmin"),
		handlers.VoidPayment,
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

	api.GET("/ftp-dashboard",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetFTPDashboard,
	)

	api.DELETE("/ftp-users/:id",
		middleware.RequireRoles("superadmin"),
		handlers.DeleteFTPUser,
	)
}
