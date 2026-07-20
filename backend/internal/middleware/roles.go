package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRoles(roles ...string) gin.HandlerFunc {

	return func(c *gin.Context) {

		role := c.GetString("role")

		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Permission denied",
		})
	}
}
