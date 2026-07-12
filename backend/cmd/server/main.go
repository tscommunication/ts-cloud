package main

import (
	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/routes"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg); err != nil {
		panic(err)
	}

	router := gin.Default()

	routes.Register(router, cfg)

	router.Run(":" + cfg.AppPort)
}
