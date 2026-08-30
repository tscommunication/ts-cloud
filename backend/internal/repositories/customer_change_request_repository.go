package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateCustomerChangeRequest(row *models.CustomerChangeRequest) error {
	return database.DB.Create(row).Error
}

func GetCustomerChangeRequest(id uint) (*models.CustomerChangeRequest, error) {
	var row models.CustomerChangeRequest
	err := database.DB.First(&row, id).Error
	return &row, err
}

func ListCustomerChangeRequests(status string, agentID uint) ([]models.CustomerChangeRequest, error) {
	var rows []models.CustomerChangeRequest
	query := database.DB.Order("created_at DESC, id DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func SaveCustomerChangeRequest(row *models.CustomerChangeRequest) error {
	return database.DB.Save(row).Error
}

func PendingCustomerChangeRequestExists(customerID uint, changeType string) (bool, error) {
	var count int64
	err := database.DB.Model(&models.CustomerChangeRequest{}).Where("customer_id = ? AND type = ? AND status = ?", customerID, changeType, "PENDING").Count(&count).Error
	return count > 0, err
}
