package repositories

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

func CreateSubscriptionRenewalTx(
	tx *gorm.DB,
	renewal *models.SubscriptionRenewal,
) error {
	return tx.Create(renewal).Error
}

func GetSubscriptionRenewalByPaymentID(
	paymentID uint,
) (*models.SubscriptionRenewal, error) {
	var renewal models.SubscriptionRenewal

	err := database.DB.
		Where("payment_id = ?", paymentID).
		First(&renewal).Error

	if err != nil {
		return nil, err
	}

	return &renewal, nil
}

func SubscriptionRenewalExistsByPaymentIDTx(
	tx *gorm.DB,
	paymentID uint,
) (bool, error) {
	var count int64

	if err := tx.Model(&models.SubscriptionRenewal{}).
		Where("payment_id = ?", paymentID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func IsSubscriptionRenewalNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
