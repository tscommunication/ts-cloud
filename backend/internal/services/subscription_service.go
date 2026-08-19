package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
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

func SetSubscriptionPPPoEPassword(
	subscription *models.Subscription,
	password string,
	keyMaterial string,
) error {
	if subscription == nil {
		return fmt.Errorf("subscription is required")
	}

	password = strings.TrimSpace(password)

	if password == "" {
		return nil
	}

	encrypted, err := security.EncryptSecret(
		password,
		keyMaterial,
	)
	if err != nil {
		return err
	}

	subscription.PPPoEPasswordEncrypted = encrypted

	// Do not persist newly supplied PPPoE secrets in plaintext.
	subscription.PPPoEPassword = ""

	return nil
}

func CreateSubscription(subscription *models.Subscription) error {
	customer, err := repositories.GetCustomerByID(subscription.CustomerID)
	if err != nil {
		return fmt.Errorf("customer not found")
	}
	if customer.Status != "ACTIVE" {
		return fmt.Errorf("subscription requires an active customer")
	}

	pkg, err := repositories.GetPackageByID(subscription.PackageID)
	if err != nil {
		return fmt.Errorf("package not found")
	}
	if pkg.Status != "ACTIVE" {
		return fmt.Errorf("subscription requires an active package")
	}
	if err := ValidateSubscriptionRouter(subscription.RouterID); err != nil {
		return err
	}

	return repositories.CreateSubscription(subscription)
}

func GetSubscriptions() ([]models.Subscription, error) {
	return repositories.GetSubscriptions()
}

func ListSubscriptions(params repositories.SubscriptionListParams, now time.Time) ([]models.Subscription, error) {
	return repositories.ListSubscriptions(params, now)
}

func GetSubscriptionByID(id uint) (*models.Subscription, error) {
	return repositories.GetSubscriptionByID(id)
}

func UpdateSubscription(subscription *models.Subscription) error {
	if err := ValidateSubscriptionRouter(subscription.RouterID); err != nil {
		return err
	}
	return repositories.UpdateSubscription(subscription)
}

func ValidateSubscriptionRouter(routerID uint) error {
	if routerID == 0 {
		return nil
	}
	router, err := repositories.GetNetworkRouter(routerID)
	if err != nil {
		return fmt.Errorf("router not found")
	}
	if router.Status != "ACTIVE" {
		return fmt.Errorf("subscription requires an active router")
	}
	return nil
}

func GetLastSubscription() (*models.Subscription, error) {
	return repositories.GetLastSubscription()
}

func DisconnectSubscription(subscription *models.Subscription) error {
	if subscription.Status == "DISCONNECTED" {
		return fmt.Errorf("subscription is already disconnected")
	}

	subscription.Status = "DISCONNECTED"
	return repositories.UpdateSubscription(subscription)
}
