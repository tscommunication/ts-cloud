package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ListDivisions() ([]models.Division, error) {
	return repositories.ListDivisions()
}

func ListDistrictsByDivision(divisionID uint) ([]models.District, error) {
	return repositories.ListDistrictsByDivision(divisionID)
}

func ListUpazilasByDistrict(districtID uint) ([]models.Upazila, error) {
	return repositories.ListUpazilasByDistrict(districtID)
}

func ListPostOfficesByUpazila(upazilaID uint) ([]models.PostOffice, error) {
	return repositories.ListPostOfficesByUpazila(upazilaID)
}

func ListPostOfficesByDistrict(districtID uint) ([]models.PostOffice, error) {
	return repositories.ListPostOfficesByDistrict(districtID)
}
