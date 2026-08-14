package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetAgentDashboard(c *gin.Context) {
	agentID := c.GetUint("agent_id")
	if agentID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent account is not linked"})
		return
	}
	summary, err := services.GetAgentDashboardSummary(agentID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agent dashboard"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
