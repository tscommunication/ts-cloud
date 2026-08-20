package repositories

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type SubscriptionListParams struct {
	Status             string
	ExpiringWithinDays int
}

func CreateSubscription(subscription *models.Subscription) error {
	return database.DB.Create(subscription).Error
}

func GetSubscriptions() ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := database.DB.
		Preload("Customer").
		Preload("Package").
		Find(&subscriptions).Error

	return subscriptions, err
}

func ListSubscriptions(params SubscriptionListParams, now time.Time) ([]models.Subscription, error) {
	query := database.DB.
		Preload("Customer").
		Preload("Package")

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.ExpiringWithinDays > 0 {
		start := beginningOfDay(now)
		end := start.AddDate(0, 0, params.ExpiringWithinDays+1)
		query = query.Where("expiry_date >= ? AND expiry_date < ?", start, end)
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

	err = database.DB.
		Model(&models.Subscription{}).
		Where("id IN ? AND status = ?", ids, "ACTIVE").
		Update("status", "EXPIRED").Error
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
		First(&subscription, id).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func UpdateSubscription(subscription *models.Subscription) error {
	return database.DB.Save(subscription).Error
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
