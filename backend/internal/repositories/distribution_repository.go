package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

func ListPOPs() ([]models.POP, error) {
	var rows []models.POP
	err := database.DB.Order("name ASC").Find(&rows).Error
	return rows, err
}

func ListArchivedPOPs() ([]models.POP, error) {
	var rows []models.POP
	err := database.DB.
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Order("name ASC").
		Find(&rows).Error
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

func preloadAgentPackages(query *gorm.DB) *gorm.DB {
	// Some isolated unit-test schemas intentionally create only the legacy
	// distribution tables. Production receives this table through migration 37.
	if database.DB.Migrator().HasTable(&models.AgentPackage{}) {
		return query.Preload("AgentPackages.Package")
	}
	return query
}

func preloadAgentRouters(query *gorm.DB) *gorm.DB {
	if database.DB.Migrator().HasTable(&models.AgentRouter{}) {
		return query.Preload("AgentRouters.Router")
	}
	return query
}

func preloadAgentNetworkDevices(query *gorm.DB) *gorm.DB {
	if database.DB.Migrator().HasTable(&models.AgentNetworkDevice{}) {
		return query.Preload("AgentNetworkDevices.NetworkDevice")
	}
	return query
}

func ListAgents(popID uint) ([]models.Agent, error) {
	var rows []models.Agent
	query := preloadAgentNetworkDevices(preloadAgentRouters(preloadAgentPackages(database.DB.Preload("POP").Preload("AgentPOPs.POP")))).Order("agents.name ASC")
	if popID > 0 {
		query = query.Joins("LEFT JOIN agent_pops ON agent_pops.agent_id = agents.id").Where("agents.pop_id = ? OR agent_pops.pop_id = ?", popID, popID).Distinct("agents.*")
	}
	err := query.Find(&rows).Error
	return rows, err
}

func ListArchivedAgents() ([]models.Agent, error) {
	var rows []models.Agent
	err := preloadAgentNetworkDevices(preloadAgentRouters(preloadAgentPackages(database.DB.
		Unscoped().
		Preload("POP").
		Preload("AgentPOPs.POP")))).
		Where("agents.deleted_at IS NOT NULL").
		Order("agents.name ASC").
		Find(&rows).Error
	return rows, err
}

func GetAgent(id uint) (*models.Agent, error) {
	var row models.Agent
	if err := preloadAgentNetworkDevices(preloadAgentRouters(preloadAgentPackages(database.DB.Preload("POP").Preload("AgentPOPs.POP")))).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func ReplaceAgentRouters(agentID uint, routerIDs []uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentRouter{}).Error; err != nil {
			return err
		}
		seen := map[uint]bool{}
		for _, routerID := range routerIDs {
			if routerID == 0 || seen[routerID] {
				continue
			}
			seen[routerID] = true
			if err := tx.Create(&models.AgentRouter{AgentID: agentID, RouterID: routerID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func AgentHasRouter(agentID, routerID uint) (bool, error) {
	if !database.DB.Migrator().HasTable(&models.AgentRouter{}) {
		router, err := GetNetworkRouter(routerID)
		if err != nil || router.POPID == nil {
			return false, err
		}
		return AgentHasPOP(agentID, *router.POPID)
	}
	var count int64
	err := database.DB.Model(&models.AgentRouter{}).Where("agent_id = ? AND router_id = ?", agentID, routerID).Count(&count).Error
	return count > 0, err
}

func ReplaceAgentNetworkDevices(agentID uint, deviceIDs []uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).
			Delete(&models.AgentNetworkDevice{}).Error; err != nil {
			return err
		}

		seen := map[uint]bool{}
		for _, deviceID := range deviceIDs {
			if deviceID == 0 || seen[deviceID] {
				continue
			}

			seen[deviceID] = true

			var device models.NetworkDevice
			if err := tx.First(&device, deviceID).Error; err != nil {
				return err
			}

			if err := tx.Create(&models.AgentNetworkDevice{
				AgentID:         agentID,
				NetworkDeviceID: deviceID,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func AgentHasNetworkDevice(agentID, deviceID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.AgentNetworkDevice{}).
		Where(
			"agent_id = ? AND network_device_id = ?",
			agentID,
			deviceID,
		).
		Count(&count).Error

	return count > 0, err
}

func ReplaceAgentPackages(agentID uint, packageIDs []uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentPackage{}).Error; err != nil {
			return err
		}
		for _, packageID := range packageIDs {
			if err := tx.Create(&models.AgentPackage{AgentID: agentID, PackageID: packageID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func AgentHasPackage(agentID, packageID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.AgentPackage{}).
		Where("agent_id = ? AND package_id = ?", agentID, packageID).
		Count(&count).Error
	return count > 0, err
}

func ListAgentPackages(agentID uint) ([]models.Package, error) {
	var packages []models.Package
	err := database.DB.Model(&models.Package{}).
		Joins("JOIN agent_packages ON agent_packages.package_id = packages.id").
		Where("agent_packages.agent_id = ? AND packages.status = ?", agentID, "ACTIVE").
		Order("packages.name ASC").Find(&packages).Error
	return packages, err
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
