package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

func SavePackageServicePolicy(
	row *models.PackageServicePolicy,
) error {
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

	if _, err := repositories.GetPackageByID(
		row.PackageID,
	); err != nil {
		return fmt.Errorf("load package: %w", err)
	}

	return repositories.SavePackageServicePolicy(row)
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
