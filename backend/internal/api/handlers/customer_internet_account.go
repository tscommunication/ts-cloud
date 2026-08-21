package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type customerInternetCredentialRequest struct {
	RouterID            uint    `json:"router_id"`
	PPPoEUsername       string  `json:"pppoe_username"`
	PPPoEPassword       string  `json:"pppoe_password" binding:"required"`
	MACAddress          *string `json:"mac_address"`
	StaticIPAddress     *string `json:"static_ip_address"`
	SyncIntervalMinutes int     `json:"sync_interval_minutes"`
}

func customerInternetAccountResponse(accountID, customerID, routerID uint, username, password, status, macAddress, staticIPAddress string, syncIntervalMinutes int) gin.H {
	return gin.H{"id": accountID, "customer_id": customerID, "router_id": routerID, "pppoe_username": username, "pppoe_password": password, "status": status, "mac_address": macAddress, "static_ip_address": staticIPAddress, "sync_interval_minutes": syncIntervalMinutes}
}

func GetCustomerInternetCredential(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		account, password, err := services.GetCustomerInternetCredential(uint(id), cfg.CredentialKey)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, nil)
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, customerInternetAccountResponse(account.ID, account.CustomerID, account.RouterID, account.PPPoEUsername, password, account.Status, account.MACAddress, account.StaticIPAddress, account.SyncIntervalMinutes))
	}
}

func SaveCustomerInternetCredential(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		var req customerInternetCredentialRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "PPPoE password is required"})
			return
		}
		if c.GetString("role") == "agent" {
			allowed, permissionErr := repositories.AgentHasRouter(c.GetUint("agent_id"), req.RouterID)
			if permissionErr != nil || !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "Router is not assigned to this agent"})
				return
			}
		}
		allowIdentityEdit := c.GetString("role") != "agent"
		account, err := services.SaveCustomerInternetCredential(uint(id), services.CustomerInternetCredentialInput{RouterID: req.RouterID, PPPoEUsername: req.PPPoEUsername, PPPoEPassword: req.PPPoEPassword, MACAddress: req.MACAddress, StaticIPAddress: req.StaticIPAddress, SyncIntervalMinutes: req.SyncIntervalMinutes}, cfg.CredentialKey, allowIdentityEdit)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		response := customerInternetAccountResponse(account.ID, account.CustomerID, account.RouterID, account.PPPoEUsername, req.PPPoEPassword, account.Status, account.MACAddress, account.StaticIPAddress, account.SyncIntervalMinutes)
		subscriptions, listErr := services.GetSubscriptionsByCustomer(uint(id))
		if listErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Credential saved, but linked subscriptions could not be loaded"})
			return
		}
		for _, subscription := range subscriptions {
			if subscription.Status != "ACTIVE" || subscription.InternetAccountID == nil || *subscription.InternetAccountID != account.ID {
				continue
			}
			reconciliation, reconcileErr := services.ReconcileSubscriptionPPPSecretCredentialWithMikroTik(subscription.ID, cfg.CredentialKey)
			if reconcileErr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "Credential saved, but MikroTik synchronization failed: " + reconcileErr.Error(), "internet_credential": response, "pppoe_reconciliation": reconciliation})
				return
			}
			response["pppoe_reconciliation"] = reconciliation
			break
		}
		c.JSON(http.StatusOK, response)
	}
}
