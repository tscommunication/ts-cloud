package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"app":     "T.S Cloud",
			"version": "0.1.0",
			"status":  "running",
		})
	})

	r.Run(":8080")
}
