package services

import (
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateCustomer(customer *models.Customer) error {
	return repositories.CreateCustomer(customer)
}

func GetCustomerByID(id uint) (*models.Customer, error) {
	return repositories.GetCustomerByID(id)
}

func GetAllCustomers() ([]models.Customer, error) {
	return repositories.GetAllCustomers()
}

func ListCustomers(params repositories.CustomerListParams) ([]models.Customer, int64, error) {
	return repositories.ListCustomers(params)
}

func UpdateCustomer(customer *models.Customer) error {
	return repositories.UpdateCustomer(customer)
}

func DeleteCustomer(customer *models.Customer) error {
	return repositories.DeleteCustomer(customer)
}
func GetLastCustomer() (*models.Customer, error) {
	return repositories.GetLastCustomer()
}
