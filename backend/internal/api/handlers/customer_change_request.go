package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type customerChangeRequestInput struct {
	CustomerID     uint   `json:"customer_id" binding:"required"`
	Type           string `json:"type" binding:"required"`
	Reason         string `json:"reason" binding:"required"`
	CurrentValue   string `json:"current_value"`
	RequestedValue string `json:"requested_value"`
}
type customerChangeReviewInput struct {
	Reason string `json:"reason"`
}

func CreateCustomerChangeRequest(c *gin.Context) {
	var input customerChangeRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id, type and reason are required"})
		return
	}
	row := models.CustomerChangeRequest{CustomerID: input.CustomerID, Type: input.Type, Reason: input.Reason, CurrentValue: input.CurrentValue, RequestedValue: input.RequestedValue}
	if err := services.CreateCustomerChangeRequest(&row, c.GetUint("agent_id"), c.GetUint("user_id"), time.Now()); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

func ListCustomerChangeRequests(c *gin.Context) {
	agentID := uint(0)
	if c.GetString("role") == "agent" {
		agentID = c.GetUint("agent_id")
		if agentID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agent account is not linked"})
			return
		}
	}
	rows, err := services.ListCustomerChangeRequests(c.Query("status"), agentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": rows})
}

func GetCustomerChangeRequestOptions(c *gin.Context) {
	options, err := services.GetCustomerChangeRequestOptions(c.GetUint("agent_id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, options)
}

func reviewCustomerChangeRequest(c *gin.Context, approve bool, cfg *config.Config) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}
	var input customerChangeReviewInput
	_ = c.ShouldBindJSON(&input)
	row, err := services.ReviewCustomerChangeRequest(uint(id), c.GetUint("user_id"), approve, input.Reason, time.Now(), cfg.CredentialKey)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}
func ApproveCustomerChangeRequest(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) { reviewCustomerChangeRequest(c, true, cfg) }
}
func RejectCustomerChangeRequest(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) { reviewCustomerChangeRequest(c, false, cfg) }
}
