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
	api.GET("/notifications", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.GetNotifications)
	api.POST("/notifications/read-all", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.MarkAllNotificationsRead)
	api.POST("/notifications/:id/read", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.MarkNotificationRead)

	// =====================================================
	// Customer Portal APIs
	// =====================================================

	customerPortal := api.Group("/customer-portal")
	customerPortal.Use(middleware.RequireRoles("customer"))
	customerPortal.GET("/me", handlers.GetCustomerPortalMe)
	customerPortal.GET("/connection", handlers.GetCustomerPortalConnection(cfg))
	customerPortal.GET("/live-traffic", handlers.GetCustomerPortalLiveTraffic(cfg))
	customerPortal.GET("/subscription", handlers.GetCustomerPortalSubscription(cfg))
	customerPortal.GET("/invoices", handlers.GetCustomerPortalInvoices)
	customerPortal.GET("/payments", handlers.GetCustomerPortalPayments)
	customerPortal.GET("/temporary-access", handlers.GetCustomerPortalTemporaryAccess)
	customerPortal.GET("/ftp-entitlements", handlers.GetCustomerPortalFTPEntitlements)
	customerPortal.GET("/service-entitlements", handlers.GetCustomerPortalServiceEntitlements)

	// Organization and distribution hierarchy
	api.GET("/divisions",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetDivisions,
	)
	api.GET("/divisions/:id/districts",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetDistrictsByDivision,
	)
	api.GET("/districts/:id/upazilas",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetUpazilasByDistrict,
	)
	api.GET("/districts/:id/post-offices",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetPostOfficesByDistrict,
	)
	api.GET("/upazilas/:id/post-offices",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetPostOfficesByUpazila,
	)

	api.GET("/pops", middleware.RequireRoles("superadmin", "admin"), handlers.GetPOPs)
	api.GET("/pops/archived", middleware.RequireRoles("superadmin"), handlers.GetArchivedPOPs)
	api.GET("/pops/:id", middleware.RequireRoles("superadmin", "admin"), handlers.GetPOP)
	api.POST("/pops", middleware.RequireRoles("superadmin"), handlers.CreatePOP)
	api.PUT("/pops/:id", middleware.RequireRoles("superadmin"), handlers.UpdatePOP)
	api.PATCH("/pops/:id/status", middleware.RequireRoles("superadmin"), handlers.UpdatePOPStatus)
	api.POST("/pops/:id/migrate", middleware.RequireRoles("superadmin"), handlers.MigratePOP)
	api.POST("/pops/:id/restore", middleware.RequireRoles("superadmin"), handlers.RestorePOP)
	api.DELETE("/pops/:id", middleware.RequireRoles("superadmin"), handlers.DeletePOP)
	api.GET("/agents", middleware.RequireRoles("superadmin", "admin"), handlers.GetAgents)
	api.GET("/agents/archived", middleware.RequireRoles("superadmin"), handlers.GetArchivedAgents)
	api.GET("/agents/:id", middleware.RequireRoles("superadmin", "admin"), handlers.GetAgent)
	api.POST("/agents", middleware.RequireRoles("superadmin", "admin"), handlers.CreateAgent)
	api.PUT("/agents/:id", middleware.RequireRoles("superadmin", "admin"), handlers.UpdateAgent)
	api.PUT("/agents/:id/packages", middleware.RequireRoles("superadmin", "admin"), handlers.UpdateAgentPackages)
	api.PUT("/agents/:id/permissions", middleware.RequireRoles("superadmin", "admin"), handlers.UpdateAgentPermissions)
	api.PATCH("/agents/:id/status", middleware.RequireRoles("superadmin"), handlers.UpdateAgentStatus)
	api.POST("/agents/:id/migrate", middleware.RequireRoles("superadmin"), handlers.MigrateAgent)
	api.POST("/agents/:id/restore", middleware.RequireRoles("superadmin"), handlers.RestoreAgent)
	api.DELETE("/agents/:id", middleware.RequireRoles("superadmin"), handlers.DeleteAgent)
	api.PUT("/code-management/:entity/:id", middleware.RequireRoles("superadmin"), handlers.UpdateManagedCode)
	api.GET("/agent-collections", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetAgentCollections)
	api.GET("/agent-dashboard", middleware.RequireRoles("agent"), handlers.GetAgentDashboard)
	api.GET("/customer-change-requests", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.ListCustomerChangeRequests)
	api.GET("/customer-change-requests/options", middleware.RequireRoles("agent"), handlers.GetCustomerChangeRequestOptions)
	api.POST("/customer-change-requests", middleware.RequireRoles("agent"), handlers.CreateCustomerChangeRequest)
	api.POST("/customer-change-requests/:id/approve", middleware.RequireRoles("superadmin", "admin"), handlers.ApproveCustomerChangeRequest(cfg))
	api.POST("/customer-change-requests/:id/reject", middleware.RequireRoles("superadmin", "admin"), handlers.RejectCustomerChangeRequest(cfg))
	api.GET("/agent-settlements", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetAgentSettlements)
	api.GET("/network/routers", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetNetworkRouters)
	api.POST("/network/routers", middleware.RequireRoles("superadmin"), handlers.CreateNetworkRouter)
	api.PUT("/network/routers/:id", middleware.RequireRoles("superadmin"), handlers.UpdateNetworkRouter)
	api.POST("/network/routers/:id/test-connection", middleware.RequireRoles("superadmin"), handlers.TestNetworkRouterConnection)
	api.PUT("/network/routers/:id/credentials", middleware.RequireRoles("superadmin"), handlers.SetNetworkRouterCredentials(cfg))
	api.POST("/network/routers/:id/sync-resource", middleware.RequireRoles("superadmin"), handlers.SyncNetworkRouterResource(cfg))
	api.GET("/network/routers/:id/history", middleware.RequireRoles("superadmin", "admin"), handlers.GetNetworkRouterHistory)
	api.GET("/network/routers/:id/pppoe-sessions", middleware.RequireRoles("superadmin", "admin"), handlers.GetNetworkRouterPPPoESessions)
	api.GET("/network/pppoe-summary", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetNetworkPPPoESummary)
	api.GET("/network/pppoe-usage-summary", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetNetworkPPPoEDailyUsageSummary)
	api.GET("/network/pppoe-usage", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.ListNetworkPPPoEUserUsage)
	api.GET("/network/pppoe-sessions", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetNetworkPPPoESessions)
	api.GET("/network/pppoe-sessions/:id/live-traffic", middleware.RequireRoles("superadmin", "admin", "agent"), handlers.GetNetworkPPPoESessionLiveTraffic(cfg))
	api.POST("/network/pppoe-sessions/:id/map", middleware.RequireRoles("superadmin", "admin"), handlers.MapNetworkPPPoESession)
	api.GET("/network/ppp-secrets", middleware.RequireRoles("superadmin", "admin"), handlers.GetNetworkRouterPPPSecrets)
	api.POST("/network/ppp-secrets/:id/map", middleware.RequireRoles("superadmin", "admin"), handlers.MapNetworkRouterPPPSecret)
	api.GET("/service-entitlements", middleware.RequireRoles("superadmin", "admin"), handlers.ListServiceEntitlements)
	api.POST("/service-entitlements", middleware.RequireRoles("superadmin", "admin"), handlers.CreateServiceEntitlement(cfg))
	api.PUT("/service-entitlements/:id", middleware.RequireRoles("superadmin", "admin"), handlers.UpdateServiceEntitlement(cfg))
	api.DELETE("/service-entitlements/:id", middleware.RequireRoles("superadmin"), handlers.DeleteServiceEntitlement)
	api.GET("/network/router-alerts", middleware.RequireRoles("superadmin", "admin"), handlers.GetNetworkRouterAlerts)
	api.GET("/network/olt-dashboard", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.GetOLTDashboard)
	api.GET("/network/devices", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.ListNetworkDevices(cfg))
	api.GET("/network/devices/:id/ports", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.ListNetworkDevicePorts)
	api.GET("/network/devices/:id/onus", middleware.RequireRoles("superadmin", "admin", "noc", "agent"), handlers.ListNetworkDeviceONUs)
	api.POST("/network/devices", middleware.RequireRoles("superadmin"), handlers.SaveNetworkDevice(cfg))
	api.PUT("/network/devices/:id", middleware.RequireRoles("superadmin"), handlers.SaveNetworkDevice(cfg))
	api.POST("/network/devices/:id/test-connection", middleware.RequireRoles("superadmin", "admin"), handlers.TestNetworkDeviceConnection(cfg))
	api.DELETE("/network/devices/:id", middleware.RequireRoles("superadmin"), handlers.DeleteNetworkDevice)
	api.POST("/customer-imports/preview", middleware.RequireRoles("superadmin"), handlers.PreviewCustomerCSV)
	api.POST("/customer-imports", middleware.RequireRoles("superadmin"), handlers.ImportCustomerCSV(cfg))
	api.POST("/agent-user-imports/preview", middleware.RequireRoles("superadmin"), handlers.PreviewAgentUserImport)
	api.POST("/agent-user-imports", middleware.RequireRoles("superadmin"), handlers.ImportAgentUsers)
	api.GET("/data-exports", middleware.RequireRoles("superadmin"), handlers.ExportData)
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
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.CreateCustomer,
	)

	api.PUT("/customers/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomer,
	)

	api.PATCH("/customers/:id/status",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomerStatus(cfg),
	)

	api.POST("/customers/:id/archive",
		middleware.RequireRoles("superadmin"),
		handlers.ArchiveCustomer,
	)

	api.POST("/customers/bulk-extend-expiry",
		middleware.RequireRoles("superadmin"),
		handlers.BulkExtendCustomerInternetExpiry(cfg),
	)

	api.GET("/customers/:id/technical-profile",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerTechnicalProfile,
	)

	api.GET("/customers/:id/network-path",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerNetworkPath(cfg),
	)

	api.GET("/customers/:id/internet-credential",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerInternetCredential(cfg),
	)

	api.PUT("/customers/:id/internet-credential",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.SaveCustomerInternetCredential(cfg),
	)

	api.GET("/customers/:id/temporary-access",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.ListTemporaryInternetAccess,
	)
	api.POST("/customers/:id/temporary-access",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GrantTemporaryInternetAccess(cfg),
	)
	api.POST("/customers/:id/temporary-access/:access_id/cancel",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.CancelTemporaryInternetAccess(cfg),
	)

	api.PUT("/customers/:id/technical-profile",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomerTechnicalProfile(cfg),
	)

	api.GET("/customers/:id/references",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.ListCustomerReferences,
	)

	api.POST("/customers/:id/references",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateCustomerReference,
	)

	api.PUT("/customers/:id/references/:reference_id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateCustomerReference,
	)

	api.DELETE("/customers/:id/references/:reference_id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.DeleteCustomerReference,
	)
	// =====================================================
	// Provision Catalog APIs
	// =====================================================

	api.GET("/provision-catalog/packages",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetProvisionCatalogPackages,
	)

	api.GET("/provision-catalog/routers",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetProvisionCatalogRouters,
	)

	// =====================================================
	// Customer Provision Request APIs
	// =====================================================

	api.POST("/customer-provision-requests",
		middleware.RequireRoles("agent"),
		handlers.CreateCustomerProvisionRequest(cfg),
	)

	api.GET("/customer-provision-requests",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerProvisionRequests,
	)

	api.GET("/customer-provision-requests/:id",
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetCustomerProvisionRequest,
	)

	api.POST("/customer-provision-requests/:id/approve",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.ApproveCustomerProvisionRequest(cfg),
	)

	api.POST("/customer-provision-requests/:id/reject",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.RejectCustomerProvisionRequest,
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
		middleware.RequireRoles("superadmin", "admin", "agent"),
		handlers.GetSubscriptions,
	)

	api.POST("/subscriptions",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.CreateSubscription(cfg),
	)

	api.GET("/subscriptions/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.GetSubscription,
	)

	api.PUT("/subscriptions/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdateSubscription(cfg),
	)

	api.POST("/subscriptions/:id/suspend",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.SuspendSubscription(cfg),
	)

	api.POST("/subscriptions/:id/activate",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.ActivateSubscription(cfg),
	)

	api.POST("/subscriptions/:id/renew",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.RenewSubscription(cfg),
	)

	api.POST("/subscriptions/:id/disconnect",
		middleware.RequireRoles("superadmin"),
		handlers.DisconnectSubscription(cfg),
	)

	api.POST("/subscriptions/:id/reconcile-pppoe",
		middleware.RequireRoles("superadmin"),
		handlers.ReconcileSubscriptionPPPSecret(cfg),
	)

	api.POST("/subscriptions/:id/adjust-date",
		middleware.RequireRoles("superadmin"),
		handlers.AdjustSubscriptionDate(cfg),
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
		handlers.CreatePayment(cfg),
	)

	api.PUT("/payments/:id",
		middleware.RequireRoles("superadmin", "admin"),
		handlers.UpdatePayment,
	)

	api.POST("/payments/:id/void",
		middleware.RequireRoles("superadmin"),
		handlers.VoidPayment(cfg),
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
