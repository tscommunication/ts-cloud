package repositories

import (
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
		Find(&invoices).Error

	return invoices, err
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

func UpdateInvoice(invoice *models.Invoice) error {
	return database.DB.Save(invoice).Error
}

func DeleteInvoice(id uint) error {
	return database.DB.Delete(&models.Invoice{}, id).Error
}

// SaveInvoiceTx saves an invoice using an existing database transaction.
func SaveInvoiceTx(tx *gorm.DB, invoice *models.Invoice) error {
	return tx.Save(invoice).Error
}
