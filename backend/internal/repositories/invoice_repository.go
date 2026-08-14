package repositories

import (
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateInvoice(invoice *models.Invoice) error {
	return database.DB.Create(invoice).Error
}

func GetInvoices() ([]models.Invoice, error) {
	var invoices []models.Invoice

	err := database.DB.
		Preload("Customer").
		Preload("Package").
		Preload("Subscription").
		Order("issue_date DESC, id DESC").
		Find(&invoices).Error

	return invoices, err
}

func GetInvoicesByAgent(agentID uint) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := database.DB.
		Joins("JOIN customers ON customers.id = invoices.customer_id").
		Where("customers.agent_id = ?", agentID).
		Preload("Customer").Preload("Package").Preload("Subscription").
		Order("invoices.issue_date DESC, invoices.id DESC").
		Find(&invoices).Error
	return invoices, err
}

func InvoiceBelongsToAgent(invoiceID, agentID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.Invoice{}).
		Joins("JOIN customers ON customers.id = invoices.customer_id").
		Where("invoices.id = ? AND customers.agent_id = ?", invoiceID, agentID).
		Count(&count).Error
	return count > 0, err
}

func GetInvoiceByID(id uint) (*models.Invoice, error) {
	var invoice models.Invoice

	err := database.DB.
		Preload("Customer").
		Preload("Package").
		Preload("Subscription").
		First(&invoice, id).Error

	if err != nil {
		return nil, err
	}

	return &invoice, nil
}

func GetInvoicesByCustomer(customerID uint) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := database.DB.Where("customer_id = ?", customerID).Order("issue_date DESC").Find(&invoices).Error
	return invoices, err
}

func UpdateInvoice(invoice *models.Invoice) error {
	return database.DB.Save(invoice).Error
}

func InvoiceExistsForPeriod(subscriptionID uint, billMonth, billYear int, excludeID uint) (bool, error) {
	query := database.DB.Model(&models.Invoice{}).
		Where("subscription_id = ? AND bill_month = ? AND bill_year = ? AND status <> ?", subscriptionID, billMonth, billYear, "CANCELLED")
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func CountInvoicePayments(invoiceID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Payment{}).Where("invoice_id = ?", invoiceID).Count(&count).Error
	return count, err
}

func MarkOverdueInvoices(now time.Time) (int64, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := database.DB.Model(&models.Invoice{}).
		Where("status IN ? AND due_amount > 0 AND due_date < ?", []string{"UNPAID", "PARTIAL"}, start).
		Update("status", "OVERDUE")
	return result.RowsAffected, result.Error
}

// SaveInvoiceTx saves an invoice using an existing database transaction.
func SaveInvoiceTx(tx *gorm.DB, invoice *models.Invoice) error {
	return tx.Save(invoice).Error
}
