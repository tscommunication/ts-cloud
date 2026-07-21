package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateCustomer(customer *models.Customer) error {
	return database.DB.Create(customer).Error
}

func GetCustomerByID(id uint) (*models.Customer, error) {
	var customer models.Customer

	err := database.DB.First(&customer, id).Error
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func GetAllCustomers() ([]models.Customer, error) {
	var customers []models.Customer

	err := database.DB.Find(&customers).Error
	if err != nil {
		return nil, err
	}

	return customers, nil
}

func UpdateCustomer(customer *models.Customer) error {
	return database.DB.Save(customer).Error
}

func DeleteCustomer(customer *models.Customer) error {
	return database.DB.Delete(customer).Error
}
func GetLastCustomer() (*models.Customer, error) {
	var customer models.Customer

	err := database.DB.
		Order("id DESC").
		First(&customer).Error

	if err != nil {
		return nil, err
	}

	return &customer, nil
}
