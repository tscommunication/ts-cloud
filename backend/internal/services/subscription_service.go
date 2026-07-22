package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateSubscription(subscription *models.Subscription) error {
	return repositories.CreateSubscription(subscription)
}

func GetSubscriptions() ([]models.Subscription, error) {
	return repositories.GetSubscriptions()
}

func GetSubscriptionByID(id uint) (*models.Subscription, error) {
	return repositories.GetSubscriptionByID(id)
}

func UpdateSubscription(subscription *models.Subscription) error {
	return repositories.UpdateSubscription(subscription)
}

func DeleteSubscription(id uint) error {
	return repositories.DeleteSubscription(id)
}

func GetLastSubscription() (*models.Subscription, error) {
	return repositories.GetLastSubscription()
}
