package repositories

import (
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreatePayment(payment *models.Payment) error {
	return database.DB.Create(payment).Error
}

// Transaction Create
func CreatePaymentTx(tx *gorm.DB, payment *models.Payment) error {
	return tx.Create(payment).Error
}

// Transaction Save
func SavePaymentTx(tx *gorm.DB, payment *models.Payment) error {
	return tx.Save(payment).Error
}

func GetPayments() ([]models.Payment, error) {
	var payments []models.Payment

	err := database.DB.
		Preload("Invoice").
		Preload("Customer").
		Preload("Subscription").
		Preload("CollectedByUser").
		Order("payment_date DESC, id DESC").
		Find(&payments).Error

	return payments, err
}

func GetPaymentsByAgent(agentID uint) ([]models.Payment, error) {
	var payments []models.Payment
	err := database.DB.
		Joins("JOIN customers ON customers.id = payments.customer_id").
		Where("customers.agent_id = ?", agentID).
		Preload("Invoice").Preload("Customer").Preload("Subscription").Preload("CollectedByUser").
		Order("payments.payment_date DESC, payments.id DESC").
		Find(&payments).Error
	return payments, err
}

func PaymentBelongsToAgent(paymentID, agentID uint) (bool, error) {
	var count int64
	err := database.DB.Model(&models.Payment{}).
		Joins("JOIN customers ON customers.id = payments.customer_id").
		Where("payments.id = ? AND customers.agent_id = ?", paymentID, agentID).
		Count(&count).Error
	return count > 0, err
}

func GetPaymentByID(id uint) (*models.Payment, error) {
	var payment models.Payment

	err := database.DB.
		Preload("Invoice").
		Preload("Customer").
		Preload("Subscription").
		Preload("CollectedByUser").
		First(&payment, id).Error

	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func GetPaymentsByCustomer(customerID uint) ([]models.Payment, error) {
	var payments []models.Payment
	err := database.DB.Where("customer_id = ?", customerID).Order("payment_date DESC").Find(&payments).Error
	return payments, err
}

func UpdatePayment(payment *models.Payment) error {
	return database.DB.Save(payment).Error
}
