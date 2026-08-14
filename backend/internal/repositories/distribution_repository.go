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
	query := database.DB.Preload("POP").Preload("AgentPOPs.POP").Order("agents.name ASC")
	if popID > 0 {
		query = query.Joins("LEFT JOIN agent_pops ON agent_pops.agent_id = agents.id").Where("agents.pop_id = ? OR agent_pops.pop_id = ?", popID, popID).Distinct("agents.*")
	}
	err := query.Find(&rows).Error
	return rows, err
}

func GetAgent(id uint) (*models.Agent, error) {
	var row models.Agent
	if err := database.DB.Preload("POP").Preload("AgentPOPs.POP").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateAgent(row *models.Agent) error { return database.DB.Create(row).Error }
func UpdateAgent(row *models.Agent) error { return database.DB.Save(row).Error }

func AgentHasPOP(agentID, popID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.AgentPOP{}).Where("agent_id = ? AND pop_id = ?", agentID, popID).Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	var agent models.Agent
	if err := database.DB.Select("pop_id").First(&agent, agentID).Error; err != nil {
		return false, err
	}
	return agent.POPID == popID, nil
}
