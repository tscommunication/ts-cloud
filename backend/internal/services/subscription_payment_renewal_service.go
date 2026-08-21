package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type PaymentRenewalResult struct {
	Renewed bool
	Reason  string
	Renewal *models.SubscriptionRenewal
}

// RenewSubscriptionForPaidInvoiceTx performs a billed renewal only when the
// payment has caused the invoice to become fully PAID.
//
// Important contracts:
//   - partial payments never extend service;
//   - one payment may create at most one renewal;
//   - payment, invoice, subscription date mutation and renewal ledger are
//     committed atomically by the caller transaction;
//   - DISCONNECTED subscriptions are never automatically renewed.
func RenewSubscriptionForPaidInvoiceTx(
	tx *gorm.DB,
	payment *models.Payment,
	invoice *models.Invoice,
	now time.Time,
) (PaymentRenewalResult, error) {
	if tx == nil {
		return PaymentRenewalResult{},
			errors.New("database transaction is required")
	}

	if payment == nil {
		return PaymentRenewalResult{},
			errors.New("payment is required")
	}

	if payment.ID == 0 {
		return PaymentRenewalResult{},
			errors.New("payment id is required")
	}

	if invoice == nil {
		return PaymentRenewalResult{},
			errors.New("invoice is required")
	}

	if invoice.ID == 0 {
		return PaymentRenewalResult{},
			errors.New("invoice id is required")
	}

	if payment.InvoiceID != invoice.ID {
		return PaymentRenewalResult{},
			errors.New("payment invoice does not match invoice")
	}

	if payment.Status != "SUCCESS" {
		return PaymentRenewalResult{
			Reason: "payment is not successful",
		}, nil
	}

	if invoice.Status != "PAID" || invoice.DueAmount > 0 {
		return PaymentRenewalResult{
			Reason: "invoice is not fully paid",
		}, nil
	}

	exists, err :=
		repositories.SubscriptionRenewalExistsByPaymentIDTx(
			tx,
			payment.ID,
		)
	if err != nil {
		return PaymentRenewalResult{}, err
	}

	if exists {
		return PaymentRenewalResult{
			Reason: "payment renewal already recorded",
		}, nil
	}

	var subscription models.Subscription

	if err := tx.First(
		&subscription,
		invoice.SubscriptionID,
	).Error; err != nil {
		return PaymentRenewalResult{}, err
	}

	if subscription.ID != payment.SubscriptionID {
		return PaymentRenewalResult{},
			errors.New("payment subscription does not match invoice subscription")
	}

	if strings.EqualFold(
		subscription.Status,
		"DISCONNECTED",
	) {
		return PaymentRenewalResult{
			Reason: "disconnected subscription cannot be automatically renewed",
		}, nil
	}

	if now.IsZero() {
		now = time.Now()
	}

	oldExpiryDate := subscription.ExpiryDate
	oldNextBillingDate := subscription.NextBillingDate

	baseDate := subscription.ExpiryDate
	today := startOfDay(now)

	// Day-to-day billing contract:
	// if service already expired, renewal starts from the payment/recharge day.
	if baseDate.Before(today) {
		baseDate = today
	}

	standardExpiryDate := baseDate.AddDate(0, 1, 0)
	standardDuration := standardExpiryDate.Sub(baseDate)
	temporaryDeduction, temporaryAccessItems, err :=
		PendingTemporaryAccessDeductionTx(tx, subscription.ID)
	if err != nil {
		return PaymentRenewalResult{}, err
	}
	if temporaryDeduction >= standardDuration {
		return PaymentRenewalResult{
			Reason: "temporary access deduction requires superadmin review",
		}, nil
	}
	newExpiryDate := standardExpiryDate.Add(-temporaryDeduction)

	subscription.ExpiryDate = newExpiryDate
	subscription.NextBillingDate = newExpiryDate
	subscription.Status = "ACTIVE"
	subscription.LastPaymentDate = &payment.PaymentDate
	subscription.LastPaidAmount = payment.Amount
	subscription.DueAmount = invoice.DueAmount

	if err := tx.Save(&subscription).Error; err != nil {
		return PaymentRenewalResult{}, err
	}
	if subscription.InternetAccountID != nil && *subscription.InternetAccountID != 0 {
		if err := tx.Model(&models.CustomerInternetAccount{}).
			Where("id = ?", *subscription.InternetAccountID).
			Updates(map[string]interface{}{
				"next_billing_date": newExpiryDate,
				"expiry_date":       newExpiryDate,
				"status":            "ACTIVE",
			}).Error; err != nil {
			return PaymentRenewalResult{}, err
		}
	}

	renewal := &models.SubscriptionRenewal{
		PaymentID:                      payment.ID,
		InvoiceID:                      invoice.ID,
		CustomerID:                     invoice.CustomerID,
		SubscriptionID:                 subscription.ID,
		OldExpiryDate:                  oldExpiryDate,
		NewExpiryDate:                  newExpiryDate,
		OldNextBillingDate:             oldNextBillingDate,
		NewNextBillingDate:             newExpiryDate,
		RenewalDate:                    now,
		Amount:                         payment.Amount,
		StandardDurationSeconds:        int64(standardDuration / time.Second),
		TemporaryAccessDeductedSeconds: int64(temporaryDeduction / time.Second),
		NetDurationSeconds:             int64((standardDuration - temporaryDeduction) / time.Second),
		Source:                         "PAYMENT",
	}

	if err := repositories.CreateSubscriptionRenewalTx(
		tx,
		renewal,
	); err != nil {
		return PaymentRenewalResult{}, err
	}
	for index := range temporaryAccessItems {
		item := &temporaryAccessItems[index]
		if err := tx.Model(&models.TemporaryInternetAccess{}).
			Where("id = ? AND status = ?", item.ID, item.Status).
			Updates(map[string]interface{}{
				"pre_settlement_status": item.Status,
				"status":                TemporaryInternetAccessSettled,
				"settlement_payment_id": payment.ID,
				"settled_at":            now,
			}).Error; err != nil {
			return PaymentRenewalResult{}, err
		}
	}

	return PaymentRenewalResult{
		Renewed: true,
		Reason:  "invoice fully paid",
		Renewal: renewal,
	}, nil
}
