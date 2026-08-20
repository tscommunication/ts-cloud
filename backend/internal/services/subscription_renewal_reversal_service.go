package services

import (
	"errors"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"gorm.io/gorm"
)

type SubscriptionRenewalReversalEligibility struct {
	PaymentID      uint
	RenewalID      uint
	SubscriptionID uint

	RenewalFound bool
	Eligible     bool
	Reason       string

	CurrentExpiryDate       time.Time
	RestoredExpiryDate      time.Time
	CurrentNextBillingDate  time.Time
	RestoredNextBillingDate time.Time
}

// EvaluateSubscriptionRenewalReversalTx determines whether the renewal
// created by paymentID can be automatically reversed without overwriting
// newer subscription lifecycle history.
//
// This function is intentionally read-only. It does not mutate Payment,
// Invoice, Subscription, renewal ledger, reversal ledger, or MikroTik.
func EvaluateSubscriptionRenewalReversalTx(
	tx *gorm.DB,
	paymentID uint,
) (SubscriptionRenewalReversalEligibility, error) {
	result := SubscriptionRenewalReversalEligibility{
		PaymentID: paymentID,
	}

	if tx == nil {
		return result, errors.New("database transaction is required")
	}

	if paymentID == 0 {
		return result, errors.New("payment id is required")
	}

	renewal, err :=
		repositories.GetSubscriptionRenewalByPaymentIDTx(
			tx,
			paymentID,
		)
	if err != nil {
		if repositories.IsSubscriptionRenewalNotFound(err) {
			result.Reason =
				"payment has no subscription renewal"
			return result, nil
		}

		return result, err
	}

	result.RenewalFound = true
	result.RenewalID = renewal.ID
	result.SubscriptionID = renewal.SubscriptionID
	result.RestoredExpiryDate = renewal.OldExpiryDate
	result.RestoredNextBillingDate =
		renewal.OldNextBillingDate

	alreadyReversed, err :=
		repositories.
			SubscriptionRenewalReversalExistsByPaymentIDTx(
				tx,
				paymentID,
			)
	if err != nil {
		return result, err
	}

	if alreadyReversed {
		result.Reason =
			"subscription renewal is already reversed"
		return result, nil
	}

	latest, err :=
		repositories.
			GetLatestSubscriptionRenewalBySubscriptionIDTx(
				tx,
				renewal.SubscriptionID,
			)
	if err != nil {
		return result, err
	}

	if latest.ID != renewal.ID {
		result.Reason =
			"a newer subscription renewal exists"
		return result, nil
	}

	// A Super Admin manual date adjustment performed after this renewal
	// makes automatic rollback unsafe even if the dates later happen to
	// equal the original renewal values again.
	var laterDateAdjustments int64

	if err := tx.Model(
		&models.SubscriptionDateAdjustment{},
	).
		Where(
			"subscription_id = ? AND created_at > ?",
			renewal.SubscriptionID,
			renewal.CreatedAt,
		).
		Count(&laterDateAdjustments).Error; err != nil {
		return result, err
	}

	if laterDateAdjustments > 0 {
		result.Reason =
			"subscription date was manually adjusted after renewal"
		return result, nil
	}

	var subscription models.Subscription

	if err := tx.First(
		&subscription,
		renewal.SubscriptionID,
	).Error; err != nil {
		return result, err
	}

	result.CurrentExpiryDate =
		subscription.ExpiryDate
	result.CurrentNextBillingDate =
		subscription.NextBillingDate

	if !subscription.ExpiryDate.Equal(
		renewal.NewExpiryDate,
	) ||
		!subscription.NextBillingDate.Equal(
			renewal.NewNextBillingDate,
		) {
		result.Reason =
			"current subscription dates no longer match renewal state"
		return result, nil
	}

	// A billed renewal establishes ACTIVE status. If lifecycle state has
	// subsequently changed (for example SUSPENDED or DISCONNECTED), an
	// automatic payment-void reversal must not overwrite that newer state.
	if !strings.EqualFold(
		strings.TrimSpace(subscription.Status),
		"ACTIVE",
	) {
		result.Reason =
			"current subscription status no longer matches renewal state"
		return result, nil
	}

	result.Eligible = true
	result.Reason =
		"renewal is safe to reverse"

	return result, nil
}

