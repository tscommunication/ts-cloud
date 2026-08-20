package services

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateInvoice(invoice *models.Invoice) error {
	if err := validateInvoice(invoice); err != nil {
		return err
	}
	subscription, err := repositories.GetSubscriptionByID(invoice.SubscriptionID)
	if err != nil {
		return errors.New("subscription not found")
	}
	if subscription.Status == "DISCONNECTED" {
		return errors.New("cannot create an invoice for a disconnected subscription")
	}
	exists, err := repositories.InvoiceExistsForPeriod(invoice.SubscriptionID, invoice.BillMonth, invoice.BillYear, 0)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("an invoice already exists for this subscription and billing period")
	}
	invoice.CustomerID = subscription.CustomerID
	invoice.PackageID = subscription.PackageID
	invoice.PackagePrice = money(subscription.Package.Price)
	invoice.TotalAmount = money(invoice.PackagePrice - invoice.Discount + invoice.Vat)
	invoice.DueAmount = invoice.TotalAmount
	invoice.PaidAmount = 0
	invoice.Status = "UNPAID"
	if invoice.TotalAmount <= 0 {
		return errors.New("invoice total must be greater than zero")
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}
		invoice.InvoiceNo = fmt.Sprintf("INV-%06d", invoice.ID)
		return tx.Save(invoice).Error
	})
}

func GetInvoices() ([]models.Invoice, error) {
	return repositories.GetInvoices()
}

func GetInvoicesByAgent(agentID uint) ([]models.Invoice, error) {
	return repositories.GetInvoicesByAgent(agentID)
}

func InvoiceBelongsToAgent(invoiceID, agentID uint) (bool, error) {
	return repositories.InvoiceBelongsToAgent(invoiceID, agentID)
}

func GetInvoiceByID(id uint) (*models.Invoice, error) {
	return repositories.GetInvoiceByID(id)
}

func UpdateInvoice(invoice *models.Invoice) error {
	original, err := repositories.GetInvoiceByID(invoice.ID)
	if err != nil {
		return err
	}
	if original.Status == "CANCELLED" {
		return errors.New("cancelled invoices cannot be edited")
	}
	if original.PaidAmount > 0 && invoiceFinancialFieldsChanged(original, invoice) {
		return errors.New("financial fields cannot be changed after payment")
	}
	if err := validateInvoice(invoice); err != nil {
		return err
	}
	subscription, err := repositories.GetSubscriptionByID(invoice.SubscriptionID)
	if err != nil {
		return errors.New("subscription not found")
	}
	if subscription.Status == "DISCONNECTED" {
		return errors.New("cannot move an invoice to a disconnected subscription")
	}
	exists, err := repositories.InvoiceExistsForPeriod(invoice.SubscriptionID, invoice.BillMonth, invoice.BillYear, invoice.ID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("an invoice already exists for this subscription and billing period")
	}
	invoice.TotalAmount = money(invoice.PackagePrice - invoice.Discount + invoice.Vat)
	invoice.DueAmount = money(invoice.TotalAmount - invoice.PaidAmount)
	if invoice.TotalAmount <= 0 || invoice.DueAmount < 0 {
		return errors.New("invoice amount is invalid")
	}
	invoice.CustomerID = subscription.CustomerID
	invoice.PackageID = subscription.PackageID
	setInvoicePaymentStatus(invoice, time.Now())
	return repositories.UpdateInvoice(invoice)
}

func CancelInvoice(invoice *models.Invoice) error {
	if invoice.Status == "CANCELLED" {
		return errors.New("invoice is already cancelled")
	}
	count, err := repositories.CountInvoicePayments(invoice.ID)
	if err != nil {
		return err
	}
	if count > 0 || invoice.PaidAmount > 0 {
		return errors.New("an invoice with payment history cannot be cancelled")
	}
	invoice.Status = "CANCELLED"
	invoice.DueAmount = 0
	return repositories.UpdateInvoice(invoice)
}

func ProcessInvoiceOverdues(now time.Time) (int64, error) {
	return repositories.MarkOverdueInvoices(now)
}

func validateInvoice(invoice *models.Invoice) error {
	if invoice.BillMonth < 1 || invoice.BillMonth > 12 {
		return errors.New("bill month must be between 1 and 12")
	}
	if invoice.BillYear < 2000 || invoice.BillYear > 2100 {
		return errors.New("bill year must be between 2000 and 2100")
	}
	if invoice.DueDate.Before(invoice.IssueDate) {
		return errors.New("due date cannot be before issue date")
	}
	if invalidMoney(invoice.PackagePrice) || invalidMoney(invoice.Discount) || invalidMoney(invoice.Vat) {
		return errors.New("invoice amounts must be valid non-negative values")
	}
	return nil
}

func invalidMoney(value float64) bool {
	return value < 0 || math.IsNaN(value) || math.IsInf(value, 0)
}

func money(value float64) float64 {
	return math.Round(value*100) / 100
}

func invoiceFinancialFieldsChanged(original, updated *models.Invoice) bool {
	return updated.SubscriptionID != original.SubscriptionID ||
		updated.BillMonth != original.BillMonth ||
		updated.BillYear != original.BillYear ||
		!updated.IssueDate.Equal(original.IssueDate) ||
		!updated.DueDate.Equal(original.DueDate) ||
		money(updated.PackagePrice) != money(original.PackagePrice) ||
		money(updated.Discount) != money(original.Discount) ||
		money(updated.Vat) != money(original.Vat)
}

func GetInvoicesByCustomer(customerID uint) ([]models.Invoice, error) {
	return repositories.GetInvoicesByCustomer(customerID)
}
