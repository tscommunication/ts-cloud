package repositories

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type BillingSummary struct {
	TotalInvoiced     float64 `json:"total_invoiced"`
	TotalCollected    float64 `json:"total_collected"`
	TotalOutstanding  float64 `json:"total_outstanding"`
	TodayCollected    float64 `json:"today_collected"`
	OverdueInvoices   int64   `json:"overdue_invoices"`
	UnpaidInvoices    int64   `json:"unpaid_invoices"`
	CancelledInvoices int64   `json:"cancelled_invoices"`
	VoidedPayments    int64   `json:"voided_payments"`
	VoidedAmount      float64 `json:"voided_amount"`
}

func GetSubscriptionsDueForBilling(now time.Time) ([]models.Subscription, error) {
	end := beginningOfDay(now).AddDate(0, 0, 1)
	var subscriptions []models.Subscription
	err := database.DB.Preload("Customer").Preload("Package").
		Where("status IN ? AND next_billing_date < ?", []string{"ACTIVE", "SUSPENDED"}, end).
		Order("next_billing_date ASC").Find(&subscriptions).Error
	return subscriptions, err
}

func CreateBillingRun(run *models.BillingRun) error          { return database.DB.Create(run).Error }
func SaveBillingRun(run *models.BillingRun) error            { return database.DB.Save(run).Error }
func CreateBillingRunItem(item *models.BillingRunItem) error { return database.DB.Create(item).Error }

func GetRecentBillingRuns(limit int) ([]models.BillingRun, error) {
	var runs []models.BillingRun
	err := database.DB.Order("id DESC").Limit(limit).Find(&runs).Error
	return runs, err
}

func GetBillingSummary(now time.Time) (*BillingSummary, error) {
	summary := &BillingSummary{}
	if err := database.DB.Model(&models.Invoice{}).Where("status <> ?", "CANCELLED").Select("COALESCE(SUM(total_amount), 0)").Scan(&summary.TotalInvoiced).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).Where("status <> ?", "CANCELLED").Select("COALESCE(SUM(due_amount), 0)").Scan(&summary.TotalOutstanding).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).Where("status = ?", "SUCCESS").Select("COALESCE(SUM(amount), 0)").Scan(&summary.TotalCollected).Error; err != nil {
		return nil, err
	}
	start := beginningOfDay(now)
	if err := database.DB.Model(&models.Payment{}).Where("status = ? AND payment_date >= ? AND payment_date < ?", "SUCCESS", start, start.AddDate(0, 0, 1)).Select("COALESCE(SUM(amount), 0)").Scan(&summary.TodayCollected).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).Where("status = ?", "OVERDUE").Count(&summary.OverdueInvoices).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).Where("status IN ?", []string{"UNPAID", "PARTIAL", "OVERDUE"}).Count(&summary.UnpaidInvoices).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).Where("status = ?", "CANCELLED").Count(&summary.CancelledInvoices).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).Where("status = ?", "VOID").Count(&summary.VoidedPayments).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).Where("status = ?", "VOID").Select("COALESCE(SUM(amount), 0)").Scan(&summary.VoidedAmount).Error; err != nil {
		return nil, err
	}
	return summary, nil
}
