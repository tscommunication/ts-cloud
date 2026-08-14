package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type agentCollectionResponse struct {
	ID               uint      `json:"id"`
	AgentID          uint      `json:"agent_id"`
	AgentName        string    `json:"agent_name"`
	CustomerID       uint      `json:"customer_id"`
	CustomerCode     string    `json:"customer_code"`
	CustomerName     string    `json:"customer_name"`
	PaymentID        uint      `json:"payment_id"`
	ReceiptNo        string    `json:"receipt_no"`
	Amount           float64   `json:"amount"`
	CommissionRate   float64   `json:"commission_rate"`
	CommissionAmount float64   `json:"commission_amount"`
	Status           string    `json:"status"`
	CollectedAt      time.Time `json:"collected_at"`
}

func GetAgentCollections(c *gin.Context) {
	var agentID uint
	if c.GetString("role") == "agent" {
		agentID = c.GetUint("agent_id")
		if agentID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agent account is not linked"})
			return
		}
	}
	if value := c.Query("agent_id"); value != "" {
		if c.GetString("role") == "agent" {
			value = strconv.FormatUint(uint64(agentID), 10)
		}
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
	response := make([]agentCollectionResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, agentCollectionResponse{ID: row.ID, AgentID: row.AgentID, AgentName: row.Agent.Name, CustomerID: row.CustomerID, CustomerCode: row.Customer.CustomerCode, CustomerName: row.Customer.FullName, PaymentID: row.PaymentID, ReceiptNo: row.Payment.ReceiptNo, Amount: row.Amount, CommissionRate: row.CommissionRate, CommissionAmount: row.CommissionAmount, Status: row.Status, CollectedAt: row.CollectedAt})
	}
	c.JSON(http.StatusOK, gin.H{"collections": response, "count": summary.Count, "total_amount": summary.TotalAmount, "total_commission": summary.TotalCommission, "void_count": summary.VoidCount, "void_amount": summary.VoidAmount})
}
