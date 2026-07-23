package services

import (
	"errors"
	"strconv"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreatePayment(payment *models.Payment) error {

	return database.DB.Transaction(func(tx *gorm.DB) error {

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

		// Save Payment
		if err := repositories.CreatePaymentTx(tx, payment); err != nil {
			return err
		}

		// Generate Receipt Number
		payment.ReceiptNo = "RCPT-" + strconv.FormatUint(uint64(payment.ID), 10)

		// Save Receipt Number
		if err := repositories.SavePaymentTx(tx, payment); err != nil {
			return err
		}
		// Update Invoice
		invoice.PaidAmount += payment.Amount
		invoice.DueAmount -= payment.Amount

		// Prevent negative due amount
		if invoice.DueAmount < 0 {
			invoice.DueAmount = 0
		}

		// Update Status
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
