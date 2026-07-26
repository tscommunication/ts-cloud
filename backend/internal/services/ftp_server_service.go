package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateFTPServer(server *models.FTPServer) error {
	return repositories.CreateFTPServer(server)
}

func GetFTPServers() ([]models.FTPServer, error) {
	return repositories.GetFTPServers()
}

func GetFTPServerByID(id uint) (*models.FTPServer, error) {
	return repositories.GetFTPServerByID(id)
}

func UpdateFTPServer(server *models.FTPServer) error {
	return repositories.UpdateFTPServer(server)
}

func DeleteFTPServer(id uint) error {
	return repositories.DeleteFTPServer(id)
}
