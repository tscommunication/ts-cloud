package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateSubscriptionDateAdjustment(
	row *models.SubscriptionDateAdjustment,
) error {
	return database.DB.Create(row).Error
}

func GetSubscriptionDateAdjustments(
	subscriptionID uint,
) ([]models.SubscriptionDateAdjustment, error) {
	var rows []models.SubscriptionDateAdjustment

	err := database.DB.
		Where("subscription_id = ?", subscriptionID).
		Order("adjusted_at DESC, id DESC").
		Find(&rows).Error

	return rows, err
}
