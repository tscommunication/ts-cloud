package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

func SavePackageServicePolicy(
	row *models.PackageServicePolicy,
) error {
	if err := validatePackageServicePolicy(row); err != nil {
		return err
	}

	if _, err := repositories.GetPackageByID(row.PackageID); err != nil {
		return fmt.Errorf("load package: %w", err)
	}

	return repositories.SavePackageServicePolicy(row)
}

// CreatePackageWithFTPPolicy creates a package and its initial FTP policy as
// one unit of work. A failed policy write therefore cannot leave a package
// without the policy chosen by its creator.
func CreatePackageWithFTPPolicy(
	pkg *models.Package,
	ftpEnabled bool,
	ftpQuotaGB int,
) error {
	if pkg == nil {
		return errors.New("package is required")
	}

	policy := models.PackageServicePolicy{
		ServiceType: "FTP",
		Enabled:     ftpEnabled,
		QuotaGB:     ftpQuotaGB,
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkg).Error; err != nil {
			return err
		}
		policy.PackageID = pkg.ID
		if err := validatePackageServicePolicy(&policy); err != nil {
			return err
		}
		return savePackageServicePolicyTx(tx, &policy)
	})
}

// UpdatePackageWithFTPPolicy updates a package and, when a policy exists or
// is explicitly requested, its FTP policy in the same transaction. Legacy
// packages without a policy stay policy-free on ordinary edits.
func UpdatePackageWithFTPPolicy(
	pkg *models.Package,
	ftpEnabled bool,
	ftpQuotaGB int,
) error {
	if pkg == nil || pkg.ID == 0 {
		return errors.New("package is required")
	}

	policy := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     ftpEnabled,
		QuotaGB:     ftpQuotaGB,
	}
	if err := validatePackageServicePolicy(&policy); err != nil {
		return err
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.PackageServicePolicy
		err := tx.Unscoped().Where(
			"package_id = ? AND service_type = ?", pkg.ID, "FTP",
		).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		shouldSavePolicy := err == nil || ftpEnabled || ftpQuotaGB != 0

		if err := tx.Save(pkg).Error; err != nil {
			return err
		}
		if !shouldSavePolicy {
			return nil
		}
		return savePackageServicePolicyTx(tx, &policy)
	})
}

func validatePackageServicePolicy(row *models.PackageServicePolicy) error {
	if row == nil {
		return errors.New("package service policy is required")
	}

	if row.PackageID == 0 {
		return errors.New("package is required")
	}

	row.ServiceType =
		strings.ToUpper(strings.TrimSpace(row.ServiceType))
	row.ConfigJSON = strings.TrimSpace(row.ConfigJSON)
	row.Remarks = strings.TrimSpace(row.Remarks)

	if !validServiceTypes[row.ServiceType] {
		return fmt.Errorf(
			"unsupported package service type %q",
			row.ServiceType,
		)
	}

	if row.QuotaGB < 0 {
		return errors.New("service quota cannot be negative")
	}

	if row.Enabled &&
		row.ServiceType == "FTP" &&
		row.QuotaGB <= 0 {
		return errors.New(
			"enabled FTP package service requires a positive quota",
		)
	}

	if row.ConfigJSON != "" &&
		!json.Valid([]byte(row.ConfigJSON)) {
		return errors.New("service config_json must contain valid JSON")
	}

	return nil
}

func savePackageServicePolicyTx(
	tx *gorm.DB,
	row *models.PackageServicePolicy,
) error {
	var existing models.PackageServicePolicy
	err := tx.Unscoped().Where(
		"package_id = ? AND service_type = ?", row.PackageID, row.ServiceType,
	).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(row).Error
	}
	if err != nil {
		return err
	}

	existing.Enabled = row.Enabled
	existing.QuotaGB = row.QuotaGB
	existing.ConfigJSON = row.ConfigJSON
	existing.Remarks = row.Remarks
	existing.DeletedAt = gorm.DeletedAt{}
	if err := tx.Unscoped().Save(&existing).Error; err != nil {
		return err
	}
	*row = existing
	return nil
}

func GetPackageServicePolicy(
	packageID uint,
	serviceType string,
) (*models.PackageServicePolicy, error) {
	if packageID == 0 {
		return nil, errors.New("package is required")
	}

	serviceType =
		strings.ToUpper(strings.TrimSpace(serviceType))

	if !validServiceTypes[serviceType] {
		return nil, fmt.Errorf(
			"unsupported package service type %q",
			serviceType,
		)
	}

	return repositories.GetPackageServicePolicy(
		packageID,
		serviceType,
	)
}

func ListPackageServicePolicies(
	packageID uint,
) ([]models.PackageServicePolicy, error) {
	return repositories.ListPackageServicePolicies(packageID)
}

func GetPackageFTPPolicySummary(
	packageID uint,
) (bool, int, error) {
	row, err := repositories.GetPackageServicePolicy(
		packageID,
		"FTP",
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, nil
		}
		return false, 0, err
	}

	return row.Enabled, row.QuotaGB, nil
}
