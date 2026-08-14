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

type agentSettlementResponse struct {
	ID            uint      `json:"id"`
	SettlementNo  string    `json:"settlement_no"`
	AgentID       uint      `json:"agent_id"`
	AgentName     string    `json:"agent_name"`
	Amount        float64   `json:"amount"`
	Method        string    `json:"method"`
	TransactionID string    `json:"transaction_id"`
	PaidAt        time.Time `json:"paid_at"`
	Status        string    `json:"status"`
	Remarks       string    `json:"remarks"`
}

func toAgentSettlementResponse(row models.AgentSettlement) agentSettlementResponse {
	return agentSettlementResponse{ID: row.ID, SettlementNo: row.SettlementNo, AgentID: row.AgentID, AgentName: row.Agent.Name, Amount: row.Amount, Method: row.Method, TransactionID: row.TransactionID, PaidAt: row.PaidAt, Status: row.Status, Remarks: row.Remarks}
}

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
	response := make([]agentSettlementResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, toAgentSettlementResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"settlements": response, "earned": balance.Earned, "paid": balance.Paid, "payable": balance.Payable})
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
	loaded, _, err := services.ListAgentSettlements(row.AgentID)
	if err == nil {
		for _, item := range loaded {
			if item.ID == row.ID {
				row = item
				break
			}
		}
	}
	c.JSON(http.StatusCreated, toAgentSettlementResponse(row))
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
