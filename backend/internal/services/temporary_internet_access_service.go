package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

const (
	TemporaryInternetAccessActive    = "ACTIVE"
	TemporaryInternetAccessExpired   = "EXPIRED"
	TemporaryInternetAccessCancelled = "CANCELLED"
	TemporaryInternetAccessSettled   = "SETTLED"
	TemporaryInternetStatusActive    = "TEMPORARY_ACTIVE"
)

type TemporaryInternetAccessGrantInput struct {
	CustomerID        uint
	Days              int
	PromisedPaymentAt *time.Time
	PromisedAmount    float64
	RequestSource     string
	Reason            string
	GrantedByUserID   uint
	Now               time.Time
}

func ListTemporaryInternetAccess(customerID uint) ([]models.TemporaryInternetAccess, error) {
	var items []models.TemporaryInternetAccess
	err := database.DB.Where("customer_id = ?", customerID).Order("id DESC").Find(&items).Error
	return items, err
}

func GrantTemporaryInternetAccess(input TemporaryInternetAccessGrantInput) (*models.TemporaryInternetAccess, error) {
	if input.CustomerID == 0 || input.GrantedByUserID == 0 {
		return nil, fmt.Errorf("customer and granting user are required")
	}
	if input.Days < 1 || input.Days > 7 {
		return nil, fmt.Errorf("temporary access must be between 1 and 7 days")
	}
	input.RequestSource = strings.ToUpper(strings.TrimSpace(input.RequestSource))
	if input.RequestSource != "CUSTOMER" && input.RequestSource != "RESELLER" {
		return nil, fmt.Errorf("request source must be CUSTOMER or RESELLER")
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		return nil, fmt.Errorf("temporary access reason is required")
	}
	if input.PromisedAmount < 0 {
		return nil, fmt.Errorf("promised amount cannot be negative")
	}
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	if input.PromisedPaymentAt != nil && input.PromisedPaymentAt.Before(input.Now) {
		return nil, fmt.Errorf("promised payment time cannot be in the past")
	}

	var granted models.TemporaryInternetAccess
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var account models.CustomerInternetAccount
		if err := tx.Where("customer_id = ?", input.CustomerID).First(&account).Error; err != nil {
			return fmt.Errorf("customer internet account not found")
		}
		var activeCount int64
		if err := tx.Model(&models.TemporaryInternetAccess{}).
			Where("internet_account_id = ? AND status = ?", account.ID, TemporaryInternetAccessActive).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return fmt.Errorf("customer already has active temporary access")
		}
		var subscription models.Subscription
		if err := tx.Where("internet_account_id = ? AND status <> ?", account.ID, "DISCONNECTED").
			Order("id ASC").First(&subscription).Error; err != nil {
			return fmt.Errorf("linked internet subscription not found")
		}

		endsAt := input.Now.Add(time.Duration(input.Days) * 24 * time.Hour)
		granted = models.TemporaryInternetAccess{
			CustomerID: input.CustomerID, InternetAccountID: account.ID,
			SubscriptionID: subscription.ID, Status: TemporaryInternetAccessActive,
			StartsAt: input.Now, EndsAt: endsAt,
			GrantedDurationSeconds: int64(input.Days) * 24 * 60 * 60,
			PromisedPaymentAt:      input.PromisedPaymentAt, PromisedAmount: input.PromisedAmount,
			RequestSource: input.RequestSource, Reason: input.Reason,
			GrantedByUserID: input.GrantedByUserID, GrantedAt: input.Now,
		}
		if err := tx.Create(&granted).Error; err != nil {
			return err
		}
		return tx.Model(&models.CustomerInternetAccount{}).
			Where("id = ?", account.ID).
			Update("status", TemporaryInternetStatusActive).Error
	})
	if err != nil {
		return nil, err
	}
	return &granted, nil
}

func ExpireDueTemporaryInternetAccess(now time.Time) ([]models.TemporaryInternetAccess, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var due []models.TemporaryInternetAccess
	if err := database.DB.Where("status = ? AND ends_at <= ?", TemporaryInternetAccessActive, now).
		Order("id ASC").Find(&due).Error; err != nil {
		return nil, err
	}
	for index := range due {
		item := &due[index]
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&models.TemporaryInternetAccess{}).
				Where("id = ? AND status = ?", item.ID, TemporaryInternetAccessActive).
				Updates(map[string]interface{}{"status": TemporaryInternetAccessExpired, "expired_at": now})
			if result.Error != nil || result.RowsAffected == 0 {
				return result.Error
			}
			item.Status = TemporaryInternetAccessExpired
			item.ExpiredAt = &now
			return restoreInternetAccountLifecycleStatus(tx, item.InternetAccountID, now)
		})
		if err != nil {
			return nil, err
		}
	}
	return due, nil
}

func CancelTemporaryInternetAccess(id, customerID, cancelledByUserID uint, reason string, now time.Time) (*models.TemporaryInternetAccess, error) {
	if id == 0 || customerID == 0 || cancelledByUserID == 0 {
		return nil, fmt.Errorf("temporary access and cancelling user are required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("cancellation reason is required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	var item models.TemporaryInternetAccess
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND customer_id = ?", id, customerID).First(&item).Error; err != nil {
			return err
		}
		if item.Status != TemporaryInternetAccessActive {
			return fmt.Errorf("only active temporary access can be cancelled")
		}
		item.Status = TemporaryInternetAccessCancelled
		item.CancelledAt = &now
		item.CancelledByUserID = &cancelledByUserID
		item.CancellationReason = reason
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		return restoreInternetAccountLifecycleStatus(tx, item.InternetAccountID, now)
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func restoreInternetAccountLifecycleStatus(tx *gorm.DB, accountID uint, now time.Time) error {
	var account models.CustomerInternetAccount
	if err := tx.First(&account, accountID).Error; err != nil {
		return err
	}
	var activeTemporaryCount int64
	if err := tx.Model(&models.TemporaryInternetAccess{}).
		Where("internet_account_id = ? AND status = ? AND ends_at > ?", accountID, TemporaryInternetAccessActive, now).
		Count(&activeTemporaryCount).Error; err != nil {
		return err
	}
	status := "ACTIVE"
	if activeTemporaryCount > 0 {
		status = TemporaryInternetStatusActive
	} else if account.ExpiryDate != nil && account.ExpiryDate.Before(startOfDay(now)) {
		status = "EXPIRED"
	}
	return tx.Model(&account).Update("status", status).Error
}

func PendingTemporaryAccessDeductionTx(tx *gorm.DB, subscriptionID uint) (time.Duration, []models.TemporaryInternetAccess, error) {
	if tx == nil {
		return 0, nil, errors.New("database transaction is required")
	}
	if !tx.Migrator().HasTable(&models.TemporaryInternetAccess{}) {
		return 0, nil, nil
	}
	var items []models.TemporaryInternetAccess
	if err := tx.Where("subscription_id = ? AND status IN ?", subscriptionID,
		[]string{TemporaryInternetAccessActive, TemporaryInternetAccessExpired}).Order("id ASC").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	var seconds int64
	for _, item := range items {
		seconds += item.GrantedDurationSeconds
	}
	return time.Duration(seconds) * time.Second, items, nil
}
