package services

import (
	"errors"
	"fmt"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

func SyncApprovedPackageCatalog() (int, int, error) {
	created, updated := 0, 0
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for name, item := range importPackageCatalogs() {
			var pkg models.Package
			packageCode := fmt.Sprintf("CAT-%06d", item.SourceID)
			err := tx.Unscoped().Where("LOWER(name) = LOWER(?) OR package_code = ?", name, packageCode).First(&pkg).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				speed := catalogPackageSpeed(name)
				pkg = models.Package{PackageCode: packageCode, Name: name, Price: item.Rate, Commission: item.Commission, DownloadSpeed: speed, UploadSpeed: speed, ValidityDays: 30, MikroTikProfile: item.Profile, Status: "ACTIVE", Description: fmt.Sprintf("Approved Packages List.xlsx catalog; source ID=%d", item.SourceID)}
				if err := tx.Create(&pkg).Error; err != nil {
					return err
				}
				created++
				continue
			}
			if err != nil {
				return err
			}
			pkg.Name = name
			pkg.Price = item.Rate
			pkg.Commission = item.Commission
			pkg.DownloadSpeed = catalogPackageSpeed(name)
			pkg.UploadSpeed = pkg.DownloadSpeed
			pkg.ValidityDays = 30
			pkg.MikroTikProfile = item.Profile
			pkg.Status = "ACTIVE"
			pkg.DeletedAt = gorm.DeletedAt{}
			pkg.Description = fmt.Sprintf("Approved Packages List.xlsx catalog; source ID=%d", item.SourceID)
			if err := tx.Save(&pkg).Error; err != nil {
				return err
			}
			updated++
		}
		return nil
	})
	return created, updated, err
}

func CreatePackage(pkg *models.Package) error {
	return repositories.CreatePackage(pkg)
}

func GetPackages() ([]models.Package, error) {
	return repositories.GetPackages()
}

func GetPackageByID(id uint) (*models.Package, error) {
	return repositories.GetPackageByID(id)
}

func UpdatePackage(pkg *models.Package) error {
	return repositories.UpdatePackage(pkg)
}

func DeletePackage(id uint) error {
	return repositories.DeletePackage(id)
}

func GetLastPackage() (*models.Package, error) {
	return repositories.GetLastPackage()
}
