package services

import (
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type AgentDashboardSummary struct {
	TotalCustomers      int64   `json:"total_customers"`
	ActiveCustomers     int64   `json:"active_customers"`
	ActiveSubscriptions int64   `json:"active_subscriptions"`
	OnlineCustomers     int64   `json:"online_customers"`
	OfflineCustomers    int64   `json:"offline_customers"`
	ExpiredCustomers    int64   `json:"expired_customers"`
	TotalInvoiced       float64 `json:"total_invoiced"`
	TotalOutstanding    float64 `json:"total_outstanding"`
	TotalCollected      float64 `json:"total_collected"`
	TodayCollected      float64 `json:"today_collected"`
	CommissionEarned    float64 `json:"commission_earned"`
	CommissionPaid      float64 `json:"commission_paid"`
	CommissionPayable   float64 `json:"commission_payable"`
	OverdueInvoices     int64   `json:"overdue_invoices"`
	VoidedCollections   int64   `json:"voided_collections"`
	VoidedAmount        float64 `json:"voided_amount"`
}

func GetAgentDashboardSummary(agentID uint, now time.Time) (*AgentDashboardSummary, error) {
	return getAgentDashboardSummary(database.DB, agentID, now)
}

func getAgentDashboardSummary(db *gorm.DB, agentID uint, now time.Time) (*AgentDashboardSummary, error) {
	summary := &AgentDashboardSummary{}
	customers := db.Model(&models.Customer{}).Where("agent_id = ?", agentID)
	if err := customers.Count(&summary.TotalCustomers).Error; err != nil {
		return nil, err
	}
	if err := customers.Where("status = ?", "ACTIVE").Count(&summary.ActiveCustomers).Error; err != nil {
		return nil, err
	}

	subscriptions := db.Model(&models.Subscription{}).
		Joins("JOIN customers ON customers.id = subscriptions.customer_id").
		Where("customers.agent_id = ? AND subscriptions.status = ?", agentID, "ACTIVE")
	if err := subscriptions.Count(&summary.ActiveSubscriptions).Error; err != nil {
		return nil, err
	}
	accounts := db.Table("customer_internet_accounts AS account").
		Joins("JOIN customers ON customers.id = account.customer_id AND customers.deleted_at IS NULL").
		Where("customers.agent_id = ?", agentID)
	if err := accounts.Where(`EXISTS (SELECT 1 FROM network_router_pppoe_sessions AS session WHERE session.router_id = account.router_id AND LOWER(session.username) = LOWER(account.pp_po_e_username) AND session.active = ?)`, true).Count(&summary.OnlineCustomers).Error; err != nil {
		return nil, err
	}
	if err := accounts.Where(`NOT EXISTS (SELECT 1 FROM network_router_pppoe_sessions AS session WHERE session.router_id = account.router_id AND LOWER(session.username) = LOWER(account.pp_po_e_username) AND session.active = ?)`, true).Count(&summary.OfflineCustomers).Error; err != nil {
		return nil, err
	}
	if err := accounts.Where("account.status = ? OR (account.expiry_date IS NOT NULL AND account.expiry_date < ?)", "EXPIRED", now).Count(&summary.ExpiredCustomers).Error; err != nil {
		return nil, err
	}

	invoices := func() *gorm.DB {
		return db.Model(&models.Invoice{}).
			Joins("JOIN customers ON customers.id = invoices.customer_id").
			Where("customers.agent_id = ? AND invoices.status <> ?", agentID, "CANCELLED")
	}
	if err := invoices().Select("COALESCE(SUM(invoices.total_amount), 0)").Scan(&summary.TotalInvoiced).Error; err != nil {
		return nil, err
	}
	if err := invoices().Select("COALESCE(SUM(invoices.due_amount), 0)").Scan(&summary.TotalOutstanding).Error; err != nil {
		return nil, err
	}
	if err := invoices().Where("invoices.status = ?", "OVERDUE").Count(&summary.OverdueInvoices).Error; err != nil {
		return nil, err
	}

	collections := db.Model(&models.AgentCollection{}).Where("agent_id = ? AND status = ?", agentID, "ACTIVE")
	if err := collections.Select("COALESCE(SUM(amount), 0)").Scan(&summary.TotalCollected).Error; err != nil {
		return nil, err
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := collections.Where("collected_at >= ?", dayStart).Select("COALESCE(SUM(amount), 0)").Scan(&summary.TodayCollected).Error; err != nil {
		return nil, err
	}
	voided := db.Model(&models.AgentCollection{}).Where("agent_id = ? AND status = ?", agentID, "VOID")
	if err := voided.Count(&summary.VoidedCollections).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.AgentCollection{}).Where("agent_id = ? AND status = ?", agentID, "VOID").Select("COALESCE(SUM(amount), 0)").Scan(&summary.VoidedAmount).Error; err != nil {
		return nil, err
	}

	balance, err := GetAgentSettlementBalance(db, agentID)
	if err != nil {
		return nil, err
	}
	summary.CommissionEarned = balance.Earned
	summary.CommissionPaid = balance.Paid
	summary.CommissionPayable = balance.Payable
	return summary, nil
}
