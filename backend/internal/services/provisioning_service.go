package services

import (
	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type ProvisioningService struct {
}

func NewProvisioningService() *ProvisioningService {
	return &ProvisioningService{}
}

// ProvisionFTPUser provisions a Linux FTP account.
func (p *ProvisioningService) ProvisionFTPUser(
	user *models.FTPUser,
) error {

	// --------------------------------------------------
	// Create Linux User
	// --------------------------------------------------

	if err := linux.CreateUser(
		user.Username,
		user.HomeDirectory,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Set Password
	// --------------------------------------------------

	if err := linux.SetPassword(
		user.Username,
		user.Password,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Create Home Directory
	// --------------------------------------------------

	if err := linux.CreateHomeDirectory(
		user.HomeDirectory,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Change Owner
	// --------------------------------------------------

	if err := linux.ChangeOwner(
		user.HomeDirectory,
		user.Username,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Set Permissions
	// --------------------------------------------------

	if err := linux.SetPermissions(
		user.HomeDirectory,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Apply Disk Quota
	// --------------------------------------------------

	if err := linux.SetQuota(
		user.Username,
		user.StorageQuotaGB,
	); err != nil {
		return err
	}

	return nil
}
