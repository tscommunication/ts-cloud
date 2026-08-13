package services

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreatePayment(payment *models.Payment) error {

	return database.DB.Transaction(func(tx *gorm.DB) error {

		// Payment Method Validation
		method := strings.ToUpper(payment.Method)

		switch method {

		case "CASH":
			// Transaction ID optional

		case "BKASH", "NAGAD", "ROCKET", "BANK":

			if strings.TrimSpace(payment.TransactionID) == "" {
				return errors.New("transaction id required for " + method)
			}

		default:
			return errors.New("invalid payment method")
		}

		payment.Method = method

		// Duplicate Transaction Check
		if payment.TransactionID != "" {

			var count int64

			err := tx.Model(&models.Payment{}).
				Where("transaction_id = ?", payment.TransactionID).
				Count(&count).Error

			if err != nil {
				return err
			}

			if count > 0 {
				return errors.New("transaction already exists")
			}
		}

		// Load Invoice
		invoice, err := repositories.GetInvoiceByID(payment.InvoiceID)

		if err != nil {
			return err
		}

		// Validation

		if payment.Amount <= 0 {
			return errors.New("payment amount must be greater than zero")
		}

		if invoice.Status == "CANCELLED" {
			return errors.New("cancelled invoices cannot receive payments")
		}
		if invoice.DueAmount <= 0 {
			return errors.New("invoice already paid")
		}

		if payment.Amount > invoice.DueAmount {
			return errors.New("payment amount exceeds due amount")
		}

		// Auto Copy Customer & Subscription

		payment.CustomerID = invoice.CustomerID
		payment.SubscriptionID = invoice.SubscriptionID

		// Save Payment

		if err := repositories.CreatePaymentTx(tx, payment); err != nil {
			return err
		}

		// Generate Receipt Number

		payment.ReceiptNo =
			"RCPT-" + strconv.FormatUint(uint64(payment.ID), 10)

		// Save Receipt Number

		if err := repositories.SavePaymentTx(tx, payment); err != nil {
			return err
		}

		// Update Invoice

		invoice.PaidAmount += payment.Amount

		invoice.DueAmount -= payment.Amount

		if invoice.DueAmount < 0 {
			invoice.DueAmount = 0
		}

		setInvoicePaymentStatus(invoice, time.Now())

		// Save Invoice

		if err := repositories.SaveInvoiceTx(tx, invoice); err != nil {
			return err
		}

		return nil
	})
}

func GetPayments() ([]models.Payment, error) {
	return repositories.GetPayments()
}

func GetPaymentByID(id uint) (*models.Payment, error) {
	return repositories.GetPaymentByID(id)
}

func UpdatePayment(payment *models.Payment) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.Payment
		if err := tx.First(&existing, payment.ID).Error; err != nil {
			return err
		}
		if existing.Status != "SUCCESS" {
			return errors.New("voided payments cannot be edited")
		}

		var invoice models.Invoice
		if err := tx.First(&invoice, existing.InvoiceID).Error; err != nil {
			return err
		}

		method, err := validatePaymentMethod(payment.Method, payment.TransactionID)
		if err != nil {
			return err
		}
		if payment.Amount <= 0 {
			return errors.New("payment amount must be greater than zero")
		}
		if payment.TransactionID != "" {
			var count int64
			if err := tx.Model(&models.Payment{}).
				Where("transaction_id = ? AND id <> ?", payment.TransactionID, payment.ID).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("transaction already exists")
			}
		}

		var otherPaymentsTotal float64
		if err := tx.Model(&models.Payment{}).
			Where("invoice_id = ? AND id <> ? AND status = ?", existing.InvoiceID, payment.ID, "SUCCESS").
			Select("COALESCE(SUM(amount), 0)").
			Scan(&otherPaymentsTotal).Error; err != nil {
			return err
		}
		if otherPaymentsTotal+payment.Amount > invoice.TotalAmount {
			return errors.New("payment amount exceeds invoice total")
		}

		payment.InvoiceID = existing.InvoiceID
		payment.CustomerID = existing.CustomerID
		payment.SubscriptionID = existing.SubscriptionID
		payment.ReceiptNo = existing.ReceiptNo
		payment.Status = existing.Status
		payment.Method = method
		if err := tx.Save(payment).Error; err != nil {
			return err
		}

		reconcileInvoice(&invoice, otherPaymentsTotal+payment.Amount)
		return tx.Save(&invoice).Error
	})
}

func VoidPayment(id uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var payment models.Payment
		if err := tx.First(&payment, id).Error; err != nil {
			return err
		}
		if payment.Status != "SUCCESS" {
			return errors.New("payment is already voided")
		}

		var invoice models.Invoice
		if err := tx.First(&invoice, payment.InvoiceID).Error; err != nil {
			return err
		}

		payment.Status = "VOID"
		if err := tx.Save(&payment).Error; err != nil {
			return err
		}

		var paidAmount float64
		if err := tx.Model(&models.Payment{}).
			Where("invoice_id = ? AND status = ?", payment.InvoiceID, "SUCCESS").
			Select("COALESCE(SUM(amount), 0)").
			Scan(&paidAmount).Error; err != nil {
			return err
		}
		reconcileInvoice(&invoice, paidAmount)
		return tx.Save(&invoice).Error
	})
}

func validatePaymentMethod(method, transactionID string) (string, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "CASH":
		return method, nil
	case "BKASH", "NAGAD", "ROCKET", "BANK":
		if strings.TrimSpace(transactionID) == "" {
			return "", errors.New("transaction id required for " + method)
		}
		return method, nil
	default:
		return "", errors.New("invalid payment method")
	}
}

func reconcileInvoice(invoice *models.Invoice, paidAmount float64) {
	invoice.PaidAmount = paidAmount
	invoice.DueAmount = math.Max(invoice.TotalAmount-paidAmount, 0)
	setInvoicePaymentStatus(invoice, time.Now())
}

func setInvoicePaymentStatus(invoice *models.Invoice, now time.Time) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case invoice.DueAmount == 0:
		invoice.Status = "PAID"
	case !invoice.DueDate.IsZero() && invoice.DueDate.Before(start):
		invoice.Status = "OVERDUE"
	case invoice.PaidAmount > 0:
		invoice.Status = "PARTIAL"
	default:
		invoice.Status = "UNPAID"
	}
}
