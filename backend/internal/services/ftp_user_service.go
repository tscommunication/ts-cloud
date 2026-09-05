package services

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
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
	ftpLinuxUserExists  = linux.UserExists
	ftpLinuxLockUser    = linux.LockUser
	ftpLinuxUnlockUser  = linux.UnlockUser
	ftpLinuxSetPassword = linux.SetPassword
	ftpLinuxRenameUser  = linux.RenameUser
	ftpUpdateUser       = repositories.UpdateFTPUser
	ftpProvisionUser    = func(user *models.FTPUser) error {
		return NewProvisioningService().ProvisionFTPUserSafe(user)
	}
	ftpProvisionUserWithPassword = func(user *models.FTPUser, password string) error {
		return NewProvisioningService().ProvisionFTPUserSafeWithPassword(user, password)
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

func GetSingleActiveFTPServer() (*models.FTPServer, error) {
	servers, err := repositories.GetActiveFTPServers()
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, errors.New("no active FTP server is configured")
	}

	if len(servers) != 1 {
		return nil, fmt.Errorf(
			"expected exactly one active FTP server, found %d",
			len(servers),
		)
	}

	return &servers[0], nil
}

func BuildManagedFTPUser(
	subscription *models.Subscription,
	account *models.CustomerInternetAccount,
	entitlement *models.ServiceEntitlement,
	server *models.FTPServer,
) (*models.FTPUser, error) {
	if subscription == nil || subscription.ID == 0 {
		return nil, errors.New("subscription is required")
	}
	if account == nil || account.ID == 0 {
		return nil, errors.New("customer internet account is required")
	}
	if entitlement == nil || entitlement.ID == 0 {
		return nil, errors.New("FTP service entitlement is required")
	}
	if server == nil || server.ID == 0 {
		return nil, errors.New("FTP server is required")
	}

	username := strings.TrimSpace(account.PPPoEUsername)
	if username == "" {
		return nil, errors.New("PPPoE username is required")
	}

	root := filepath.Clean(strings.TrimSpace(server.RootPath))
	if root == "." || root == "/" || strings.TrimSpace(server.RootPath) == "" {
		return nil, errors.New("FTP server root path is invalid")
	}

	home := filepath.Join(root, username)
	if filepath.Dir(home) != root {
		return nil, errors.New("PPPoE username cannot be used safely as an FTP home directory")
	}

	status := strings.ToUpper(strings.TrimSpace(entitlement.Status))
	if status == "" {
		status = "ACTIVE"
	}

	quota := entitlement.QuotaGB
	if quota < 0 {
		return nil, errors.New("FTP quota cannot be negative")
	}

	return &models.FTPUser{
		CustomerID:           subscription.CustomerID,
		SubscriptionID:       subscription.ID,
		ServiceEntitlementID: &entitlement.ID,
		FTPServerID:          server.ID,
		Username:             username,

		// Managed FTP credentials come from the encrypted customer PPPoE
		// credential and are supplied to Linux provisioning only in memory.
		Password: "",

		HomeDirectory:  home,
		StorageQuotaGB: quota,
		Status:         status,
	}, nil
}

func ProvisionManagedFTPUser(
	user *models.FTPUser,
	password string,
) error {
	if user == nil {
		return errors.New("FTP user is required")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("PPPoE credential is required for FTP provisioning")
	}

	return ftpProvisionUserWithPassword(user, password)
}

