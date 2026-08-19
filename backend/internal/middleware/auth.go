package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/auth"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func AuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Bearer authorization required",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil || !user.Active {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Account is unavailable or disabled",
			})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		if user.AgentID != nil {
			c.Set("agent_id", *user.AgentID)
		}
		if user.CustomerID != nil {
			c.Set("customer_id", *user.CustomerID)
		}

		c.Next()
	}
}
