package services

import (
	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type ProvisionResult struct {
	LinuxUserCreated bool
}

// ProvisionFTPUserSafe provisions a user with rollback support.
func (p *ProvisioningService) ProvisionFTPUserSafe(
	user *models.FTPUser,
) error {
	return p.ProvisionFTPUserSafeWithPassword(user, user.Password)
}

func (p *ProvisioningService) ProvisionFTPUserSafeWithPassword(
	user *models.FTPUser,
	password string,
) error {

	result := &ProvisionResult{}

	// --------------------------------------------------
	// Step 1 - Create Linux User
	// --------------------------------------------------

	if err := linux.CreateUser(
		user.Username,
		user.HomeDirectory,
	); err != nil {
		return err
	}

	result.LinuxUserCreated = true

	// --------------------------------------------------
	// Step 2 - Set Password
	// --------------------------------------------------

	if err := linux.SetPassword(
		user.Username,
		password,
	); err != nil {

		p.Rollback(result, user)

		return err
	}

	// --------------------------------------------------
	// Step 3 - Create Home Directory
	// --------------------------------------------------

	if err := linux.CreateHomeDirectory(
		user.HomeDirectory,
	); err != nil {

		p.Rollback(result, user)

		return err
	}

	// --------------------------------------------------
	// Step 4 - Change Owner
	// --------------------------------------------------

	if err := linux.ChangeOwner(
		user.HomeDirectory,
		user.Username,
	); err != nil {

		p.Rollback(result, user)

		return err
	}

	// --------------------------------------------------
	// Step 5 - Set Permissions
	// --------------------------------------------------

	if err := linux.SetPermissions(
		user.HomeDirectory,
	); err != nil {

		p.Rollback(result, user)

		return err
	}

	// --------------------------------------------------
	// Step 6 - Apply Disk Quota
	// --------------------------------------------------

	if err := linux.SetQuota(
		user.Username,
		user.StorageQuotaGB,
	); err != nil {

		p.Rollback(result, user)

		return err
	}

	return nil
}

// Rollback removes created Linux resources.
func (p *ProvisioningService) Rollback(
	result *ProvisionResult,
	user *models.FTPUser,
) {

	if result.LinuxUserCreated {
		_ = linux.DeleteUser(user.Username)
	}
}
