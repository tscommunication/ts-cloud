package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetBillingSummary(c *gin.Context) {
	summary, err := services.GetBillingSummary(time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load billing summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func GetBillingRuns(c *gin.Context) {
	runs, err := services.GetRecentBillingRuns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load billing runs"})
		return
	}
	c.JSON(http.StatusOK, runs)
}

func RunBilling(c *gin.Context) {
	userID, _ := c.Get("user_id")
	triggeredBy, _ := userID.(uint)
	run, err := services.RunDueBilling(time.Now(), triggeredBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, run)
}

func GetCustomerLedger(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}
	customer, err := services.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if !canAccessCustomer(c, customer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}
	entries, err := services.GetCustomerLedger(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer ledger"})
		return
	}
	c.JSON(http.StatusOK, entries)
}
