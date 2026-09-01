package services

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type CustomerInternetCredentialInput struct {
	RouterID            uint
	PPPoEUsername       string
	PPPoEPassword       string
	MACAddress          *string
	StaticIPAddress     *string
	SyncIntervalMinutes int
}

func GetCustomerInternetAccount(customerID uint) (*models.CustomerInternetAccount, error) {
	var account models.CustomerInternetAccount
	err := database.DB.Where("customer_id = ?", customerID).First(&account).Error
	return &account, err
}

func GetCustomerInternetCredential(customerID uint, keyMaterial string) (*models.CustomerInternetAccount, string, error) {
	account, err := GetCustomerInternetAccount(customerID)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(account.PPPoEPasswordEncrypted) == "" {
		return nil, "", errors.New("customer PPPoE credential is not configured")
	}
	password, err := security.DecryptSecret(account.PPPoEPasswordEncrypted, keyMaterial)
	if err != nil {
		return nil, "", fmt.Errorf("decrypt customer PPPoE credential: %w", err)
	}
	return account, password, nil
}

// SaveCustomerInternetCredential atomically keeps the customer-owned PPPoE
// credential and the customer portal password in sync. CID remains the
// canonical username; PPPoE username is resolved as a login alias.
func SaveCustomerInternetCredential(customerID uint, input CustomerInternetCredentialInput, keyMaterial string, allowIdentityEdit bool) (*models.CustomerInternetAccount, error) {
	input.PPPoEUsername = strings.TrimSpace(input.PPPoEUsername)
	input.PPPoEPassword = strings.TrimSpace(input.PPPoEPassword)
	if customerID == 0 {
		return nil, fmt.Errorf("customer is required")
	}
	if input.MACAddress != nil && strings.TrimSpace(*input.MACAddress) != "" {
		if _, err := net.ParseMAC(strings.TrimSpace(*input.MACAddress)); err != nil {
			return nil, fmt.Errorf("MAC address is invalid")
		}
	}
	if input.StaticIPAddress != nil && strings.TrimSpace(*input.StaticIPAddress) != "" &&
		net.ParseIP(strings.TrimSpace(*input.StaticIPAddress)) == nil {
		return nil, fmt.Errorf("static IP address is invalid")
	}
	if input.SyncIntervalMinutes != 0 && input.SyncIntervalMinutes != 30 {
		return nil, fmt.Errorf("sync interval must be 30 minutes")
	}

	var saved models.CustomerInternetAccount
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var customer models.Customer
		if err := tx.First(&customer, customerID).Error; err != nil {
			return fmt.Errorf("customer not found")
		}
		var account models.CustomerInternetAccount
		err := tx.Where("customer_id = ?", customerID).First(&account).Error
		creating := errors.Is(err, gorm.ErrRecordNotFound)
		if err != nil && !creating {
			return err
		}
		if creating && input.PPPoEPassword == "" {
			return fmt.Errorf("PPPoE password is required")
		}

		// Blank password on an existing account means preserve the current
		// credential. This supports legacy RouterOS adoption records whose
		// password is intentionally not stored in TS-Cloud.
		passwordChanged := input.PPPoEPassword != ""

		if !creating && input.PPPoEPassword != "" && len(input.PPPoEPassword) < 8 {
			if strings.TrimSpace(account.PPPoEPasswordEncrypted) == "" {
				return fmt.Errorf("new PPPoE password must be at least 8 characters")
			}

			existingPassword, decryptErr := security.DecryptSecret(
				account.PPPoEPasswordEncrypted,
				keyMaterial,
			)
			if decryptErr != nil {
				return fmt.Errorf("decrypt existing PPPoE credential: %w", decryptErr)
			}
			passwordChanged = input.PPPoEPassword != existingPassword
		}

		if input.PPPoEPassword != "" &&
			passwordChanged &&
			len(input.PPPoEPassword) < 8 {
			return fmt.Errorf("new PPPoE password must be at least 8 characters")
		}
		if creating || allowIdentityEdit {
			if input.PPPoEUsername == "" || input.RouterID == 0 {
				return fmt.Errorf("PPPoE username and router are required")
			}
			account.CustomerID = customerID
			account.RouterID = input.RouterID
			account.PPPoEUsername = input.PPPoEUsername
			account.Status = "ACTIVE"
		} else if input.PPPoEUsername != "" && !strings.EqualFold(input.PPPoEUsername, account.PPPoEUsername) {
			return fmt.Errorf("agent cannot change PPPoE username")
		}
		if creating {
			account.AccountCode = fmt.Sprintf("NET-%06d", customerID)
		}
		if passwordChanged {
			encrypted, err := security.EncryptSecret(input.PPPoEPassword, keyMaterial)
			if err != nil {
				return fmt.Errorf("encrypt customer PPPoE credential: %w", err)
			}
			account.PPPoEPasswordEncrypted = encrypted
		}
		if input.MACAddress != nil {
			account.MACAddress = strings.TrimSpace(*input.MACAddress)
		}
		if input.StaticIPAddress != nil {
			account.StaticIPAddress = strings.TrimSpace(*input.StaticIPAddress)
		}
		if input.SyncIntervalMinutes != 0 {
			account.SyncIntervalMinutes = input.SyncIntervalMinutes
		} else if creating {
			account.SyncIntervalMinutes = 30
		}
		if creating {
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&account).Error; err != nil {
			return err
		}

		var identity models.User
		identityErr := tx.Where("customer_id = ?", customerID).First(&identity).Error
		if errors.Is(identityErr, gorm.ErrRecordNotFound) {
			// Older credential-less imports legitimately have no portal identity.
			// Once an authorized staff member supplies a password, create the
			// customer login in the same transaction. RouterOS is reconciled by
			// the caller only after this transaction succeeds.
			if !passwordChanged {
				return fmt.Errorf("a PPPoE password is required to create the customer portal account")
			}
			if _, _, err := ensureCustomerPortalIdentity(tx, &customer, input.PPPoEPassword); err != nil {
				return fmt.Errorf("create customer portal account: %w", err)
			}
		} else if identityErr != nil {
			return fmt.Errorf("load customer auth account: %w", identityErr)
		} else {
			// Customer portal authentication and PPPoE deliberately share one
			// credential. CID remains the canonical login and PPPoE username is an
			// alias; every authorized PPPoE password change updates the portal hash.
			identityUpdates := map[string]interface{}{"active": true}
			if passwordChanged {
				hash, err := bcrypt.GenerateFromPassword([]byte(input.PPPoEPassword), bcrypt.DefaultCost)
				if err != nil {
					return fmt.Errorf("secure customer portal password: %w", err)
				}
				identityUpdates["password"] = string(hash)
			}
			if err := tx.Model(&identity).Updates(identityUpdates).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.Subscription{}).
			Where("customer_id = ? AND internet_account_id IS NULL", customerID).
			Update("internet_account_id", account.ID).Error; err != nil {
			return err
		}
		saved = account
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
