package repositories

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func GetPackageServicePolicy(
	packageID uint,
	serviceType string,
) (*models.PackageServicePolicy, error) {
	var row models.PackageServicePolicy

	err := database.DB.
		Where(
			"package_id = ? AND service_type = ?",
			packageID,
			strings.ToUpper(strings.TrimSpace(serviceType)),
		).
		First(&row).Error
	if err != nil {
		return nil, err
	}

	return &row, nil
}

func ListPackageServicePolicies(
	packageID uint,
) ([]models.PackageServicePolicy, error) {
	var rows []models.PackageServicePolicy

	query := database.DB.
		Order("service_type ASC, id ASC")

	if packageID != 0 {
		query = query.Where("package_id = ?", packageID)
	}

	return rows, query.Find(&rows).Error
}

func SavePackageServicePolicy(
	row *models.PackageServicePolicy,
) error {
	if row == nil {
		return errors.New("package service policy is required")
	}

	var existing models.PackageServicePolicy

	err := database.DB.
		Unscoped().
		Where(
			"package_id = ? AND service_type = ?",
			row.PackageID,
			row.ServiceType,
		).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.DB.Create(row).Error
	}
	if err != nil {
		return err
	}

	existing.Enabled = row.Enabled
	existing.QuotaGB = row.QuotaGB
	existing.ConfigJSON = row.ConfigJSON
	existing.Remarks = row.Remarks
	existing.DeletedAt = gorm.DeletedAt{}

	if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
		return err
	}

	*row = existing
	return nil
}
