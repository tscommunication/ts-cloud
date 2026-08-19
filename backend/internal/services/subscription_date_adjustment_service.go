package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type SubscriptionDateAdjustmentResult struct {
	Subscription *models.Subscription
	Audit        *models.SubscriptionDateAdjustment
}

func AdjustSubscriptionDateWithoutBilling(
	subscription *models.Subscription,
	newExpiryDate time.Time,
	reason string,
	adjustedByUserID uint,
	now time.Time,
) (*SubscriptionDateAdjustmentResult, error) {
	if subscription == nil {
		return nil, fmt.Errorf("subscription is required")
	}

	if subscription.ID == 0 {
		return nil, fmt.Errorf("subscription id is required")
	}

	if adjustedByUserID == 0 {
		return nil, fmt.Errorf("adjusted by user id is required")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("adjustment reason is required")
	}

	if newExpiryDate.IsZero() {
		return nil, fmt.Errorf("new expiry date is required")
	}

	newExpiryDate = startOfDay(newExpiryDate)
	now = startOfDay(now)

	if strings.EqualFold(subscription.Status, "DISCONNECTED") {
		return nil, fmt.Errorf(
			"disconnected subscription cannot be reactivated by manual date adjustment",
		)
	}

	oldExpiryDate := subscription.ExpiryDate
	oldNextBillingDate := subscription.NextBillingDate
	oldStatus := subscription.Status

	newStatus := subscription.Status

	// Manual date adjustment is authoritative for the service expiry state:
	// today/future keeps or restores service ACTIVE, while a past expiry
	// makes the service EXPIRED. DISCONNECTED was rejected above.
	switch {
	case newExpiryDate.Before(now):
		newStatus = "EXPIRED"
	case strings.EqualFold(subscription.Status, "EXPIRED"),
		strings.EqualFold(subscription.Status, "ACTIVE"):
		newStatus = "ACTIVE"
	}

	subscription.ExpiryDate = newExpiryDate
	subscription.NextBillingDate = newExpiryDate
	subscription.Status = newStatus

	audit := &models.SubscriptionDateAdjustment{
		SubscriptionID:     subscription.ID,
		OldExpiryDate:      oldExpiryDate,
		NewExpiryDate:      newExpiryDate,
		OldNextBillingDate: oldNextBillingDate,
		NewNextBillingDate: newExpiryDate,
		OldStatus:          oldStatus,
		NewStatus:          newStatus,
		Reason:             reason,
		AdjustedByUserID:   adjustedByUserID,
		AdjustedAt:         time.Now(),
		WithoutBilling:     true,
	}

	if !now.IsZero() {
		audit.AdjustedAt = now
	}

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(subscription).Error; err != nil {
			return err
		}

		if err := tx.Create(audit).Error; err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return &SubscriptionDateAdjustmentResult{
		Subscription: subscription,
		Audit:        audit,
	}, nil
}