type SubscriptionRenewalReversalResult struct {
	PaymentID      uint
	RenewalID      uint
	SubscriptionID uint

	Reversed bool
	Reason   string

	Reversal *models.SubscriptionRenewalReversal
}

func FindSubscriptionRenewalReversalByPaymentID(
	paymentID uint,
) (*models.SubscriptionRenewalReversal, bool, error) {
	if paymentID == 0 {
		return nil, false, errors.New("payment id is required")
	}

	reversal, err :=
		repositories.GetSubscriptionRenewalReversalByPaymentID(
			paymentID,
		)
	if err != nil {
		if repositories.IsSubscriptionRenewalReversalNotFound(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return reversal, true, nil
}

func ReverseSubscriptionRenewalForPaymentTx(
	tx *gorm.DB,
	paymentID uint,
	reason string,
	reversedByUserID uint,
	now time.Time,
) (SubscriptionRenewalReversalResult, error) {
	result := SubscriptionRenewalReversalResult{
		PaymentID: paymentID,
	}

	if tx == nil {
		return result, errors.New("database transaction is required")
	}

	if paymentID == 0 {
		return result, errors.New("payment id is required")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return result, errors.New("reversal reason is required")
	}

	if reversedByUserID == 0 {
		return result, errors.New("reversal actor is required")
	}

	if now.IsZero() {
		now = time.Now()
	}

	eligibility, err :=
		EvaluateSubscriptionRenewalReversalTx(
			tx,
			paymentID,
		)
	if err != nil {
		return result, err
	}

	result.RenewalID = eligibility.RenewalID
	result.SubscriptionID = eligibility.SubscriptionID

	if !eligibility.RenewalFound {
		result.Reason = eligibility.Reason
		return result, nil
	}

	if !eligibility.Eligible {
		result.Reason = eligibility.Reason
		return result, nil
	}

	renewal, err :=
		repositories.GetSubscriptionRenewalByPaymentIDTx(
			tx,
			paymentID,
		)
	if err != nil {
		return result, err
	}

	var subscription models.Subscription
	if err := tx.First(
		&subscription,
		renewal.SubscriptionID,
	).Error; err != nil {
		return result, err
	}

	previousExpiryDate :=
		subscription.ExpiryDate
	previousNextBillingDate :=
		subscription.NextBillingDate

	subscription.ExpiryDate =
		renewal.OldExpiryDate
	subscription.NextBillingDate =
		renewal.OldNextBillingDate

	if subscription.ExpiryDate.Before(
		startOfDay(now),
	) {
		subscription.Status = "EXPIRED"
	} else {
		subscription.Status = "ACTIVE"
	}

	if err := tx.Save(&subscription).Error; err != nil {
		return result, err
	}

	reversal := &models.SubscriptionRenewalReversal{
		RenewalID:      renewal.ID,
		PaymentID:      renewal.PaymentID,
		InvoiceID:      renewal.InvoiceID,
		CustomerID:     renewal.CustomerID,
		SubscriptionID: renewal.SubscriptionID,

		PreviousExpiryDate: previousExpiryDate,
		RestoredExpiryDate: renewal.OldExpiryDate,

		PreviousNextBillingDate: previousNextBillingDate,
		RestoredNextBillingDate: renewal.OldNextBillingDate,

		Reason:           reason,
		ReversedByUserID: reversedByUserID,
		ReversedAt:       now,
	}

	if err := repositories.
		CreateSubscriptionRenewalReversalTx(
			tx,
			reversal,
		); err != nil {
		return result, err
	}

	result.Reversed = true
	result.Reason = "subscription renewal reversed"
	result.Reversal = reversal

	return result, nil
}
