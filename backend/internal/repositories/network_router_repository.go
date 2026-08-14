package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func ListNetworkRouters() ([]models.NetworkRouter, error) {
	var rows []models.NetworkRouter
	err := database.DB.Preload("POP").Order("name, id").Find(&rows).Error
	return rows, err
}

func GetNetworkRouter(id uint) (*models.NetworkRouter, error) {
	var row models.NetworkRouter
	if err := database.DB.Preload("POP").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateNetworkRouter(row *models.NetworkRouter) error { return database.DB.Create(row).Error }
func UpdateNetworkRouter(row *models.NetworkRouter) error { return database.DB.Save(row).Error }
