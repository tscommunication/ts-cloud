package services

import (
	"errors"
	"fmt"
	"strconv"
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

const agentMigrationMarker = "MIGRATED TO AGENT="

func migratedAgentTarget(reference string) uint {
	position := strings.LastIndex(reference, agentMigrationMarker)
	if position < 0 {
		return 0
	}
	value := reference[position+len(agentMigrationMarker):]
	if separator := strings.Index(value, ";"); separator >= 0 {
		value = value[:separator]
	}
	id, _ := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	return uint(id)
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
		} else if agent.DeletedAt.Valid {
			targetID := migratedAgentTarget(agent.SourceReference)
			if targetID > 0 {
				var targetCount int64
				if err := tx.Model(&models.Agent{}).Where("id = ?", targetID).Count(&targetCount).Error; err != nil {
					return nil, err
				}
				if targetCount > 0 {
					agentIDs[managerKey] = targetID
				}
			}
			continue
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

type AgentMigrationResult struct {
	SourceAgentID uint  `json:"source_agent_id"`
	TargetAgentID uint  `json:"target_agent_id"`
	Customers     int64 `json:"customers_migrated"`
	Users         int64 `json:"login_users_migrated"`
	POPs          int   `json:"pops_migrated"`
}

func MigrateAgent(sourceID, targetID uint) (*AgentMigrationResult, error) {
	if sourceID == 0 || targetID == 0 || sourceID == targetID {
		return nil, fmt.Errorf("source and target agents must be different")
	}
	result := &AgentMigrationResult{SourceAgentID: sourceID, TargetAgentID: targetID}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var source, target models.Agent
		if err := tx.Preload("AgentPOPs").First(&source, sourceID).Error; err != nil {
			return fmt.Errorf("source agent not found")
		}
		if err := tx.Preload("AgentPOPs").First(&target, targetID).Error; err != nil {
			return fmt.Errorf("target agent not found")
		}
		if target.Status != "ACTIVE" {
			return fmt.Errorf("target agent must be ACTIVE")
		}
		customerUpdate := tx.Model(&models.Customer{}).Where("agent_id = ?", sourceID).Update("agent_id", targetID)
		if customerUpdate.Error != nil {
			return customerUpdate.Error
		}
		result.Customers = customerUpdate.RowsAffected
		userUpdate := tx.Model(&models.User{}).Where("agent_id = ?", sourceID).Update("agent_id", targetID)
		if userUpdate.Error != nil {
			return userUpdate.Error
		}
		result.Users = userUpdate.RowsAffected
		popIDs := normalizeAgentPOPIDs(target.POPID, nil)
		seen := map[uint]bool{target.POPID: true}
		for _, membership := range target.AgentPOPs {
			if !seen[membership.POPID] {
				seen[membership.POPID] = true
				popIDs = append(popIDs, membership.POPID)
			}
		}
		for _, membership := range source.AgentPOPs {
			if !seen[membership.POPID] {
				seen[membership.POPID] = true
				popIDs = append(popIDs, membership.POPID)
				result.POPs++
			}
		}
		if !seen[source.POPID] {
			popIDs = append(popIDs, source.POPID)
			result.POPs++
		}
		if err := replaceAgentPOPs(tx, targetID, popIDs); err != nil {
			return err
		}
		if err := tx.Where("agent_id = ?", sourceID).Delete(&models.AgentPOP{}).Error; err != nil {
			return err
		}
		source.Status = "INACTIVE"
		source.SourceReference = strings.TrimSpace(source.SourceReference + "; " + agentMigrationMarker + strconv.FormatUint(uint64(targetID), 10))
		if err := tx.Save(&source).Error; err != nil {
			return err
		}
		return tx.Delete(&source).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func DeleteAgent(id uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var source models.Agent
		if err := tx.First(&source, id).Error; err != nil {
			return fmt.Errorf("agent not found")
		}
		var dependencies int64
		for _, model := range []interface{}{&models.Customer{}, &models.User{}, &models.AgentCollection{}, &models.AgentSettlement{}, &models.Payment{}} {
			var count int64
			column := "agent_id"
			if _, ok := model.(*models.Payment); ok {
				column = "collected_by_agent_id"
			}
			if err := tx.Model(model).Where(column+" = ?", id).Count(&count).Error; err != nil {
				return err
			}
			dependencies += count
		}
		if dependencies > 0 {
			return fmt.Errorf("agent has linked customers, users or financial history; migrate it instead")
		}
		if err := tx.Where("agent_id = ?", id).Delete(&models.AgentPOP{}).Error; err != nil {
			return err
		}
		source.Status = "INACTIVE"
		source.SourceReference = strings.TrimSpace(source.SourceReference + "; DELETED BY USER")
		if err := tx.Save(&source).Error; err != nil {
			return err
		}
		return tx.Delete(&source).Error
	})
}
