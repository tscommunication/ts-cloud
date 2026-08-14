package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetAgentSettlements(c *gin.Context) {
	var agentID uint
	if value := c.Query("agent_id"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil || parsed == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
			return
		}
		agentID = uint(parsed)
	}
	rows, balance, err := services.ListAgentSettlements(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load settlements"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settlements": rows, "earned": balance.Earned, "paid": balance.Paid, "payable": balance.Payable})
}

func CreateAgentSettlement(c *gin.Context) {
	var req dto.CreateAgentSettlementRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settlement data"})
		return
	}
	paidAt := time.Now()
	if req.PaidAt != "" {
		parsed, err := time.Parse("2006-01-02", req.PaidAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Paid date must be YYYY-MM-DD"})
			return
		}
		paidAt = parsed
	}
	row := models.AgentSettlement{AgentID: req.AgentID, Amount: req.Amount, Method: req.Method, TransactionID: req.TransactionID, PaidAt: paidAt, Remarks: req.Remarks}
	if err := services.CreateAgentSettlement(&row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

func VoidAgentSettlement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settlement ID"})
		return
	}
	if err := services.VoidAgentSettlement(uint(id)); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Settlement voided successfully"})
}
