package repositories

import (
	"errors"
	"fmt"
	"strings"
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

func ListNetworkRoutersForAgent(agentID uint) ([]models.NetworkRouter, error) {
	var rows []models.NetworkRouter
	err := database.DB.Model(&models.NetworkRouter{}).Preload("POP").
		Joins("JOIN agent_routers ON agent_routers.router_id = network_routers.id").
		Where("agent_routers.agent_id = ?", agentID).Order("network_routers.name, network_routers.id").Find(&rows).Error
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
			Updates(map[string]any{"active": false, "disconnected_at": observedAt, "disconnect_reason": "NOT_OBSERVED_ON_SYNC"}).Error; err != nil {
			return err
		}
		for index := range rows {
			row := &rows[index]
			row.RouterID = routerID
			row.Active = true
			row.LastSeenAt = observedAt
			row.DisconnectedAt, row.DisconnectReason = nil, ""
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
				// PPP active statistics provide byte counters. Derive live bit/s
				// from the counter delta because RouterOS does not reliably return
				// rx-rate/tx-rate for this menu through the API.
				elapsed := observedAt.Sub(existing.LastSeenAt).Seconds()
				if elapsed > 0 {
					if row.RxBytes >= existing.RxBytes {
						row.RxRateBps = int64(float64(row.RxBytes-existing.RxBytes) * 8 / elapsed)
					} else {
						row.RxRateBps = 0
					}
					if row.TxBytes >= existing.TxBytes {
						row.TxRateBps = int64(float64(row.TxBytes-existing.TxBytes) * 8 / elapsed)
					} else {
						row.TxRateBps = 0
					}
				}
				rxDelta, txDelta := int64(0), int64(0)
				if row.RxBytes >= existing.RxBytes {
					rxDelta = row.RxBytes - existing.RxBytes
				}
				if row.TxBytes >= existing.TxBytes {
					txDelta = row.TxBytes - existing.TxBytes
				}
				if rxDelta > 0 || txDelta > 0 {
					usageDate := time.Date(observedAt.Year(), observedAt.Month(), observedAt.Day(), 0, 0, 0, 0, observedAt.Location())
					var usage models.NetworkRouterPPPoEDailyUsage
					usageErr := tx.Where("router_id = ? AND session_key = ? AND usage_date = ?", routerID, row.SessionKey, usageDate).First(&usage).Error
					if errors.Is(usageErr, gorm.ErrRecordNotFound) {
						usage = models.NetworkRouterPPPoEDailyUsage{RouterID: routerID, SessionKey: row.SessionKey, UsageDate: usageDate, Username: row.Username, RxBytes: rxDelta, TxBytes: txDelta}
						if err := tx.Create(&usage).Error; err != nil {
							return err
						}
					} else if usageErr != nil {
						return usageErr
					} else if err := tx.Model(&usage).Updates(map[string]any{"username": row.Username, "rx_bytes": usage.RxBytes + rxDelta, "tx_bytes": usage.TxBytes + txDelta}).Error; err != nil {
						return err
					}
				}
				existing.Username, existing.Service, existing.CallerID = row.Username, row.Service, row.CallerID
				existing.Address, existing.Uptime, existing.SessionID = row.Address, row.Uptime, row.SessionID
				existing.RxRateBps, existing.TxRateBps = row.RxRateBps, row.TxRateBps
				existing.RxBytes, existing.TxBytes = row.RxBytes, row.TxBytes
				existing.Active, existing.LastSeenAt, existing.DisconnectedAt, existing.DisconnectReason = true, observedAt, nil, ""
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func SyncNetworkRouterPPPSecrets(routerID uint, rows []models.NetworkRouterPPPSecret, observedAt time.Time) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.NetworkRouterPPPSecret{}).Where("router_id = ? AND present = ?", routerID, true).Update("present", false).Error; err != nil {
			return err
		}
		for index := range rows {
			row := &rows[index]
			row.RouterID, row.Present, row.LastSeenAt = routerID, true, observedAt
			var existing models.NetworkRouterPPPSecret
			err := tx.Where("router_id = ? AND router_os_id = ?", routerID, row.RouterOSID).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				row.FirstSeenAt = observedAt
				if err := tx.Create(row).Error; err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				existing.Username, existing.Service, existing.Profile = row.Username, row.Service, row.Profile
				existing.CallerID, existing.RemoteAddress = row.CallerID, row.RemoteAddress
				existing.Disabled, existing.Present, existing.LastSeenAt = row.Disabled, true, observedAt
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ListNetworkRouterPPPSecrets(presentOnly bool, limit int) ([]models.NetworkRouterPPPSecretView, error) {
	var rows []models.NetworkRouterPPPSecretView
	query := database.DB.Table("network_router_ppp_secrets AS secret").
		Select(`secret.id, secret.router_id, router.code AS router_code, router.name AS router_name,
			secret.username, secret.service, secret.profile, secret.caller_id, secret.remote_address,
			secret.disabled, secret.present, secret.last_seen_at, subscription.id AS subscription_id,
			subscription.subscription_code, subscription.status AS subscription_status,
			customer.id AS customer_id, customer.customer_code, customer.full_name AS customer_name`).
		Joins("JOIN network_routers AS router ON router.id = secret.router_id").
		Joins("LEFT JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(secret.username) AND (subscription.router_id = secret.router_id OR subscription.router_id = 0)").
		Joins("LEFT JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Order("secret.present DESC, secret.username, secret.id").Limit(limit)
	if presentOnly {
		query = query.Where("secret.present = ?", true)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list PPP secrets: %w", err)
	}
	return rows, nil
}

func GetNetworkRouterPPPSecret(id uint) (*models.NetworkRouterPPPSecret, error) {
	var row models.NetworkRouterPPPSecret
	if err := database.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func ListNetworkRouterPPPoESessions(routerID uint, activeOnly bool, limit int) ([]models.NetworkRouterPPPoESessionView, error) {
	var rows []models.NetworkRouterPPPoESessionView
	query := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`session.id, session.router_id, router.code AS router_code, router.name AS router_name,
			session.username, session.service, session.caller_id,
			session.address, session.uptime, session.session_id, session.rx_rate_bps, session.tx_rate_bps,
			session.rx_bytes, session.tx_bytes, session.active, session.first_seen_at,
			session.last_seen_at, session.disconnected_at, subscription.id AS subscription_id,
			session.disconnect_reason,
			subscription.subscription_code, subscription.status AS subscription_status,
			customer.id AS customer_id, customer.customer_code, customer.full_name AS customer_name,
			package.id AS package_id, package.package_code, package.name AS package_name`).
		Joins("JOIN network_routers AS router ON router.id = session.router_id").
		Joins("LEFT JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
		Joins("LEFT JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Joins("LEFT JOIN agents AS agent ON agent.id = customer.agent_id AND agent.deleted_at IS NULL").
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
	return listNetworkPPPoESessions(0, activeOnly, limit)
}

func ListNetworkPPPoESessionsForAgent(agentID uint, activeOnly bool, limit int) ([]models.NetworkRouterPPPoESessionView, error) {
	if agentID == 0 {
		return []models.NetworkRouterPPPoESessionView{}, nil
	}
	return listNetworkPPPoESessions(agentID, activeOnly, limit)
}

func listNetworkPPPoESessions(agentID uint, activeOnly bool, limit int) ([]models.NetworkRouterPPPoESessionView, error) {
	var rows []models.NetworkRouterPPPoESessionView
	query := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`session.id, session.router_id, router.code AS router_code, router.name AS router_name,
			session.username, session.service, session.caller_id, session.address, session.uptime,
			session.session_id, session.rx_rate_bps, session.tx_rate_bps, session.rx_bytes, session.tx_bytes,
			session.active, session.first_seen_at, session.last_seen_at,
			session.disconnected_at, session.disconnect_reason, subscription.id AS subscription_id, subscription.subscription_code,
			subscription.status AS subscription_status, customer.id AS customer_id, customer.customer_code,
			customer.full_name AS customer_name, customer.agent_id, agent.code AS agent_code, agent.name AS agent_name,
			package.id AS package_id, package.package_code,
			package.name AS package_name`).
		Joins("JOIN network_routers AS router ON router.id = session.router_id").
		Joins("LEFT JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
		Joins("LEFT JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Joins("LEFT JOIN agents AS agent ON agent.id = customer.agent_id AND agent.deleted_at IS NULL").
		Joins("LEFT JOIN packages AS package ON package.id = subscription.package_id AND package.deleted_at IS NULL").
		Order("session.active DESC, session.last_seen_at DESC, session.id DESC").Limit(limit)
	if agentID > 0 {
		query = query.Where("customer.agent_id = ?", agentID)
	}
	if activeOnly {
		query = query.Where("session.active = ?", true)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list all PPPoE sessions: %w", err)
	}
	return rows, nil
}

func NetworkPPPoESessionBelongsToAgent(sessionID, agentID uint) (bool, error) {
	if sessionID == 0 || agentID == 0 {
		return false, nil
	}
	var count int64
	err := database.DB.Table("network_router_pppoe_sessions AS session").
		Joins("JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
		Joins("JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
		Where("session.id = ? AND customer.agent_id = ?", sessionID, agentID).
		Count(&count).Error
	return count > 0, err
}

func GetNetworkRouterPPPoESession(id uint) (*models.NetworkRouterPPPoESession, error) {
	var row models.NetworkRouterPPPoESession
	if err := database.DB.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetNetworkRouterPPPoESessionForIdentity(routerID uint, username string) (*models.NetworkRouterPPPoESession, error) {
	var row models.NetworkRouterPPPoESession
	err := database.DB.Where("router_id = ? AND LOWER(username) = LOWER(?) AND active = ?", routerID, strings.TrimSpace(username), true).First(&row).Error
	return &row, err
}

func PPPoEUsernameMappedToAnotherSubscription(username string, subscriptionID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.Subscription{}).
		Where("LOWER(pp_po_e_username) = LOWER(?) AND id <> ?", username, subscriptionID).
		Count(&count).Error
	return count > 0, err
}

func MapPPPoESessionToSubscription(session *models.NetworkRouterPPPoESession, subscription *models.Subscription) error {
	return MapPPPIdentityToSubscription(session.RouterID, session.Username, subscription)
}

func MapPPPIdentityToSubscription(routerID uint, username string, subscription *models.Subscription) error {
	if subscription == nil || subscription.ID == 0 {
		return fmt.Errorf("subscription is required")
	}
	username = strings.TrimSpace(username)
	if routerID == 0 || username == "" {
		return fmt.Errorf("router and PPPoE username are required")
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var account models.CustomerInternetAccount
		accountID := subscription.InternetAccountID
		if accountID != nil {
			if err := tx.First(&account, *accountID).Error; err != nil {
				return err
			}
		} else {
			err := tx.Where("customer_id = ?", subscription.CustomerID).First(&account).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				account = models.CustomerInternetAccount{AccountCode: fmt.Sprintf("NET-MAP-%06d", subscription.ID), CustomerID: subscription.CustomerID, RouterID: routerID, PackageID: subscription.PackageID, ActivationDate: &subscription.ActivationDate, BillingDay: subscription.BillingDay, NextBillingDate: &subscription.NextBillingDate, ExpiryDate: &subscription.ExpiryDate, PPPoEUsername: username, Status: subscription.Status, SyncIntervalMinutes: 30}
				if err := tx.Create(&account).Error; err != nil {
					return fmt.Errorf("create credential-less internet account: %w", err)
				}
				accountID = &account.ID
			} else if err != nil {
				return err
			} else {
				var linked int64
				if err := tx.Model(&models.Subscription{}).Where("internet_account_id = ? AND id <> ?", account.ID, subscription.ID).Count(&linked).Error; err != nil {
					return err
				}
				if linked > 0 {
					return fmt.Errorf("customer internet account is already linked to another subscription")
				}
				accountID = &account.ID
			}
		}
		var conflicting int64
		if err := tx.Model(&models.CustomerInternetAccount{}).Where("LOWER(pp_po_e_username) = LOWER(?) AND id <> ?", username, account.ID).Count(&conflicting).Error; err != nil {
			return err
		}
		if conflicting > 0 {
			return fmt.Errorf("PPPoE username is already owned by another internet account")
		}
		if err := tx.Model(&models.CustomerInternetAccount{}).Where("id = ?", account.ID).Updates(map[string]any{"router_id": routerID, "pp_po_e_username": username, "package_id": subscription.PackageID, "activation_date": subscription.ActivationDate, "billing_day": subscription.BillingDay, "next_billing_date": subscription.NextBillingDate, "expiry_date": subscription.ExpiryDate, "status": subscription.Status}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Subscription{}).Where("id = ?", subscription.ID).Updates(map[string]any{"router_id": routerID, "pp_po_e_username": username, "internet_account_id": accountID}).Error
	})
}

type NetworkPPPoESummary struct {
	ActiveSessions   int64 `json:"active_sessions"`
	MappedSessions   int64 `json:"mapped_sessions"`
	UnmappedSessions int64 `json:"unmapped_sessions"`
}

type NetworkPPPoEDailyUsageSummary struct {
	Days    int   `json:"days"`
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

type NetworkPPPoEUserUsage struct {
	RouterID   uint   `json:"router_id"`
	RouterCode string `json:"router_code"`
	Username   string `json:"username"`
	RxBytes    int64  `json:"rx_bytes"`
	TxBytes    int64  `json:"tx_bytes"`
}

func GetNetworkPPPoEDailyUsageSummary(days int, now time.Time) (*NetworkPPPoEDailyUsageSummary, error) {
	return getNetworkPPPoEDailyUsageSummary(0, days, now)
}

func GetNetworkPPPoEDailyUsageSummaryForAgent(agentID uint, days int, now time.Time) (*NetworkPPPoEDailyUsageSummary, error) {
	return getNetworkPPPoEDailyUsageSummary(agentID, days, now)
}

func getNetworkPPPoEDailyUsageSummary(agentID uint, days int, now time.Time) (*NetworkPPPoEDailyUsageSummary, error) {
	if days < 1 {
		days = 1
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	result := &NetworkPPPoEDailyUsageSummary{Days: days}
	query := database.DB.Table("network_router_pppoe_daily_usage AS usage").Where("usage.usage_date >= ?", start)
	if agentID > 0 {
		query = query.Joins("JOIN network_router_pppoe_sessions AS session ON session.router_id = usage.router_id AND session.session_key = usage.session_key").
			Joins("JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
			Joins("JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
			Where("customer.agent_id = ?", agentID)
	}
	if err := query.Select("COALESCE(SUM(usage.rx_bytes), 0) AS rx_bytes, COALESCE(SUM(usage.tx_bytes), 0) AS tx_bytes").Scan(result).Error; err != nil {
		return nil, fmt.Errorf("load PPPoE daily usage: %w", err)
	}
	return result, nil
}

func ListNetworkPPPoEUserUsage(days, limit int, now time.Time) ([]NetworkPPPoEUserUsage, error) {
	return listNetworkPPPoEUserUsage(0, days, limit, now)
}

func ListNetworkPPPoEUserUsageForAgent(agentID uint, days, limit int, now time.Time) ([]NetworkPPPoEUserUsage, error) {
	return listNetworkPPPoEUserUsage(agentID, days, limit, now)
}

func listNetworkPPPoEUserUsage(agentID uint, days, limit int, now time.Time) ([]NetworkPPPoEUserUsage, error) {
	if days < 1 {
		days = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	var rows []NetworkPPPoEUserUsage
	query := database.DB.Table("network_router_pppoe_daily_usage AS usage").Select("usage.router_id, router.code AS router_code, usage.username, SUM(usage.rx_bytes) AS rx_bytes, SUM(usage.tx_bytes) AS tx_bytes").Joins("JOIN network_routers AS router ON router.id = usage.router_id").Where("usage.usage_date >= ?", start)
	if agentID > 0 {
		query = query.Joins("JOIN network_router_pppoe_sessions AS session ON session.router_id = usage.router_id AND session.session_key = usage.session_key").
			Joins("JOIN subscriptions AS subscription ON LOWER(subscription.pp_po_e_username) = LOWER(session.username) AND (subscription.router_id = session.router_id OR subscription.router_id = 0)").
			Joins("JOIN customers AS customer ON customer.id = subscription.customer_id AND customer.deleted_at IS NULL").
			Where("customer.agent_id = ?", agentID)
	}
	err := query.Group("usage.router_id, router.code, usage.username").Order("(SUM(usage.rx_bytes) + SUM(usage.tx_bytes)) DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func GetNetworkPPPoESummary() (*NetworkPPPoESummary, error) {
	return getNetworkPPPoESummary(0)
}

func GetNetworkPPPoESummaryForAgent(agentID uint) (*NetworkPPPoESummary, error) {
	return getNetworkPPPoESummary(agentID)
}

func getNetworkPPPoESummary(agentID uint) (*NetworkPPPoESummary, error) {
	var summary NetworkPPPoESummary
	query := database.DB.Table("network_router_pppoe_sessions AS session").
		Select(`COUNT(*) AS active_sessions,
			COALESCE(SUM(CASE WHEN EXISTS (
				SELECT 1 FROM subscriptions AS subscription
				WHERE LOWER(subscription.pp_po_e_username) = LOWER(session.username)
				AND (subscription.router_id = session.router_id OR subscription.router_id = 0)
			) THEN 1 ELSE 0 END), 0) AS mapped_sessions`).
		Where("session.active = ?", true)
	if agentID > 0 {
		query = query.Joins("JOIN subscriptions AS subscription_scope ON LOWER(subscription_scope.pp_po_e_username) = LOWER(session.username) AND (subscription_scope.router_id = session.router_id OR subscription_scope.router_id = 0)").
			Joins("JOIN customers AS customer_scope ON customer_scope.id = subscription_scope.customer_id AND customer_scope.deleted_at IS NULL").
			Where("customer_scope.agent_id = ?", agentID)
	}
	err := query.Scan(&summary).Error
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
