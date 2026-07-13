package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Me(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{
		"id":       c.GetUint("user_id"),
		"username": c.GetString("username"),
		"role":     c.GetString("role"),
	})
}
