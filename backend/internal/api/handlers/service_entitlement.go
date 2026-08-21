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

type serviceEntitlementRequest struct {
	CustomerID     uint       `json:"customer_id"`
	SubscriptionID *uint      `json:"subscription_id"`
	ServiceType    string     `json:"service_type"`
	ServiceName    string     `json:"service_name"`
	Username       string     `json:"username"`
	Password       string     `json:"password"`
	Endpoint       string     `json:"endpoint"`
	Status         string     `json:"status"`
	ExpiryAt       *time.Time `json:"expiry_at"`
	QuotaGB        int        `json:"quota_gb"`
	Remarks        string     `json:"remarks"`
}

func entitlementResponse(row models.ServiceEntitlement, includeCustomer bool) gin.H {
	result := gin.H{"id": row.ID, "customer_id": row.CustomerID, "subscription_id": row.SubscriptionID, "service_type": row.ServiceType, "service_name": row.ServiceName, "username": row.Username, "endpoint": row.Endpoint, "status": row.Status, "expiry_at": row.ExpiryAt, "quota_gb": row.QuotaGB, "remarks": row.Remarks, "password_configured": row.PasswordEncrypted != ""}
	if includeCustomer {
		result["customer_code"], result["customer_name"] = row.Customer.CustomerCode, row.Customer.FullName
	}
	return result
}
func ListServiceEntitlements(c *gin.Context) {
	rows, err := services.ListServiceEntitlements(0)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to load service entitlements"})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, entitlementResponse(row, true))
	}
	c.JSON(200, gin.H{"entitlements": out})
}
func CreateServiceEntitlement(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req serviceEntitlementRequest
		if c.ShouldBindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		row := models.ServiceEntitlement{CustomerID: req.CustomerID, SubscriptionID: req.SubscriptionID, ServiceType: req.ServiceType, ServiceName: req.ServiceName, Username: req.Username, Endpoint: req.Endpoint, Status: req.Status, ExpiryAt: req.ExpiryAt, QuotaGB: req.QuotaGB, Remarks: req.Remarks}
		if err := services.SaveServiceEntitlement(&row, req.Password, cfg.CredentialKey); err != nil {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, entitlementResponse(row, false))
	}
}
func UpdateServiceEntitlement(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}
		row, err := services.GetServiceEntitlement(uint(id))
		if err != nil {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}
		var req serviceEntitlementRequest
		if c.ShouldBindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		row.CustomerID, row.SubscriptionID, row.ServiceType, row.ServiceName, row.Username, row.Endpoint, row.Status, row.ExpiryAt, row.QuotaGB, row.Remarks = req.CustomerID, req.SubscriptionID, req.ServiceType, req.ServiceName, req.Username, req.Endpoint, req.Status, req.ExpiryAt, req.QuotaGB, req.Remarks
		if err := services.SaveServiceEntitlement(row, req.Password, cfg.CredentialKey); err != nil {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, entitlementResponse(*row, false))
	}
}
func DeleteServiceEntitlement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	if err := services.DeleteServiceEntitlement(uint(id)); err != nil {
		c.JSON(422, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
func GetCustomerPortalServiceEntitlements(c *gin.Context) {
	customerID := c.GetUint("customer_id")
	if customerID == 0 {
		c.JSON(403, gin.H{"error": "Customer account is not linked"})
		return
	}
	rows, err := services.ListServiceEntitlements(customerID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to load services"})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, entitlementResponse(row, false))
	}
	c.JSON(200, gin.H{"entitlements": out})
}
