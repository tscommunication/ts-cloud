package repositories

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

func CreateSubscriptionRenewalReversalTx(
	tx *gorm.DB,
	reversal *models.SubscriptionRenewalReversal,
) error {
	return tx.Create(reversal).Error
}

func GetSubscriptionRenewalReversalByPaymentID(
	paymentID uint,
) (*models.SubscriptionRenewalReversal, error) {
	var reversal models.SubscriptionRenewalReversal

	err := database.DB.
		Where("payment_id = ?", paymentID).
		First(&reversal).Error

	if err != nil {
		return nil, err
	}

	return &reversal, nil
}

func SubscriptionRenewalReversalExistsByPaymentIDTx(
	tx *gorm.DB,
	paymentID uint,
) (bool, error) {
	var count int64

	if err := tx.Model(&models.SubscriptionRenewalReversal{}).
		Where("payment_id = ?", paymentID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func IsSubscriptionRenewalReversalNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
