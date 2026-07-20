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

func GetUsers(page, limit int, search, sort, order string) ([]models.User, int64, error) {
	return repositories.GetUsers(page, limit, search, sort, order)
}

func CreateUser(user *models.User) error {
	return repositories.CreateUser(user)
}

func UpdateUser(user *models.User) error {
	return repositories.UpdateUser(user)
}

func DeleteUser(id uint) error {
	return repositories.DeleteUser(id)
}
