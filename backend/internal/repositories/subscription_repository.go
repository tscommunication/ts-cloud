package repositories

import (
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type SubscriptionListParams struct {
	Status             string
	ExpiringWithinDays int
	// AgentID scopes the list to customers assigned to that agent. It is
	// deliberately a database filter so an agent can never receive another
	// agent's subscription in the API response.
	AgentID uint
}

func CreateSubscription(subscription *models.Subscription) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(subscription).Error; err != nil {
			return err
		}
		return syncInternetLifecycle(tx, subscription)
	})
}

func GetSubscriptions() ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := database.DB.
		Preload("Customer").
		Preload("Package").
		Preload("InternetAccount").
		Preload("InternetAccount.Package").
		Find(&subscriptions).Error

	return subscriptions, err
}

func ListSubscriptions(params SubscriptionListParams, now time.Time) ([]models.Subscription, error) {
	query := database.DB.
		Preload("Customer").
		Preload("Package").
		Preload("InternetAccount").
		Preload("InternetAccount.Package")

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.ExpiringWithinDays > 0 {
		start := beginningOfDay(now)
		end := start.AddDate(0, 0, params.ExpiringWithinDays+1)
		query = query.Where("expiry_date >= ? AND expiry_date < ?", start, end)
	}
	if params.AgentID > 0 {
		query = query.Where("customer_id IN (?)", database.DB.Model(&models.Customer{}).
			Select("id").Where("agent_id = ?", params.AgentID))
	}

	var subscriptions []models.Subscription
	err := query.Order("expiry_date ASC").Find(&subscriptions).Error
	return subscriptions, err
}

func ExpireOverdueSubscriptions(
	now time.Time,
) ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := database.DB.
		Where(
			"status = ? AND expiry_date < ?",
			"ACTIVE",
			beginningOfDay(now),
		).
		Order("id ASC").
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	if len(subscriptions) == 0 {
		return subscriptions, nil
	}

	ids := make([]uint, 0, len(subscriptions))

	for index := range subscriptions {
		ids = append(ids, subscriptions[index].ID)
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Subscription{}).
			Where("id IN ? AND status = ?", ids, "ACTIVE").
			Update("status", "EXPIRED").Error; err != nil {
			return err
		}
		return tx.Model(&models.CustomerInternetAccount{}).
			Where("id IN (?)", tx.Model(&models.Subscription{}).
				Select("internet_account_id").
				Where("id IN ? AND internet_account_id IS NOT NULL", ids)).
			Where("status <> ?", "TEMPORARY_ACTIVE").
			Update("status", "EXPIRED").Error
	})
	if err != nil {
		return nil, err
	}

	for index := range subscriptions {
		subscriptions[index].Status = "EXPIRED"
	}

	return subscriptions, nil
}

func beginningOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func GetSubscriptionByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription

	err := database.DB.
		Preload("Customer").
		Preload("Package").
		Preload("InternetAccount").
		Preload("InternetAccount.Package").
		First(&subscription, id).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func UpdateSubscription(subscription *models.Subscription) error {
	// A subscription loaded with Preload("Package") still contains the old
	// belongs-to relation. Saving associations here can restore that old
	// package_id after a package change. Persist only subscription columns;
	// related records are managed through their own repositories.
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Customer", "Package", "InternetAccount").
			Save(subscription).Error; err != nil {
			return err
		}
		return syncInternetLifecycle(tx, subscription)
	})
}

func syncInternetLifecycle(tx *gorm.DB, subscription *models.Subscription) error {
	if subscription == nil || subscription.InternetAccountID == nil || *subscription.InternetAccountID == 0 {
		return nil
	}
	return tx.Model(&models.CustomerInternetAccount{}).
		Where("id = ?", *subscription.InternetAccountID).
		Updates(map[string]interface{}{
			"package_id":        subscription.PackageID,
			"activation_date":   subscription.ActivationDate,
			"billing_day":       subscription.BillingDay,
			"next_billing_date": subscription.NextBillingDate,
			"expiry_date":       subscription.ExpiryDate,
			"status":            subscription.Status,
		}).Error
}

func GetLastSubscription() (*models.Subscription, error) {
	var subscription models.Subscription

	err := database.DB.
		Order("id DESC").
		First(&subscription).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func GetSubscriptionsByCustomer(customerID uint) ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := database.DB.
		Where("customer_id = ?", customerID).
		Order("id DESC").
		Find(&subscriptions).Error

	return subscriptions, err
}
