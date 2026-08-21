package repositories

import (
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerListParams struct {
	Search   string
	Status   string
	Page     int
	PageSize int
	AgentID  uint
	View     string
	Now      time.Time
}

type CustomerSummary struct {
	Subscriptions       int64
	ActiveSubscriptions int64
	Invoices            int64
	OutstandingAmount   float64
	SuccessfulPayments  int64
	TotalPaid           float64
	CancelledInvoices   int64
	VoidedPayments      int64
	VoidedAmount        float64
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
	now := params.Now
	if now.IsZero() {
		now = time.Now()
	}

	if search := strings.TrimSpace(params.Search); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			"customers.customer_code LIKE ? OR customers.full_name LIKE ? OR customers.mobile LIKE ? OR customers.email LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}

	if params.Status != "" {
		query = query.Where("customers.status = ?", params.Status)
	}
	if params.AgentID > 0 {
		query = query.Where("customers.agent_id = ?", params.AgentID)
	}

	switch strings.ToUpper(strings.TrimSpace(params.View)) {
	case "EXPIRED":
		query = query.Where(`EXISTS (SELECT 1 FROM customer_internet_accounts cia WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL AND (cia.status = 'EXPIRED' OR (cia.expiry_date IS NOT NULL AND cia.expiry_date < ?)))`, now)
	case "PENDING":
		query = query.Where(`NOT EXISTS (SELECT 1 FROM customer_internet_accounts cia WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM customer_internet_accounts cia WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL AND cia.status = 'PENDING')`)
	case "RECENT":
		query = query.Where("customers.created_at >= ?", now.AddDate(0, 0, -7))
	case "DISABLED":
		query = query.Where(`EXISTS (SELECT 1 FROM customer_internet_accounts cia WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL AND cia.status IN ('SUSPENDED', 'DISCONNECTED', 'DISABLED'))`)
	case "ONLINE":
		query = query.Where(`EXISTS (SELECT 1 FROM customer_internet_accounts cia JOIN network_router_pppoe_sessions session ON session.router_id = cia.router_id AND session.username = cia.pp_po_e_username AND session.active = ? WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL)`, true)
	case "OFFLINE":
		query = query.Where(`EXISTS (SELECT 1 FROM customer_internet_accounts cia WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL) AND NOT EXISTS (SELECT 1 FROM customer_internet_accounts cia JOIN network_router_pppoe_sessions session ON session.router_id = cia.router_id AND session.username = cia.pp_po_e_username AND session.active = ? WHERE cia.customer_id = customers.id AND cia.deleted_at IS NULL)`, true)
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

func GetCustomerSummary(customerID uint) (*CustomerSummary, error) {
	summary := &CustomerSummary{}

	if err := database.DB.Model(&models.Subscription{}).
		Where("customer_id = ?", customerID).
		Count(&summary.Subscriptions).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Subscription{}).
		Where("customer_id = ? AND status = ?", customerID, "ACTIVE").
		Count(&summary.ActiveSubscriptions).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).
		Where("customer_id = ?", customerID).
		Count(&summary.Invoices).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).
		Where("customer_id = ? AND status <> ?", customerID, "CANCELLED").
		Select("COALESCE(SUM(due_amount), 0)").
		Scan(&summary.OutstandingAmount).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).
		Where("customer_id = ? AND status = ?", customerID, "SUCCESS").
		Count(&summary.SuccessfulPayments).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).
		Where("customer_id = ? AND status = ?", customerID, "SUCCESS").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&summary.TotalPaid).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Invoice{}).Where("customer_id = ? AND status = ?", customerID, "CANCELLED").Count(&summary.CancelledInvoices).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).Where("customer_id = ? AND status = ?", customerID, "VOID").Count(&summary.VoidedPayments).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&models.Payment{}).Where("customer_id = ? AND status = ?", customerID, "VOID").Select("COALESCE(SUM(amount), 0)").Scan(&summary.VoidedAmount).Error; err != nil {
		return nil, err
	}

	return summary, nil
}

func CountActiveCustomerSubscriptions(customerID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Subscription{}).
		Where("customer_id = ? AND status = ?", customerID, "ACTIVE").
		Count(&count).Error
	return count, err
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

func GetCustomerTechnicalProfile(customerID uint) (*models.CustomerTechnicalProfile, error) {
	var profile models.CustomerTechnicalProfile

	if err := database.DB.
		Where("customer_id = ?", customerID).
		First(&profile).Error; err != nil {
		return nil, err
	}

	return &profile, nil
}

func SaveCustomerTechnicalProfile(profile *models.CustomerTechnicalProfile) error {
	if profile.ID == 0 {
		return database.DB.Create(profile).Error
	}

	return database.DB.Save(profile).Error
}

func ListCustomerReferences(customerID uint) ([]models.CustomerReference, error) {
	var references []models.CustomerReference

	err := database.DB.
		Where("customer_id = ?", customerID).
		Order("id ASC").
		Find(&references).Error
	if err != nil {
		return nil, err
	}

	return references, nil
}

func GetCustomerReference(
	customerID uint,
	referenceID uint,
) (*models.CustomerReference, error) {
	var reference models.CustomerReference

	err := database.DB.
		Where(
			"customer_id = ? AND id = ?",
			customerID,
			referenceID,
		).
		First(&reference).Error
	if err != nil {
		return nil, err
	}

	return &reference, nil
}

func CreateCustomerReference(reference *models.CustomerReference) error {
	return database.DB.Create(reference).Error
}

func UpdateCustomerReference(reference *models.CustomerReference) error {
	return database.DB.Save(reference).Error
}

func DeleteCustomerReference(reference *models.CustomerReference) error {
	return database.DB.Delete(reference).Error
}
