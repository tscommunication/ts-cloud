package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomerPortalMe(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	customer, err := services.GetCustomerByID(customerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Customer not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load customer",
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToCustomerPortalMeResponse(customer))
}

func GetCustomerPortalSubscription(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetUint("customer_id")
		if customerID == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Customer account is not linked",
			})
			return
		}

		subscriptions, err := services.GetSubscriptionsByCustomer(customerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to load subscriptions",
			})
			return
		}

		response := dto.ToCustomerPortalSubscriptionResponses(subscriptions)
		_, password, credentialErr := services.GetCustomerInternetCredential(customerID, cfg.CredentialKey)
		if credentialErr != nil && !errors.Is(credentialErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load your PPPoE credential"})
			return
		}
		for index := range response {
			response[index].PPPoEPassword = password
		}
		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func GetCustomerPortalInvoices(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	invoices, err := services.GetInvoicesByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load invoices",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerPortalInvoiceResponses(invoices),
	)
}

func GetCustomerPortalPayments(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Customer account is not linked",
		})
		return
	}

	payments, err := services.GetPaymentsByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load payments",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToCustomerPortalPaymentResponses(payments),
	)
}

func GetCustomerPortalTemporaryAccess(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Customer account is not linked"})
		return
	}
	items, err := services.ListTemporaryInternetAccess(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load temporary access history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"temporary_accesses": items})
}

func GetCustomerPortalFTPEntitlements(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Customer account is not linked"})
		return
	}
	users, err := services.GetFTPUsersByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load FTP entitlements"})
		return
	}
	response := make([]gin.H, 0, len(users))
	for _, user := range users {
		response = append(response, gin.H{
			"id": user.ID, "subscription_id": user.SubscriptionID,
			"username": user.Username, "home_directory": user.HomeDirectory,
			"storage_quota_gb": user.StorageQuotaGB, "status": user.Status,
			"last_login": user.LastLogin, "last_ip": user.LastIP,
			"total_upload_bytes":   user.TotalUploadBytes,
			"total_download_bytes": user.TotalDownloadBytes,
			"server_name":          user.FTPServer.Name, "server_host": user.FTPServer.Host,
			"server_port": user.FTPServer.Port,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ftp_entitlements": response})
}
