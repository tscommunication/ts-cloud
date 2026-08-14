package repositories

import (
	"errors"
	"fmt"
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

func SyncNetworkRouterPPPoESessions(routerID uint, rows []models.NetworkRouterPPPoESession, observedAt time.Time) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.NetworkRouterPPPoESession{}).
			Where("router_id = ? AND active = ?", routerID, true).
			Updates(map[string]any{"active": false, "disconnected_at": observedAt}).Error; err != nil {
			return err
		}
		for index := range rows {
			row := &rows[index]
			row.RouterID = routerID
			row.Active = true
			row.LastSeenAt = observedAt
			row.DisconnectedAt = nil
			var existing models.NetworkRouterPPPoESession
			err := tx.Where("router_id = ? AND session_key = ?", routerID, row.SessionKey).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				row.FirstSeenAt = observedAt
				if err := tx.Create(row).Error; err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				existing.Username, existing.Service, existing.CallerID = row.Username, row.Service, row.CallerID
				existing.Address, existing.Uptime, existing.SessionID = row.Address, row.Uptime, row.SessionID
				existing.Active, existing.LastSeenAt, existing.DisconnectedAt = true, observedAt, nil
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ListNetworkRouterPPPoESessions(routerID uint, activeOnly bool, limit int) ([]models.NetworkRouterPPPoESession, error) {
	var rows []models.NetworkRouterPPPoESession
	query := database.DB.Where("router_id = ?", routerID).Order("active DESC, last_seen_at DESC, id DESC").Limit(limit)
	if activeOnly {
		query = query.Where("active = ?", true)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list PPPoE sessions: %w", err)
	}
	return rows, nil
}

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
