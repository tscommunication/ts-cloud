package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetNetworkDeviceSLA(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	rows, err := services.ListNetworkDeviceSLA(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load SLA report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "devices": rows})
}
