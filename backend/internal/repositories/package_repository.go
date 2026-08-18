package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreatePackage(pkg *models.Package) error {
	return database.DB.Create(pkg).Error
}

func GetPackages() ([]models.Package, error) {
	var packages []models.Package

	err := database.DB.
		Order("id DESC").
		Find(&packages).Error

	return packages, err
}

func GetPackageByID(id uint) (*models.Package, error) {
	var pkg models.Package

	err := database.DB.First(&pkg, id).Error

	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

func UpdatePackage(pkg *models.Package) error {
	return database.DB.Save(pkg).Error
}

func DeletePackage(id uint) error {
	return database.DB.Delete(&models.Package{}, id).Error
}

func GetLastPackage() (*models.Package, error) {
	var pkg models.Package

	err := database.DB.
		Order("id DESC").
		First(&pkg).Error

	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

func ListActivePackages() ([]models.Package, error) {
	var packages []models.Package

	err := database.DB.
		Where("status = ?", "ACTIVE").
		Order("name ASC, id ASC").
		Find(&packages).Error

	return packages, err
}
