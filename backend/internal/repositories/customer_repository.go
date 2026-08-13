package repositories

import (
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerListParams struct {
	Search   string
	Status   string
	Page     int
	PageSize int
}

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

func ListCustomers(params CustomerListParams) ([]models.Customer, int64, error) {
	query := database.DB.Model(&models.Customer{})

	if search := strings.TrimSpace(params.Search); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			"customer_code LIKE ? OR full_name LIKE ? OR mobile LIKE ? OR email LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var customers []models.Customer
	err := query.
		Order("id DESC").
		Limit(params.PageSize).
		Offset((params.Page - 1) * params.PageSize).
		Find(&customers).Error
	if err != nil {
		return nil, 0, err
	}

	return customers, total, nil
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
