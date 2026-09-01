package main

// @title TS Cloud ISP Billing API
// @version 1.0
// @description Enterprise ISP Billing & Network Management System
// @contact.name TS Communication
// @contact.email ts.communicationmagura@gmail.com
// @license.name MIT
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

import (
	"log"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/tscommunication/ts-cloud/docs"

	"github.com/tscommunication/ts-cloud/internal/api/routes"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/database/seeder"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func main() {

	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	// Database
	if err := database.Connect(cfg); err != nil {
		panic(err)
	}

	backfillResult, err :=
		services.BackfillLegacyPPPoECredentials(
			cfg.CredentialKey,
		)
	if err != nil {
		panic(err)
	}

	log.Printf(
		"Legacy PPPoE credential backfill complete "+
			"(subscriptions=%d, provision_requests=%d)",
		backfillResult.SubscriptionsUpdated,
		backfillResult.ProvisionRequestsUpdated,
	)

	portalBackfillResult, err := services.BackfillCustomerPortalIdentities(cfg.CredentialKey)
	if err != nil {
		panic(err)
	}
	log.Printf(
		"Customer portal identity backfill complete (created=%d, updated=%d)",
		portalBackfillResult.Created,
		portalBackfillResult.Updated,
	)

	// Seed default admin
	seeder.SeedAdmin()
	if _, _, err := services.SyncApprovedPackageCatalog(); err != nil {
		panic(err)
	}
	if _, err := services.SyncApprovedDistributionCatalog(); err != nil {
		panic(err)
	}
	if _, err := services.SyncApprovedLocationCatalog(); err != nil {
		panic(err)
	}

	// Start FTP Background Monitor
	services.StartFTPMonitor()
	services.StartSubscriptionExpiryWorker(cfg.CredentialKey)
	services.StartCustomerPPPoESyncWorker(cfg.CredentialKey)
	services.StartTemporaryInternetAccessWorker(cfg.CredentialKey)
	services.StartInvoiceOverdueWorker()
	services.StartBillingWorker()
	services.StartNetworkRouterMonitor(cfg.CredentialKey, cfg.RouterMonitorInterval, cfg.RouterCPUAlertPercent, cfg.RouterMemoryAlertPercent)
	services.StartNetworkDeviceMonitor(cfg.CredentialKey)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	if err := router.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		panic(err)
	}

	// Swagger UI
	router.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler),
	)

	// Register API Routes
	routes.Register(router, cfg)

	// Start HTTP Server
	if err := router.Run(":" + cfg.AppPort); err != nil {
		panic(err)
	}
}
