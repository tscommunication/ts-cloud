package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetOLTDashboard(c *gin.Context) {
	var agentID uint

	if c.GetString("role") == "agent" {
		agentID = c.GetUint("agent_id")
		if agentID == 0 {
			c.JSON(
				http.StatusForbidden,
				gin.H{"error": "Agent account is not linked"},
			)
			return
		}
	}

	dashboard, err := services.GetOLTDashboard(agentID)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Failed to load OLT dashboard"},
		)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}
