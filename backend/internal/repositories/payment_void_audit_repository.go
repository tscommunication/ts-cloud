package repositories

import (
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreatePaymentVoidAuditTx(
	tx *gorm.DB,
	audit *models.PaymentVoidAudit,
) error {
	return tx.Create(audit).Error
}

func GetPaymentVoidAuditByPaymentID(
	paymentID uint,
) (*models.PaymentVoidAudit, error) {
	var audit models.PaymentVoidAudit

	if err := database.DB.
		Where("payment_id = ?", paymentID).
		First(&audit).Error; err != nil {
		return nil, err
	}

	return &audit, nil
}
