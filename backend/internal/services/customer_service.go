package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

var (
	bangladeshMobileRegex = regexp.MustCompile(`^01[3-9][0-9]{8}$`)
	customerNIDRegex      = regexp.MustCompile(`^[0-9]{10,17}$`)
)

func ValidateCustomerIdentity(mobile, altMobile, nid string) error {
	mobile = strings.TrimSpace(mobile)
	altMobile = strings.TrimSpace(altMobile)
	nid = strings.TrimSpace(nid)

	if !bangladeshMobileRegex.MatchString(mobile) {
		return fmt.Errorf("mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019")
	}

	if altMobile != "" && !bangladeshMobileRegex.MatchString(altMobile) {
		return fmt.Errorf("alternative mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019")
	}

	if !customerNIDRegex.MatchString(nid) {
		return fmt.Errorf("NID must contain only digits and be between 10 and 17 digits")
	}

	return nil
}

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

func GetCustomerSummary(customerID uint) (*repositories.CustomerSummary, error) {
	return repositories.GetCustomerSummary(customerID)
}

func ArchiveCustomer(customer *models.Customer) error {
	activeSubscriptions, err := repositories.CountActiveCustomerSubscriptions(customer.ID)
	if err != nil {
		return err
	}
	if activeSubscriptions > 0 {
		return fmt.Errorf("customer has %d active subscription(s)", activeSubscriptions)
	}

	customer.Status = "ARCHIVED"
	return repositories.UpdateCustomer(customer)
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
