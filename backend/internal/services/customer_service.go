package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

var (
	bangladeshMobileRegex = regexp.MustCompile(`^01[3-9][0-9]{8}$`)
	customerNIDRegex      = regexp.MustCompile(`^[0-9]{10,17}$`)
)

var (
	ErrCustomerMobileExists = errors.New("customer mobile already exists")
	ErrCustomerNIDExists    = errors.New("customer NID already exists")
)

func TranslateCustomerWriteError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "idx_customers_mobile_unique":
		return ErrCustomerMobileExists
	case "idx_customers_nid_unique":
		return ErrCustomerNIDExists
	default:
		return err
	}
}

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
	provisionalBytes := make([]byte, 8)
	if _, err := rand.Read(provisionalBytes); err != nil {
		return fmt.Errorf("generate provisional customer identity: %w", err)
	}

	credentialBytes := make([]byte, 32)
	if _, err := rand.Read(credentialBytes); err != nil {
		return fmt.Errorf("generate inactive customer credential: %w", err)
	}
	credentialHash, err := bcrypt.GenerateFromPassword(
		[]byte(hex.EncodeToString(credentialBytes)),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return fmt.Errorf("secure inactive customer credential: %w", err)
	}

	createErr := database.DB.Transaction(func(tx *gorm.DB) error {
		customer.CustomerCode = "PENDING-" + hex.EncodeToString(provisionalBytes)
		if err := tx.Create(customer).Error; err != nil {
			return err
		}

		customer.CustomerCode = fmt.Sprintf("CUS-%06d", customer.ID)
		if err := tx.Model(customer).
			Update("customer_code", customer.CustomerCode).Error; err != nil {
			return err
		}

		customerID := customer.ID
		identity := models.User{
			Name:       customer.FullName,
			Username:   customer.CustomerCode,
			Email:      strings.ToLower(customer.CustomerCode) + "@customer.invalid",
			Password:   string(credentialHash),
			Role:       "customer",
			Active:     false,
			CustomerID: &customerID,
		}
		if err := tx.Create(&identity).Error; err != nil {
			return err
		}
		if err := tx.Model(&identity).Update("active", false).Error; err != nil {
			return err
		}
		return createCustomerNotification(tx, customer)
	})

	return TranslateCustomerWriteError(createErr)
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
	return TranslateCustomerWriteError(
		repositories.UpdateCustomer(customer),
	)
}

func DeleteCustomer(customer *models.Customer) error {
	return repositories.DeleteCustomer(customer)
}
func GetLastCustomer() (*models.Customer, error) {
	return repositories.GetLastCustomer()
}
