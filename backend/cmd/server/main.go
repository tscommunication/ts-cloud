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

	// Database
	if err := database.Connect(cfg); err != nil {
		panic(err)
	}

	// Seed default admin
	seeder.SeedAdmin()

	// Start FTP Background Monitor
	services.StartFTPMonitor()

	router := gin.Default()

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
