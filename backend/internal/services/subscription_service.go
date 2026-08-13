package services

import (
	"fmt"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func SuspendSubscription(subscription *models.Subscription) error {
	if subscription.Status != "ACTIVE" {
		return fmt.Errorf("only active subscriptions can be suspended")
	}

	subscription.Status = "SUSPENDED"
	return repositories.UpdateSubscription(subscription)
}

func ActivateSubscription(subscription *models.Subscription, now time.Time) error {
	if subscription.Status != "SUSPENDED" {
		return fmt.Errorf("only suspended subscriptions can be activated")
	}
	if subscription.ExpiryDate.Before(startOfDay(now)) {
		return fmt.Errorf("expired subscription must be renewed before activation")
	}

	subscription.Status = "ACTIVE"
	return repositories.UpdateSubscription(subscription)
}

func RenewSubscription(subscription *models.Subscription, months int, now time.Time) error {
	if months < 1 || months > 12 {
		return fmt.Errorf("renewal months must be between 1 and 12")
	}
	if subscription.Status == "DISCONNECTED" {
		return fmt.Errorf("disconnected subscriptions cannot be renewed")
	}

	baseDate := subscription.ExpiryDate
	if baseDate.Before(startOfDay(now)) {
		baseDate = startOfDay(now)
	}

	renewedUntil := baseDate.AddDate(0, months, 0)
	subscription.ExpiryDate = renewedUntil
	subscription.NextBillingDate = renewedUntil
	subscription.Status = "ACTIVE"

	return repositories.UpdateSubscription(subscription)
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

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
