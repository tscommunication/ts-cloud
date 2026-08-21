package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type bulkCustomerDateExtensionRequest struct {
	CustomerIDs []uint `json:"customer_ids" binding:"required,min=1,max=100"`
	Days        int    `json:"days" binding:"required,min=1,max=365"`
	Reason      string `json:"reason" binding:"required"`
}

func BulkExtendCustomerInternetExpiry(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req bulkCustomerDateExtensionRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customer_ids, days (1-365), and reason are required"})
			return
		}
		actorID := c.GetUint("user_id")
		if actorID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
			return
		}
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		results := make([]gin.H, 0, len(req.CustomerIDs))
		for _, customerID := range req.CustomerIDs {
			subscriptions, err := services.GetSubscriptionsByCustomer(customerID)
			if err != nil || len(subscriptions) == 0 {
				results = append(results, gin.H{"customer_id": customerID, "success": false, "error": "Internet subscription not found"})
				continue
			}
			subscription := &subscriptions[0]
			base := subscription.ExpiryDate
			if base.Before(today) {
				base = today
			}
			newExpiry := base.AddDate(0, 0, req.Days)
			adjusted, err := services.AdjustSubscriptionDateWithoutBilling(subscription, newExpiry, strings.TrimSpace(req.Reason), actorID, now)
			if err != nil {
				results = append(results, gin.H{"customer_id": customerID, "success": false, "error": err.Error()})
				continue
			}
			_, reconcileErr := services.ReconcileSubscriptionPPPSecretWithMikroTik(adjusted.Subscription.ID, cfg.CredentialKey)
			row := gin.H{"customer_id": customerID, "subscription_id": adjusted.Subscription.ID, "new_expiry_date": newExpiry.Format("2006-01-02"), "success": true}
			if reconcileErr != nil {
				row["pppoe_reconciliation_error"] = reconcileErr.Error()
			}
			results = append(results, row)
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}
