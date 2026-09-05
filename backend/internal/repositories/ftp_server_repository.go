package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateFTPServer(server *models.FTPServer) error {
	return database.DB.Create(server).Error
}

func GetFTPServers() ([]models.FTPServer, error) {
	var servers []models.FTPServer

	err := database.DB.Find(&servers).Error

	return servers, err
}

func GetFTPServerByID(id uint) (*models.FTPServer, error) {
	var server models.FTPServer

	err := database.DB.First(&server, id).Error

	return &server, err
}

func UpdateFTPServer(server *models.FTPServer) error {
	return database.DB.Save(server).Error
}

func DeleteFTPServer(id uint) error {
	return database.DB.Delete(&models.FTPServer{}, id).Error
}

func GetActiveFTPServers() ([]models.FTPServer, error) {
	var servers []models.FTPServer

	err := database.DB.
		Where("UPPER(status) = ?", "ACTIVE").
		Order("id ASC").
		Find(&servers).Error

	return servers, err
}
