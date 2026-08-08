package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func GetSystemLogOffset(service string) (*models.SystemLogOffset, error) {

	var offset models.SystemLogOffset

	err := database.DB.
		Where("service_name = ?", service).
		First(&offset).Error

	if err != nil {
		return nil, err
	}

	return &offset, nil
}

func SaveSystemLogOffset(offset *models.SystemLogOffset) error {

	return database.DB.Save(offset).Error
}

func CreateSystemLogOffset(offset *models.SystemLogOffset) error {

	return database.DB.Create(offset).Error
}
