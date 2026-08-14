package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func ListPOPs() ([]models.POP, error) {
	var rows []models.POP
	err := database.DB.Order("name ASC").Find(&rows).Error
	return rows, err
}

func GetPOP(id uint) (*models.POP, error) {
	var row models.POP
	if err := database.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreatePOP(row *models.POP) error { return database.DB.Create(row).Error }
func UpdatePOP(row *models.POP) error { return database.DB.Save(row).Error }

func ListAgents(popID uint) ([]models.Agent, error) {
	var rows []models.Agent
	query := database.DB.Preload("POP").Order("name ASC")
	if popID > 0 {
		query = query.Where("pop_id = ?", popID)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func GetAgent(id uint) (*models.Agent, error) {
	var row models.Agent
	if err := database.DB.Preload("POP").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateAgent(row *models.Agent) error { return database.DB.Create(row).Error }
func UpdateAgent(row *models.Agent) error { return database.DB.Save(row).Error }
