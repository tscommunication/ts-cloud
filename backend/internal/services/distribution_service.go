package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

type distributionCatalogSyncResult struct {
	POPIDs        map[string]uint
	POPAgentIDs   map[string]uint
	CreatedPOPs   int
	CreatedAgents int
	UpdatedPOPs   int
	UpdatedAgents int
}

func syncApprovedDistributionCatalog(tx *gorm.DB) (*distributionCatalogSyncResult, error) {
	catalog, err := importAgentPOPCatalogs()
	if err != nil {
		return nil, err
	}
	result := &distributionCatalogSyncResult{POPIDs: map[string]uint{}, POPAgentIDs: map[string]uint{}}
	managerPrimaryPOP := map[string]uint{}
	managerPOPIDs := map[string][]uint{}
	popManagers := map[string]string{}
	for _, item := range catalog {
		popKey := normalizedCatalogName(item.POPName)
		managerKey := normalizedCatalogName(item.ManagerName)
		code := "CAT-POP-" + strings.ToUpper(item.POPID)
		var pop models.POP
		err := tx.Unscoped().Where("LOWER(name) = LOWER(?) OR code = ?", item.POPName, code).First(&pop).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pop = models.POP{Code: code, Name: item.POPName, ManagerName: item.ManagerName, Mobile: item.POPContact, Address: item.POPLocation, Status: "ACTIVE"}
			if err := tx.Create(&pop).Error; err != nil {
				return nil, fmt.Errorf("create approved POP %s: %w", item.POPName, err)
			}
			result.CreatedPOPs++
		} else if err != nil {
			return nil, err
		} else {
			pop.Name = item.POPName
			pop.ManagerName = item.ManagerName
			pop.Mobile = item.POPContact
			pop.Address = item.POPLocation
			pop.Status = "ACTIVE"
			pop.DeletedAt = gorm.DeletedAt{}
			if err := tx.Save(&pop).Error; err != nil {
				return nil, err
			}
			result.UpdatedPOPs++
		}
		result.POPIDs[popKey] = pop.ID
		popManagers[popKey] = managerKey
		if managerPrimaryPOP[managerKey] == 0 {
			managerPrimaryPOP[managerKey] = pop.ID
		}
		managerPOPIDs[managerKey] = append(managerPOPIDs[managerKey], pop.ID)
	}

	agentIDs := map[string]uint{}
	for _, item := range catalog {
		managerKey := normalizedCatalogName(item.ManagerName)
		if agentIDs[managerKey] != 0 {
			continue
		}
		code := "CAT-AGT-" + strings.ToUpper(item.ManagerID)
		var agent models.Agent
		err := tx.Unscoped().Where("LOWER(name) = LOWER(?) OR code = ?", item.ManagerName, code).First(&agent).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			agent = models.Agent{Code: code, Name: item.ManagerName, POPID: managerPrimaryPOP[managerKey], Mobile: item.ManagerContact, Address: item.ManagerAddress, OpeningBalance: item.OpeningBalance, SourceReference: "MANAGER-" + item.ManagerID + "; TYPE=" + item.ResellerType, Status: "ACTIVE"}
			if err := tx.Create(&agent).Error; err != nil {
				return nil, fmt.Errorf("create approved agent %s: %w", item.ManagerName, err)
			}
			result.CreatedAgents++
		} else if err != nil {
			return nil, err
		} else {
			agent.Name = item.ManagerName
			agent.POPID = managerPrimaryPOP[managerKey]
			agent.Mobile = item.ManagerContact
			agent.Address = item.ManagerAddress
			agent.OpeningBalance = item.OpeningBalance
			agent.SourceReference = "MANAGER-" + item.ManagerID + "; TYPE=" + item.ResellerType
			agent.Status = "ACTIVE"
			agent.DeletedAt = gorm.DeletedAt{}
			if err := tx.Save(&agent).Error; err != nil {
				return nil, err
			}
			result.UpdatedAgents++
		}
		agentIDs[managerKey] = agent.ID
		if err := replaceAgentPOPs(tx, agent.ID, managerPOPIDs[managerKey]); err != nil {
			return nil, err
		}
	}
	for popKey, managerKey := range popManagers {
		result.POPAgentIDs[popKey] = agentIDs[managerKey]
	}
	return result, nil
}

func SyncApprovedDistributionCatalog() (*distributionCatalogSyncResult, error) {
	var result *distributionCatalogSyncResult
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = syncApprovedDistributionCatalog(tx)
		return err
	})
	return result, err
}

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
	return CreateAgentWithPOPs(row, nil)
}

func normalizeAgentPOPIDs(primary uint, popIDs []uint) []uint {
	result := []uint{primary}
	seen := map[uint]bool{primary: true}
	for _, popID := range popIDs {
		if popID == 0 || seen[popID] {
			continue
		}
		seen[popID] = true
		result = append(result, popID)
	}
	return result
}

func replaceAgentPOPs(tx *gorm.DB, agentID uint, popIDs []uint) error {
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentPOP{}).Error; err != nil {
		return err
	}
	for _, popID := range popIDs {
		if err := tx.Create(&models.AgentPOP{AgentID: agentID, POPID: popID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func CreateAgentWithPOPs(row *models.Agent, popIDs []uint) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	row.Status = "ACTIVE"
	if row.Code == "" || row.Name == "" {
		return fmt.Errorf("code and name are required")
	}
	if row.CommissionPercent < 0 || row.CommissionPercent > 100 {
		return fmt.Errorf("commission percent must be between 0 and 100")
	}
	assigned := normalizeAgentPOPIDs(row.POPID, popIDs)
	return database.DB.Transaction(func(tx *gorm.DB) error {
		for _, popID := range assigned {
			var count int64
			if err := tx.Model(&models.POP{}).Where("id = ?", popID).Count(&count).Error; err != nil || count == 0 {
				return fmt.Errorf("POP %d not found", popID)
			}
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return replaceAgentPOPs(tx, row.ID, assigned)
	})
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

func UpdateAgentWithPOPs(row *models.Agent, popIDs []uint) error {
	row.Name = strings.TrimSpace(row.Name)
	if row.CommissionPercent < 0 || row.CommissionPercent > 100 {
		return fmt.Errorf("commission percent must be between 0 and 100")
	}
	assigned := normalizeAgentPOPIDs(row.POPID, popIDs)
	return database.DB.Transaction(func(tx *gorm.DB) error {
		for _, popID := range assigned {
			var count int64
			if err := tx.Model(&models.POP{}).Where("id = ?", popID).Count(&count).Error; err != nil || count == 0 {
				return fmt.Errorf("POP %d not found", popID)
			}
		}
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		return replaceAgentPOPs(tx, row.ID, assigned)
	})
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
	linked, err := repositories.AgentHasPOP(agent.ID, *popID)
	if err != nil {
		return err
	}
	if !linked {
		return fmt.Errorf("agent does not belong to the selected POP")
	}
	return nil
}
