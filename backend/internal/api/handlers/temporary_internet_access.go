package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type temporaryInternetAccessRequest struct {
	Days              int     `json:"days" binding:"required,min=1,max=7"`
	PromisedPaymentAt string  `json:"promised_payment_at"`
	PromisedAmount    float64 `json:"promised_amount"`
	RequestSource     string  `json:"request_source" binding:"required"`
	Reason            string  `json:"reason" binding:"required"`
}

func ListTemporaryInternetAccess(c *gin.Context) {
	customerID, customer, ok := temporaryAccessCustomer(c)
	if !ok {
		return
	}
	if !canAccessCustomer(c, customer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	items, err := services.ListTemporaryInternetAccess(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load temporary access history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"temporary_accesses": items})
}

func GrantTemporaryInternetAccess(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, customer, ok := temporaryAccessCustomer(c)
		if !ok {
			return
		}
		if !canAccessCustomer(c, customer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		var req temporaryInternetAccessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "days (1-7), request_source and reason are required"})
			return
		}
		var promisedAt *time.Time
		if value := strings.TrimSpace(req.PromisedPaymentAt); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "promised_payment_at must use RFC3339 format"})
				return
			}
			promisedAt = &parsed
		}
		item, err := services.GrantTemporaryInternetAccess(services.TemporaryInternetAccessGrantInput{
			CustomerID: customerID, Days: req.Days, PromisedPaymentAt: promisedAt,
			PromisedAmount: req.PromisedAmount, RequestSource: req.RequestSource,
			Reason: req.Reason, GrantedByUserID: c.GetUint("user_id"), Now: time.Now(),
		})
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		reconciliation, reconcileErr := services.ReconcileSubscriptionPPPSecretWithMikroTik(item.SubscriptionID, cfg.CredentialKey)
		response := gin.H{"temporary_access": item, "pppoe_reconciliation": reconciliation}
		if reconcileErr != nil {
			response["pppoe_reconciliation_error"] = reconcileErr.Error()
			c.JSON(http.StatusBadGateway, response)
			return
		}
		c.JSON(http.StatusCreated, response)
	}
}

type cancelTemporaryAccessRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func CancelTemporaryInternetAccess(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, customer, ok := temporaryAccessCustomer(c)
		if !ok {
			return
		}
		if !canAccessCustomer(c, customer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		id, err := strconv.ParseUint(c.Param("access_id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid temporary access ID"})
			return
		}
		var req cancelTemporaryAccessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
			return
		}
		item, err := services.CancelTemporaryInternetAccess(uint(id), customerID, c.GetUint("user_id"), req.Reason, time.Now())
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		reconciliation, reconcileErr := services.ReconcileSubscriptionPPPSecretWithMikroTik(item.SubscriptionID, cfg.CredentialKey)
		response := gin.H{"temporary_access": item, "pppoe_reconciliation": reconciliation}
		if reconcileErr != nil {
			response["pppoe_reconciliation_error"] = reconcileErr.Error()
			c.JSON(http.StatusBadGateway, response)
			return
		}
		c.JSON(http.StatusOK, response)
	}
}

func temporaryAccessCustomer(c *gin.Context) (uint, *models.Customer, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return 0, nil, false
	}
	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return 0, nil, false
	}
	return uint(id), customer, true
}
