package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func GetCustomerPortalConnection(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetUint("customer_id")
		if customerID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Customer account is not linked"})
			return
		}
		account, err := services.GetCustomerInternetAccount(customerID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, dto.CustomerPortalConnectionResponse{})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load internet connection"})
			return
		}
		response := dto.CustomerPortalConnectionResponse{
			PPPoEUsername:   account.PPPoEUsername,
			Status:          account.Status,
			MACAddress:      account.MACAddress,
			StaticIPAddress: account.StaticIPAddress,
		}
		if account.ExpiryDate != nil {
			response.ExpiryDate = account.ExpiryDate.Format("02-01-2006")
		}
		if pkg, packageErr := services.GetPackageByID(account.PackageID); packageErr == nil {
			response.PackageCode, response.PackageName = pkg.PackageCode, pkg.Name
		}
		if router, routerErr := services.GetNetworkRouter(account.RouterID); routerErr == nil {
			response.RouterCode, response.RouterName = router.Code, router.Name
		}
		session, sessionErr := services.GetNetworkRouterPPPoESessionForIdentity(account.RouterID, account.PPPoEUsername)
		if sessionErr == nil {
			response.Online, response.IPAddress, response.Uptime = true, session.Address, session.Uptime
			response.DownloadBps, response.UploadBps = session.RxRateBps, session.TxRateBps
			response.LastSeenAt = session.LastSeenAt.Format(time.RFC3339)
		} else if !errors.Is(sessionErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load connection status"})
			return
		}
		// Optical information exposed to the customer portal is intentionally
		// limited to RX signal only. Network topology, ONU identity, OLT details,
		// SNMP information and management credentials are never included here.
		if path, pathErr := services.GetCustomerNetworkPath(
			c.Request.Context(),
			customerID,
			cfg.CredentialKey,
		); pathErr == nil &&
			path != nil &&
			path.LatestOptical != nil &&
			path.LatestOptical.RxPowerDBM != nil {
			response.RXSignalDBM = path.LatestOptical.RxPowerDBM
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetCustomerPortalLiveTraffic reads a direct traffic sample only for the
// authenticated customer's own active PPPoE session.
func GetCustomerPortalLiveTraffic(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetUint("customer_id")
		if customerID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Customer account is not linked"})
			return
		}
		account, err := services.GetCustomerInternetAccount(customerID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Internet account not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load internet connection"})
			return
		}
		session, err := services.GetNetworkRouterPPPoESessionForIdentity(account.RouterID, account.PPPoEUsername)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"online": false, "download_bps": 0, "upload_bps": 0})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load active PPPoE session"})
			return
		}
		traffic, err := services.GetNetworkPPPoESessionLiveTraffic(session.ID, cfg.CredentialKey)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read live traffic from router"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"online": true, "download_bps": traffic.DownloadBps, "upload_bps": traffic.UploadBps})
	}
}

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

	response := dto.ToCustomerPortalMeResponse(customer)
	if customer.AgentID != nil {
		if agent, agentErr := services.GetAgent(*customer.AgentID); agentErr == nil {
			response.AgentCode = agent.Code
			response.AgentName = agent.Name
			response.AgentMobile = agent.Mobile
		}
	}
	if customer.PopID != nil {
		if pop, popErr := services.GetPOP(*customer.PopID); popErr == nil {
			response.POPCode = pop.Code
			response.POPName = pop.Name
		}
	}
	c.JSON(http.StatusOK, response)
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
