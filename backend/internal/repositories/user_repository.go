package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func GetUserByUsername(username string) (*models.User, error) {

	var user models.User

	err := database.DB.
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByID(id uint) (*models.User, error) {

	var user models.User

	err := database.DB.
		First(&user, id).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
