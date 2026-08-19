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
		if err := syncAgentCollection(tx, payment, nil); err != nil {
			return err
		}

		return nil
	})
}

func GetPayments() ([]models.Payment, error) {
	return repositories.GetPayments()
}

func GetPaymentsByAgent(agentID uint) ([]models.Payment, error) {
	return repositories.GetPaymentsByAgent(agentID)
}

func PaymentBelongsToAgent(paymentID, agentID uint) (bool, error) {
	return repositories.PaymentBelongsToAgent(paymentID, agentID)
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
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}
		return syncAgentCollection(tx, payment, &existing)
	})
}

func VoidPayment(
	id uint,
	reason string,
	voidedByUserID uint,
	now time.Time,
) (*models.PaymentVoidAudit, error) {
	reason = strings.TrimSpace(reason)

	if id == 0 {
		return nil, errors.New("payment id is required")
	}

	if reason == "" {
		return nil, errors.New("void reason is required")
	}

	if voidedByUserID == 0 {
		return nil, errors.New("void actor is required")
	}

	if now.IsZero() {
		now = time.Now()
	}

	var audit *models.PaymentVoidAudit

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var payment models.Payment

		if err := tx.First(&payment, id).Error; err != nil {
			return err
		}

		if payment.Status != "SUCCESS" {
			return errors.New("payment is already voided")
		}

		var invoice models.Invoice

		if err := tx.First(
			&invoice,
			payment.InvoiceID,
		).Error; err != nil {
			return err
		}

		previousStatus := payment.Status

		payment.Status = "VOID"

		if err := tx.Save(&payment).Error; err != nil {
			return err
		}

		var paidAmount float64

		if err := tx.Model(&models.Payment{}).
			Where(
				"invoice_id = ? AND status = ?",
				payment.InvoiceID,
				"SUCCESS",
			).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&paidAmount).Error; err != nil {
			return err
		}

		reconcileInvoice(
			&invoice,
			paidAmount,
		)

		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.AgentCollection{}).
			Where("payment_id = ?", payment.ID).
			Update("status", "VOID").Error; err != nil {
			return err
		}

		audit = &models.PaymentVoidAudit{
			PaymentID:      payment.ID,
			InvoiceID:      payment.InvoiceID,
			CustomerID:     payment.CustomerID,
			SubscriptionID: payment.SubscriptionID,
			ReceiptNo:      payment.ReceiptNo,
			Amount:         payment.Amount,
			PreviousStatus: previousStatus,
			NewStatus:      payment.Status,
			Reason:         reason,
			VoidedByUserID: voidedByUserID,
			VoidedAt:       now,
		}

		if err := repositories.CreatePaymentVoidAuditTx(
			tx,
			audit,
		); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return audit, nil
}

func syncAgentCollection(tx *gorm.DB, payment *models.Payment, _ *models.Payment) error {
	var collection models.AgentCollection
	err := tx.Where("payment_id = ?", payment.ID).First(&collection).Error
	if err == nil {
		collection.Amount = payment.Amount
		collection.CommissionAmount = math.Round(payment.Amount*collection.CommissionRate) / 100
		collection.CollectedAt = payment.PaymentDate
		collection.Status = "ACTIVE"
		return tx.Save(&collection).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var customer models.Customer
	if err := tx.First(&customer, payment.CustomerID).Error; err != nil {
		return err
	}
	if customer.AgentID == nil {
		return nil
	}
	var agent models.Agent
	if err := tx.First(&agent, *customer.AgentID).Error; err != nil {
		return err
	}
	rate := agent.CommissionPercent
	collection = models.AgentCollection{AgentID: agent.ID, CustomerID: customer.ID, PaymentID: payment.ID, Amount: payment.Amount, CommissionRate: rate, CommissionAmount: math.Round(payment.Amount*rate) / 100, Status: "ACTIVE", CollectedAt: payment.PaymentDate}
	return tx.Create(&collection).Error
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
