package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/services"
)

var customerInternetCredentialPostSaveReconciler = services.ReconcileCustomerInternetCredentialWithManagedServicesPostSave

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
		var oldRouterID uint
		var oldUsername string
		hasExistingAccount := false
		if existing, existingErr := services.GetCustomerInternetAccount(uint(id)); existingErr == nil {
			hasExistingAccount = true
			oldRouterID = existing.RouterID
			oldUsername = existing.PPPoEUsername
		} else if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": existingErr.Error()})
			return
		}
		if c.GetString("role") == "agent" {
			// Customer ownership has already been verified by canAccessCustomer.
			// An assigned agent may rotate the password of an existing customer
			// account even where the historical AgentRouter mapping is absent.
			// Router and username remain immutable for agents.
			if hasExistingAccount {
				if req.RouterID != oldRouterID || !strings.EqualFold(strings.TrimSpace(req.PPPoEUsername), strings.TrimSpace(oldUsername)) {
					c.JSON(http.StatusForbidden, gin.H{"error": "Agent can change password only; router and PPPoE username are managed by Admin"})
					return
				}
			} else {
				allowed, permissionErr := repositories.AgentHasRouter(c.GetUint("agent_id"), req.RouterID)
				if permissionErr != nil || !allowed {
					c.JSON(http.StatusForbidden, gin.H{"error": "Router is not assigned to this agent"})
					return
				}
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
			if subscription.Status != "ACTIVE" ||
				subscription.InternetAccountID == nil ||
				*subscription.InternetAccountID != account.ID {
				continue
			}

			identityChanged := oldRouterID != 0 &&
				(oldRouterID != account.RouterID ||
					!strings.EqualFold(
						strings.TrimSpace(oldUsername),
						strings.TrimSpace(account.PPPoEUsername),
					))

			reconciliation, reconcileErr :=
				customerInternetCredentialPostSaveReconciler(
					&subscription,
					cfg.CredentialKey,
					identityChanged,
					oldRouterID,
					oldUsername,
				)

			if reconcileErr != nil {
				c.JSON(
					http.StatusBadGateway,
					gin.H{
						"error": "Credential saved, but service synchronization could not be started: " +
							reconcileErr.Error(),
						"internet_credential": response,
					},
				)
				return
			}

			response["pppoe_reconciliation"] =
				reconciliation.PPP
			response["ftp_reconciliation"] =
				reconciliation.FTP

			if reconciliation.PPPError != "" {
				response["pppoe_reconciliation_error"] =
					reconciliation.PPPError
			}

			if reconciliation.FTPError != "" {
				response["ftp_reconciliation_error"] =
					reconciliation.FTPError
			}

			if reconciliation.PPPError != "" ||
				reconciliation.FTPError != "" {
				errorMessage :=
					"Credential saved, but service synchronization failed"

				if reconciliation.PPPError != "" &&
					reconciliation.FTPError == "" {
					errorMessage =
						"Credential saved, but MikroTik synchronization failed: " +
							reconciliation.PPPError
				} else if reconciliation.PPPError == "" &&
					reconciliation.FTPError != "" {
					errorMessage =
						"Credential saved, but FTP synchronization failed: " +
							reconciliation.FTPError
				} else {
					errorMessage +=
						": MikroTik: " +
							reconciliation.PPPError +
							"; FTP: " +
							reconciliation.FTPError
				}

				c.JSON(
					http.StatusBadGateway,
					gin.H{
						"error":                      errorMessage,
						"internet_credential":        response,
						"pppoe_reconciliation":       reconciliation.PPP,
						"ftp_reconciliation":         reconciliation.FTP,
						"pppoe_reconciliation_error": reconciliation.PPPError,
						"ftp_reconciliation_error":   reconciliation.FTPError,
					},
				)
				return
			}

			break
		}
		c.JSON(http.StatusOK, response)
	}
}
