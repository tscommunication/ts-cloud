package main

import (
	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/routes"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/database/seeder"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg); err != nil {
		panic(err)
	}

	seeder.SeedAdmin()

	router := gin.Default()

	routes.Register(router, cfg)

	router.Run(":" + cfg.AppPort)
}
