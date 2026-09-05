package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateFTPUser(user *models.FTPUser) error {

	if user.SubscriptionID == 0 {
		return errors.New("subscription is required")
	}

	if user.FTPServerID == 0 {
		return errors.New("ftp server is required")
	}

	if user.Username == "" {
		return errors.New("username is required")
	}

	if user.Password == "" {
		return errors.New("password is required")
	}

	if user.HomeDirectory == "" {
		return errors.New("home directory is required")
	}

	subscription, err := repositories.GetSubscriptionByID(user.SubscriptionID)
	if err != nil {
		return err
	}
	user.CustomerID = subscription.CustomerID

	// -----------------------------------------------------
	// Linux Provisioning
	// -----------------------------------------------------

	provisioner := NewProvisioningService()

	if err := provisioner.ProvisionFTPUserSafe(user); err != nil {
		return err
	}

	// -----------------------------------------------------
	// Save into Database
	// -----------------------------------------------------

	if err := repositories.CreateFTPUser(user); err != nil {

		// Rollback Linux resources if database save fails
		provisioner.Rollback(
			&ProvisionResult{
				LinuxUserCreated: true,
			},
			user,
		)

		return err
	}

	return nil
}

func GetFTPUsers() ([]models.FTPUser, error) {
	return repositories.GetFTPUsers()
}

func GetFTPUserByID(id uint) (*models.FTPUser, error) {
	return repositories.GetFTPUserByID(id)
}

func GetFTPUsersByCustomer(customerID uint) ([]models.FTPUser, error) {
	return repositories.GetFTPUsersByCustomer(customerID)
}

func UpdateFTPUser(user *models.FTPUser) error {
	subscription, err := repositories.GetSubscriptionByID(user.SubscriptionID)
	if err != nil {
		return err
	}
	user.CustomerID = subscription.CustomerID
	return repositories.UpdateFTPUser(user)
}

func DeleteFTPUser(id uint) error {

	user, err := repositories.GetFTPUserByID(id)
	if err != nil {
		return err
	}

	// Delete Linux user (also removes home directory)
	if err := linux.DeleteUser(user.Username); err != nil {
		return err
	}

	// Delete database record
	if err := repositories.DeleteFTPUser(id); err != nil {
		return err
	}

	return nil
}
func SuspendFTPUser(id uint) error {

	user, err := repositories.GetFTPUserByID(id)
	if err != nil {
		return err
	}

	if err := ftpLinuxLockUser(user.Username); err != nil {
		return err
	}

	return repositories.UpdateFTPUserStatus(id, "suspended")
}

func EnableFTPUser(id uint) error {

	user, err := repositories.GetFTPUserByID(id)
	if err != nil {
		return err
	}

	if err := ftpLinuxUnlockUser(user.Username); err != nil {
		return err
	}

	return repositories.UpdateFTPUserStatus(id, "active")
}

var (
	ftpLinuxUserExists = linux.UserExists
	ftpLinuxLockUser   = linux.LockUser
	ftpLinuxUnlockUser = linux.UnlockUser
	ftpProvisionUser   = func(user *models.FTPUser) error {
		return NewProvisioningService().ProvisionFTPUserSafe(user)
	}
)

func ReconcileFTPServiceEntitlement(entitlement *models.ServiceEntitlement) error {
	if entitlement == nil || entitlement.ID == 0 {
		return errors.New("service entitlement is required")
	}

	if strings.ToUpper(strings.TrimSpace(entitlement.ServiceType)) != "FTP" {
		return nil
	}

	user, err := repositories.GetFTPUserByServiceEntitlementID(entitlement.ID)
	if err != nil {
		return fmt.Errorf("load FTP user for service entitlement %d: %w", entitlement.ID, err)
	}

	status := strings.ToUpper(strings.TrimSpace(entitlement.Status))

	switch status {
	case "ACTIVE":
		if ftpLinuxUserExists(user.Username) {
			if err := ftpLinuxUnlockUser(user.Username); err != nil {
				return err
			}
		} else {
			if err := ftpProvisionUser(user); err != nil {
				return err
			}
		}

		return repositories.UpdateFTPUserStatus(user.ID, "ACTIVE")

	case "SUSPENDED", "EXPIRED", "DISABLED":
		if ftpLinuxUserExists(user.Username) {
			if err := ftpLinuxLockUser(user.Username); err != nil {
				return err
			}
		}

		return repositories.UpdateFTPUserStatus(user.ID, status)

	default:
		return fmt.Errorf("unsupported FTP entitlement status %q", entitlement.Status)
	}
}
