package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func GetUserByUsername(username string) (*models.User, error) {
	return repositories.GetUserByUsername(username)
}

func GetUserByEmail(email string) (*models.User, error) {
	return repositories.GetUserByEmail(email)
}

func GetUserByID(id uint) (*models.User, error) {
	return repositories.GetUserByID(id)
}

func GetAllUsers() ([]models.User, error) {
	return repositories.GetAllUsers()
}

func CreateUser(user *models.User) error {
	return repositories.CreateUser(user)
}
