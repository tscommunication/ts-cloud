package services

import (
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func ListPOPs() ([]models.POP, error)     { return repositories.ListPOPs() }
func GetPOP(id uint) (*models.POP, error) { return repositories.GetPOP(id) }

func CreatePOP(row *models.POP) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	row.Status = "ACTIVE"
	if row.Code == "" || row.Name == "" {
		return fmt.Errorf("code and name are required")
	}
	return repositories.CreatePOP(row)
}

func UpdatePOP(row *models.POP) error {
	row.Name = strings.TrimSpace(row.Name)
	return repositories.UpdatePOP(row)
}
func ListAgents(popID uint) ([]models.Agent, error) { return repositories.ListAgents(popID) }
func GetAgent(id uint) (*models.Agent, error)       { return repositories.GetAgent(id) }

func CreateAgent(row *models.Agent) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	row.Status = "ACTIVE"
	if row.Code == "" || row.Name == "" {
		return fmt.Errorf("code and name are required")
	}
	if row.CommissionPercent < 0 || row.CommissionPercent > 100 {
		return fmt.Errorf("commission percent must be between 0 and 100")
	}
	if _, err := repositories.GetPOP(row.POPID); err != nil {
		return fmt.Errorf("POP not found")
	}
	return repositories.CreateAgent(row)
}

func UpdateAgent(row *models.Agent) error {
	row.Name = strings.TrimSpace(row.Name)
	if row.CommissionPercent < 0 || row.CommissionPercent > 100 {
		return fmt.Errorf("commission percent must be between 0 and 100")
	}
	if _, err := repositories.GetPOP(row.POPID); err != nil {
		return fmt.Errorf("POP not found")
	}
	return repositories.UpdateAgent(row)
}

func ValidateCustomerDistribution(popID, agentID *uint) error {
	if popID != nil {
		if _, err := repositories.GetPOP(*popID); err != nil {
			return fmt.Errorf("POP not found")
		}
	}
	if agentID == nil {
		return nil
	}
	agent, err := repositories.GetAgent(*agentID)
	if err != nil {
		return fmt.Errorf("agent not found")
	}
	if popID == nil {
		return fmt.Errorf("POP is required when an agent is assigned")
	}
	if agent.POPID != *popID {
		return fmt.Errorf("agent does not belong to the selected POP")
	}
	return nil
}
