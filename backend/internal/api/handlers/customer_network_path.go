package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// GetCustomerNetworkPath returns recorded access-path fields and, when the
// recorded ONU identity matches OLT inventory, its latest real optical sample.
func GetCustomerNetworkPath(cfg *config.Config) gin.HandlerFunc {
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

		path, err := services.GetCustomerNetworkPath(
			c.Request.Context(),
			uint(id),
			cfg.CredentialKey,
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"profile": nil, "onu": nil, "optical": nil})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer network path"})
			return
		}

		response := gin.H{"profile": path.Profile, "onu": path.ONU, "optical": path.LatestOptical}
		if path.ONU != nil && path.ONU.NetworkDevice != nil {
			response["olt_name"] = path.ONU.NetworkDevice.Name
			response["olt_code"] = path.ONU.NetworkDevice.Code
		}
		c.JSON(http.StatusOK, response)
	}
}
