package services

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateFTPUser(user *models.FTPUser) error {

	if user.SubscriptionID == 0 {
		return errors.New("subscription is required")
	}

	if user.FTPServerID == 0 {
		return errors.New("ftp server is required")
	}

	if user.Username == "" {
		return errors.New("username is required")
	}

	if user.HomeDirectory == "" {
		return errors.New("home directory is required")
	}

	return repositories.CreateFTPUser(user)
}

func GetFTPUsers() ([]models.FTPUser, error) {
	return repositories.GetFTPUsers()
}

func GetFTPUserByID(id uint) (*models.FTPUser, error) {
	return repositories.GetFTPUserByID(id)
}

func UpdateFTPUser(user *models.FTPUser) error {
	return repositories.UpdateFTPUser(user)
}

func DeleteFTPUser(id uint) error {
	return repositories.DeleteFTPUser(id)
}