func EnsureManagedFTPUserProjection(
	subscription *models.Subscription,
	account *models.CustomerInternetAccount,
	entitlement *models.ServiceEntitlement,
	server *models.FTPServer,
) (*models.FTPUser, bool, error) {
	if entitlement == nil || entitlement.ID == 0 {
		return nil, false, errors.New("FTP service entitlement is required")
	}

	existing, err := repositories.GetFTPUserByServiceEntitlementID(
		entitlement.ID,
	)

	if err == nil {
		expected, buildErr := BuildManagedFTPUser(
			subscription,
			account,
			entitlement,
			server,
		)
		if buildErr != nil {
			return nil, false, buildErr
		}

		oldUsername := existing.Username
		oldHome := existing.HomeDirectory

		usernameChanged := oldUsername != expected.Username ||
			oldHome != expected.HomeDirectory

		renamedLinuxUser := false

		if usernameChanged {
			usernameOwner, ownerErr :=
				repositories.GetFTPUserByUsername(expected.Username)

			if ownerErr == nil && usernameOwner.ID != existing.ID {
				return nil, false, fmt.Errorf(
					"FTP username %q is already owned by FTP user %d",
					expected.Username,
					usernameOwner.ID,
				)
			}

			if ownerErr != nil &&
				!errors.Is(ownerErr, gorm.ErrRecordNotFound) {
				return nil, false, ownerErr
			}

			oldLinuxExists := ftpLinuxUserExists(oldUsername)
			newLinuxExists := ftpLinuxUserExists(expected.Username)

			if oldLinuxExists {
				if newLinuxExists {
					return nil, false, fmt.Errorf(
						"linux user %q already exists",
						expected.Username,
					)
				}

				if err := ftpLinuxRenameUser(
					oldUsername,
					expected.Username,
					oldHome,
					expected.HomeDirectory,
				); err != nil {
					return nil, false, err
				}

				renamedLinuxUser = true
			} else if newLinuxExists {
				return nil, false, fmt.Errorf(
					"linux user %q exists while managed FTP projection still owns %q",
					expected.Username,
					oldUsername,
				)
			}
		}

		existing.CustomerID = expected.CustomerID
		existing.SubscriptionID = expected.SubscriptionID
		existing.FTPServerID = expected.FTPServerID
		existing.Username = expected.Username
		existing.HomeDirectory = expected.HomeDirectory
		existing.StorageQuotaGB = expected.StorageQuotaGB
		existing.Status = expected.Status

		// Never persist the canonical PPPoE credential in FTPUser.
		existing.Password = ""

		if err := ftpUpdateUser(existing); err != nil {
			if renamedLinuxUser {
				rollbackErr := ftpLinuxRenameUser(
					expected.Username,
					oldUsername,
					expected.HomeDirectory,
					oldHome,
				)
				if rollbackErr != nil {
					return nil, false, fmt.Errorf(
						"update FTP projection: %v; rollback Linux rename: %w",
						err,
						rollbackErr,
					)
				}
			}

			return nil, false, err
		}

		return existing, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	newUser, err := BuildManagedFTPUser(
		subscription,
		account,
		entitlement,
		server,
	)
	if err != nil {
		return nil, false, err
	}

	// Refuse to adopt or overwrite a legacy/manual FTP account that happens
	// to use the same username.
	usernameOwner, err := repositories.GetFTPUserByUsername(
		newUser.Username,
	)
	if err == nil {
		return nil, false, fmt.Errorf(
			"FTP username %q is already owned by FTP user %d",
			newUser.Username,
			usernameOwner.ID,
		)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if err := repositories.CreateFTPUser(newUser); err != nil {
		return nil, false, err
	}

	return newUser, true, nil
}

type ManagedFTPReconciliationResult struct {
	SubscriptionID       uint
	InternetAccountID    uint
	ServiceEntitlementID uint
	FTPUserID            uint
	FTPServerID          uint
	Username             string
	Status               string
	Action               string
	Executed             bool
}

func ReconcileManagedFTPForSubscription(
	subscription *models.Subscription,
	keyMaterial string,
) (ManagedFTPReconciliationResult, error) {
	var result ManagedFTPReconciliationResult

	if subscription == nil || subscription.ID == 0 {
		return result, errors.New("subscription is required")
	}

	result.SubscriptionID = subscription.ID

	if subscription.InternetAccountID == nil ||
		*subscription.InternetAccountID == 0 {
		return result, errors.New(
			"subscription is not linked to an internet account",
		)
	}

	account, err := GetCustomerInternetAccount(subscription.CustomerID)
	if err != nil {
		return result, err
	}
	if account == nil || account.ID == 0 {
		return result, errors.New("customer internet account is required")
	}

	// Only ACTIVE reconciliation needs the canonical PPPoE credential.
	// Suspended/expired/disabled lifecycle operations must still be able
	// to lock an existing FTP account when a legacy credential is blank.
	status := strings.ToUpper(strings.TrimSpace(subscription.Status))
	password := ""
	if status == "ACTIVE" {
		credentialAccount, credentialPassword, credentialErr :=
			GetCustomerInternetCredential(
				subscription.CustomerID,
				keyMaterial,
			)
		if credentialErr != nil {
			return result, credentialErr
		}
		if credentialAccount == nil ||
			credentialAccount.ID != account.ID {
			return result, errors.New(
				"customer internet credential account mismatch",
			)
		}
		password = credentialPassword
	}

	if account.ID != *subscription.InternetAccountID {
		return result, errors.New(
			"subscription internet account does not match customer internet account",
		)
	}

	result.InternetAccountID = account.ID
	result.Username = strings.TrimSpace(account.PPPoEUsername)

	server, err := GetSingleActiveFTPServer()
	if err != nil {
		return result, err
	}
	result.FTPServerID = server.ID

	entitlement, _, err := EnsureManagedFTPServiceEntitlement(
		subscription,
		account,
		server,
	)
	if err != nil {
		return result, err
	}

	result.ServiceEntitlementID = entitlement.ID
	result.Status = strings.ToUpper(strings.TrimSpace(entitlement.Status))

	user, _, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err != nil {
		return result, err
	}

	result.FTPUserID = user.ID
	result.Username = user.Username

	switch result.Status {
	case "ACTIVE":
		if strings.TrimSpace(password) == "" {
			return result, errors.New(
				"customer PPPoE credential is not configured",
			)
		}

		if ftpLinuxUserExists(user.Username) {
			if err := ftpLinuxSetPassword(
				user.Username,
				password,
			); err != nil {
				return result, err
			}

			if err := ftpLinuxUnlockUser(user.Username); err != nil {
				return result, err
			}

			result.Action = "PASSWORD_SYNC_AND_UNLOCK"
			result.Executed = true
		} else {
			if err := ProvisionManagedFTPUser(
				user,
				password,
			); err != nil {
				return result, err
			}

			result.Action = "PROVISION"
			result.Executed = true
		}

		if err := repositories.UpdateFTPUserStatus(
			user.ID,
			"ACTIVE",
		); err != nil {
			return result, err
		}

	case "SUSPENDED", "EXPIRED", "DISABLED":
		if ftpLinuxUserExists(user.Username) {
			if err := ftpLinuxLockUser(user.Username); err != nil {
				return result, err
			}

			result.Action = "LOCK"
			result.Executed = true
		} else {
			result.Action = "NO_LINUX_USER"
		}

		if err := repositories.UpdateFTPUserStatus(
			user.ID,
			result.Status,
		); err != nil {
			return result, err
		}

	default:
		return result, fmt.Errorf(
			"unsupported managed FTP status %q",
			result.Status,
		)
	}

	return result, nil
}
