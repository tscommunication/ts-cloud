package services

import (
	"errors"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

var validServiceTypes = map[string]bool{"FTP": true, "JELLYFIN": true, "IPTV": true, "CLOUD_STORAGE": true}
var validEntitlementStatuses = map[string]bool{"ACTIVE": true, "SUSPENDED": true, "EXPIRED": true, "DISABLED": true}

func SaveServiceEntitlement(row *models.ServiceEntitlement, password, key string) error {
	row.ServiceType = strings.ToUpper(strings.TrimSpace(row.ServiceType))
	row.Status = strings.ToUpper(strings.TrimSpace(row.Status))
	row.ServiceName, row.Username, row.Endpoint = strings.TrimSpace(row.ServiceName), strings.TrimSpace(row.Username), strings.TrimSpace(row.Endpoint)
	if row.CustomerID == 0 || !validServiceTypes[row.ServiceType] || row.ServiceName == "" || !validEntitlementStatuses[row.Status] {
		return errors.New("customer, valid service type, service name and status are required")
	}
	if row.SubscriptionID != nil {
		var subscription models.Subscription
		if err := database.DB.First(&subscription, *row.SubscriptionID).Error; err != nil || subscription.CustomerID != row.CustomerID {
			return errors.New("subscription does not belong to customer")
		}
	}
	if password != "" {
		encrypted, err := security.EncryptSecret(password, key)
		if err != nil {
			return err
		}
		row.PasswordEncrypted = encrypted
	}
	if row.ID == 0 && row.Username != "" && row.PasswordEncrypted == "" {
		return errors.New("password is required when username is configured")
	}
	if row.ID == 0 {
		return database.DB.Create(row).Error
	}
	return database.DB.Save(row).Error
}

func ListServiceEntitlements(customerID uint) ([]models.ServiceEntitlement, error) {
	var rows []models.ServiceEntitlement
	query := database.DB.Preload("Customer").Order("service_type, service_name, id")
	if customerID != 0 {
		query = query.Where("customer_id = ?", customerID)
	}
	return rows, query.Find(&rows).Error
}

func GetServiceEntitlement(id uint) (*models.ServiceEntitlement, error) {
	var row models.ServiceEntitlement
	return &row, database.DB.First(&row, id).Error
}
func DeleteServiceEntitlement(id uint) error {
	return database.DB.Delete(&models.ServiceEntitlement{}, id).Error
}
func DecryptServiceEntitlementPassword(row *models.ServiceEntitlement, key string) (string, error) {
	if row.PasswordEncrypted == "" {
		return "", nil
	}
	return security.DecryptSecret(row.PasswordEncrypted, key)
}
