package services

import (
	"errors"
	"strconv"
	"strings"

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


		switch {

		case invoice.DueAmount == 0:
			invoice.Status = "PAID"

		case invoice.PaidAmount > 0:
			invoice.Status = "PARTIAL"

		default:
			invoice.Status = "UNPAID"
		}


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
	return repositories.UpdatePayment(payment)
}


func DeletePayment(id uint) error {
	return repositories.DeletePayment(id)
}
