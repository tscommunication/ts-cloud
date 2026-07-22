package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

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

func DeleteSubscription(id uint) error {
	return database.DB.Delete(&models.Subscription{}, id).Error
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
