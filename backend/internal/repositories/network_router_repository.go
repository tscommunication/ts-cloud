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

func ListNetworkRouterPPPoESessions(routerID uint, activeOnly bool, limit int) ([]models.NetworkRouterPPPoESessionView, error) {
	var rows []models.NetworkRouterPPPoESessionView
	query := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`session.id, session.router_id, router.code AS router_code, router.name AS router_name,
			session.username, session.service, session.caller_id,
			session.address, session.uptime, session.session_id, session.active, session.first_seen_at,
			session.last_seen_at, session.disconnected_at, subscription.id AS subscription_id,
			subscription.subscription_code, subscription.status AS subscription_status,
			customer.id AS customer_id, customer.customer_code, customer.full_name AS customer_name,
			package.id AS package_id, package.package_code, package.name AS package_name`).
		Joins("JOIN network_routers AS router ON router.id = session.router_id").
		Joins("LEFT JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
		Joins("LEFT JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Joins("LEFT JOIN packages AS package ON package.id = subscription.package_id AND package.deleted_at IS NULL").
		Where("session.router_id = ?", routerID).
		Order("session.active DESC, session.last_seen_at DESC, session.id DESC").Limit(limit)
	if activeOnly {
		query = query.Where("session.active = ?", true)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list PPPoE sessions: %w", err)
	}
	return rows, nil
}

func ListNetworkPPPoESessions(activeOnly bool, limit int) ([]models.NetworkRouterPPPoESessionView, error) {
	var rows []models.NetworkRouterPPPoESessionView
	query := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`session.id, session.router_id, router.code AS router_code, router.name AS router_name,
			session.username, session.service, session.caller_id, session.address, session.uptime,
			session.session_id, session.active, session.first_seen_at, session.last_seen_at,
			session.disconnected_at, subscription.id AS subscription_id, subscription.subscription_code,
			subscription.status AS subscription_status, customer.id AS customer_id, customer.customer_code,
			customer.full_name AS customer_name, package.id AS package_id, package.package_code,
			package.name AS package_name`).
		Joins("JOIN network_routers AS router ON router.id = session.router_id").
		Joins("LEFT JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
		Joins("LEFT JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Joins("LEFT JOIN packages AS package ON package.id = subscription.package_id AND package.deleted_at IS NULL").
		Order("session.active DESC, session.last_seen_at DESC, session.id DESC").Limit(limit)
	if activeOnly {
		query = query.Where("session.active = ?", true)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list all PPPoE sessions: %w", err)
	}
	return rows, nil
}

func GetNetworkRouterPPPoESession(id uint) (*models.NetworkRouterPPPoESession, error) {
	var row models.NetworkRouterPPPoESession
	if err := database.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func PPPoEUsernameMappedToAnotherSubscription(username string, subscriptionID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.Subscription{}).
		Where("LOWER(pp_po_e_username) = LOWER(?) AND id <> ?", username, subscriptionID).
		Count(&count).Error
	return count > 0, err
}

func MapPPPoESessionToSubscription(session *models.NetworkRouterPPPoESession, subscription *models.Subscription) error {
	return database.DB.Model(subscription).Updates(map[string]any{
		"router_id":        session.RouterID,
		"pp_po_e_username": session.Username,
	}).Error
}

type NetworkPPPoESummary struct {
	ActiveSessions   int64 `json:"active_sessions"`
	MappedSessions   int64 `json:"mapped_sessions"`
	UnmappedSessions int64 `json:"unmapped_sessions"`
}

func GetNetworkPPPoESummary() (*NetworkPPPoESummary, error) {
	var summary NetworkPPPoESummary
	err := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`COUNT(*) AS active_sessions,
			COALESCE(SUM(CASE WHEN EXISTS (
				SELECT 1 FROM subscriptions AS subscription
				WHERE LOWER(subscription.pp_po_e_username) = LOWER(session.username)
				AND (subscription.router_id = session.router_id OR subscription.router_id = 0)
			) THEN 1 ELSE 0 END), 0) AS mapped_sessions`).
		Where("session.active = ?", true).
		Scan(&summary).Error
	if err != nil {
		return nil, fmt.Errorf("load PPPoE summary: %w", err)
	}
	summary.UnmappedSessions = summary.ActiveSessions - summary.MappedSessions
	return &summary, nil
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

func ListActiveNetworkRoutersByPOPIDs(popIDs []uint) ([]models.NetworkRouter, error) {
	var rows []models.NetworkRouter

	if len(popIDs) == 0 {
		return rows, nil
	}

	err := database.DB.
		Preload("POP").
		Where("status = ?", "ACTIVE").
		Where("pop_id IN ?", popIDs).
		Order("name ASC, id ASC").
		Find(&rows).Error

	return rows, err
}

func ListActiveNetworkRouters() ([]models.NetworkRouter, error) {
	var rows []models.NetworkRouter

	err := database.DB.
		Preload("POP").
		Where("status = ?", "ACTIVE").
		Order("name ASC, id ASC").
		Find(&rows).Error

	return rows, err
}
