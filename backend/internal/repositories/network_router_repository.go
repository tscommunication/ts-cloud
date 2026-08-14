package repositories

import (
	"errors"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
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

func ListMonitoredNetworkRouters() ([]models.NetworkRouter, error) {
	var rows []models.NetworkRouter
	err := database.DB.Where("status = ? AND api_password_encrypted <> ''", "ACTIVE").Order("id").Find(&rows).Error
	return rows, err
}

func CreateNetworkRouter(row *models.NetworkRouter) error { return database.DB.Create(row).Error }
func UpdateNetworkRouter(row *models.NetworkRouter) error { return database.DB.Save(row).Error }

func LatestNetworkRouterHealth(routerID uint) (*models.NetworkRouterHealth, error) {
	var row models.NetworkRouterHealth
	err := database.DB.Where("router_id = ?", routerID).Order("observed_at DESC, id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func CreateNetworkRouterHealth(row *models.NetworkRouterHealth) error {
	return database.DB.Create(row).Error
}

func ListNetworkRouterHealth(routerID uint, limit int) ([]models.NetworkRouterHealth, error) {
	var rows []models.NetworkRouterHealth
	err := database.DB.Where("router_id = ?", routerID).Order("observed_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func DeleteNetworkRouterHealthBefore(cutoff time.Time) error {
	return database.DB.Where("observed_at < ?", cutoff).Delete(&models.NetworkRouterHealth{}).Error
}

func ActiveNetworkRouterAlert(routerID uint, alertType string) (*models.NetworkRouterAlert, error) {
	var row models.NetworkRouterAlert
	err := database.DB.Where("router_id = ? AND type = ? AND status = ?", routerID, alertType, "ACTIVE").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func SaveNetworkRouterAlert(row *models.NetworkRouterAlert) error {
	if row.ID == 0 {
		return database.DB.Create(row).Error
	}
	return database.DB.Save(row).Error
}

func ListNetworkRouterAlerts(status string, limit int) ([]models.NetworkRouterAlert, error) {
	var rows []models.NetworkRouterAlert
	query := database.DB.Preload("Router").Order("last_observed_at DESC, id DESC").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&rows).Error
	return rows, err
}
