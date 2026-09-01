package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type CustomerPortalIdentityBackfillResult struct {
	Created int
	Updated int
}

func inactiveCustomerPortalPassword() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate inactive customer credential: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ensureCustomerPortalIdentity creates the stable customer identity used by
// the portal. A supplied password is hashed only for the portal; it never
// writes to RouterOS.
func ensureCustomerPortalIdentity(
	tx *gorm.DB,
	customer *models.Customer,
	password string,
) (bool, bool, error) {
	password = strings.TrimSpace(password)
	var identity models.User
	err := tx.Where("customer_id = ?", customer.ID).First(&identity).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return false, false, err
	}

	if err == gorm.ErrRecordNotFound {
		active := password != ""
		if password == "" {
			var generateErr error
			password, generateErr = inactiveCustomerPortalPassword()
			if generateErr != nil {
				return false, false, generateErr
			}
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return false, false, fmt.Errorf("secure customer portal password: %w", hashErr)
		}
		customerID := customer.ID
		identity = models.User{
			Name:       customer.FullName,
			Username:   customer.CustomerCode,
			Email:      strings.ToLower(customer.CustomerCode) + "@customer.invalid",
			Password:   string(hash),
			Role:       "customer",
			Active:     active,
			CustomerID: &customerID,
		}
		if err := tx.Create(&identity).Error; err != nil {
			return false, false, err
		}
		return true, false, nil
	}

	if password == "" {
		return false, false, nil
	}
	if identity.Active {
		// A portal password may have been changed by an authorized user after
		// import. Never overwrite an existing active identity during backfill.
		return false, false, nil
	}
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashErr != nil {
		return false, false, fmt.Errorf("secure customer portal password: %w", hashErr)
	}
	if err := tx.Model(&identity).Updates(map[string]any{
		"name":     customer.FullName,
		"password": string(hash),
		"active":   true,
	}).Error; err != nil {
		return false, false, err
	}
	return false, true, nil
}

// BackfillCustomerPortalIdentities repairs imports made before portal
// identities were created. Only credentials already encrypted in TS-Cloud are
// used; RouterOS passwords are never read or changed.
func BackfillCustomerPortalIdentities(keyMaterial string) (CustomerPortalIdentityBackfillResult, error) {
	if strings.TrimSpace(keyMaterial) == "" {
		return CustomerPortalIdentityBackfillResult{}, fmt.Errorf("credential encryption key is required")
	}
	result := CustomerPortalIdentityBackfillResult{}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var accounts []models.CustomerInternetAccount
		if err := tx.Where("TRIM(COALESCE(pp_po_e_password_encrypted, '')) <> ''").Find(&accounts).Error; err != nil {
			return fmt.Errorf("load portal identity candidates: %w", err)
		}
		for index := range accounts {
			account := &accounts[index]
			password, err := security.DecryptSecret(account.PPPoEPasswordEncrypted, keyMaterial)
			if err != nil {
				return fmt.Errorf("decrypt customer internet credential %d: %w", account.ID, err)
			}
			var customer models.Customer
			if err := tx.First(&customer, account.CustomerID).Error; err != nil {
				return fmt.Errorf("load customer for internet credential %d: %w", account.ID, err)
			}
			created, updated, err := ensureCustomerPortalIdentity(tx, &customer, password)
			if err != nil {
				return fmt.Errorf("backfill customer portal identity %d: %w", customer.ID, err)
			}
			if created {
				result.Created++
			}
			if updated {
				result.Updated++
			}
		}
		return nil
	})
	if err != nil {
		return CustomerPortalIdentityBackfillResult{}, err
	}
	return result, nil
}
