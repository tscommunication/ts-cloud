package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

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
