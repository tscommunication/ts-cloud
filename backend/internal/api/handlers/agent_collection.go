package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func GetAgentCollections(c *gin.Context) {
	var agentID uint
	if value := c.Query("agent_id"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil || parsed == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
			return
		}
		agentID = uint(parsed)
	}
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status != "" && status != "ACTIVE" && status != "VOID" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be ACTIVE or VOID"})
		return
	}
	rows, summary, err := repositories.ListAgentCollections(agentID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agent collections"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"collections": rows, "count": summary.Count, "total_amount": summary.TotalAmount, "total_commission": summary.TotalCommission})
}
