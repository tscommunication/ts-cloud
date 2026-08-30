package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func ListDivisions() ([]models.Division, error) {
	var rows []models.Division

	err := database.DB.
		Order("name ASC").
		Find(&rows).Error

	return rows, err
}

func ListDistrictsByDivision(divisionID uint) ([]models.District, error) {
	var rows []models.District

	err := database.DB.
		Where("division_id = ?", divisionID).
		Order("name ASC").
		Find(&rows).Error

	return rows, err
}

func ListUpazilasByDistrict(districtID uint) ([]models.Upazila, error) {
	var rows []models.Upazila

	err := database.DB.
		Where("district_id = ?", districtID).
		Order("name ASC").
		Find(&rows).Error

	return rows, err
}

func ListPostOfficesByUpazila(upazilaID uint) ([]models.PostOffice, error) {
	var rows []models.PostOffice

	err := database.DB.
		Where("upazila_id = ?", upazilaID).
		Order("name ASC").
		Find(&rows).Error

	return rows, err
}

func ListPostOfficesByDistrict(districtID uint) ([]models.PostOffice, error) {
	var rows []models.PostOffice

	err := database.DB.
		Joins("JOIN upazilas ON upazilas.id = post_offices.upazila_id").
		Where("upazilas.district_id = ?", districtID).
		Order("post_offices.name ASC, post_offices.id ASC").
		Find(&rows).Error

	return rows, err
}
