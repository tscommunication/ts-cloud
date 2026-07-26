package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateFTPUser(user *models.FTPUser) error {
	return database.DB.Create(user).Error
}

func GetFTPUsers() ([]models.FTPUser, error) {
	var users []models.FTPUser

	err := database.DB.Preload("Subscription").
		Preload("FTPServer").
		Find(&users).Error

	return users, err
}

func GetFTPUserByID(id uint) (*models.FTPUser, error) {
	var user models.FTPUser

	err := database.DB.Preload("Subscription").
		Preload("FTPServer").
		First(&user, id).Error

	return &user, err
}

func UpdateFTPUser(user *models.FTPUser) error {
	return database.DB.Save(user).Error
}

func DeleteFTPUser(id uint) error {
	return database.DB.Delete(&models.FTPUser{}, id).Error
}
