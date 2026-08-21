package services

import (
	"errors"

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

	if err := linux.LockUser(user.Username); err != nil {
		return err
	}

	return repositories.UpdateFTPUserStatus(id, "suspended")
}

func EnableFTPUser(id uint) error {

	user, err := repositories.GetFTPUserByID(id)
	if err != nil {
		return err
	}

	if err := linux.UnlockUser(user.Username); err != nil {
		return err
	}

	return repositories.UpdateFTPUserStatus(id, "active")
}
