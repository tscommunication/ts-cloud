package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
	"gorm.io/gorm"
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

func managedFTPStatus(subscriptionStatus string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(subscriptionStatus)) {
	case "ACTIVE":
		return "ACTIVE", nil
	case "SUSPENDED":
		return "SUSPENDED", nil
	case "EXPIRED":
		return "EXPIRED", nil
	case "DISCONNECTED":
		return "DISABLED", nil
	default:
		return "", fmt.Errorf(
			"unsupported subscription status %q for managed FTP",
			subscriptionStatus,
		)
	}
}

func EnsureManagedFTPServiceEntitlement(
	subscription *models.Subscription,
	account *models.CustomerInternetAccount,
	server *models.FTPServer,
) (*models.ServiceEntitlement, bool, error) {
	if subscription == nil || subscription.ID == 0 {
		return nil, false, errors.New("subscription is required")
	}
	if account == nil || account.ID == 0 {
		return nil, false, errors.New("customer internet account is required")
	}
	if server == nil || server.ID == 0 {
		return nil, false, errors.New("FTP server is required")
	}
	if subscription.CustomerID == 0 ||
		subscription.CustomerID != account.CustomerID {
		return nil, false, errors.New(
			"subscription and internet account customer do not match",
		)
	}
	if subscription.InternetAccountID == nil ||
		*subscription.InternetAccountID != account.ID {
		return nil, false, errors.New(
			"subscription is not linked to the internet account",
		)
	}

	username := strings.TrimSpace(account.PPPoEUsername)
	if username == "" {
		return nil, false, errors.New("PPPoE username is required")
	}

	status, err := managedFTPStatus(subscription.Status)
	if err != nil {
		return nil, false, err
	}

	keyValue := fmt.Sprintf("PPPOE_FTP:%d", account.ID)

	var entitlement models.ServiceEntitlement
	err = database.DB.
		Where("managed_key = ?", keyValue).
		First(&entitlement).Error

	if err == nil {
		updates := map[string]interface{}{
			"customer_id":     subscription.CustomerID,
			"subscription_id": subscription.ID,
			"service_type":    "FTP",
			"service_name":    "PPPoE FTP",
			"username":        username,
			"endpoint": fmt.Sprintf(
				"ftp://%s:%d",
				strings.TrimSpace(server.Host),
				server.Port,
			),
			"status": status,
		}

		if err := database.DB.Model(&entitlement).
			Updates(updates).Error; err != nil {
			return nil, false, err
		}

		if err := database.DB.First(
			&entitlement,
			entitlement.ID,
		).Error; err != nil {
			return nil, false, err
		}

		return &entitlement, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// Managed FTP credentials are intentionally NOT duplicated here.
	// The canonical encrypted credential remains on CustomerInternetAccount.
	entitlement = models.ServiceEntitlement{
		CustomerID:     subscription.CustomerID,
		SubscriptionID: &subscription.ID,
		ManagedKey:     &keyValue,
		ServiceType:    "FTP",
		ServiceName:    "PPPoE FTP",
		Username:       username,
		Endpoint: fmt.Sprintf(
			"ftp://%s:%d",
			strings.TrimSpace(server.Host),
			server.Port,
		),
		Status:  status,
		QuotaGB: 0,
		Remarks: "System-managed from PPPoE internet account",
	}

	if err := database.DB.Create(&entitlement).Error; err != nil {
		return nil, false, err
	}

	return &entitlement, true, nil
}
