package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerProvisionRequestListParams struct {
	Status  string
	AgentID uint
}

func CreateCustomerProvisionRequest(request *models.CustomerProvisionRequest) error {
	return database.DB.Create(request).Error
}

func GetCustomerProvisionRequestByID(id uint) (*models.CustomerProvisionRequest, error) {
	var request models.CustomerProvisionRequest

	err := database.DB.First(&request, id).Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func GetCustomerProvisionRequestByCode(code string) (*models.CustomerProvisionRequest, error) {
	var request models.CustomerProvisionRequest

	err := database.DB.
		Where("request_code = ?", code).
		First(&request).Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func ListCustomerProvisionRequests(
	params CustomerProvisionRequestListParams,
) ([]models.CustomerProvisionRequest, error) {
	var requests []models.CustomerProvisionRequest

	query := database.DB.Model(&models.CustomerProvisionRequest{})

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if params.AgentID > 0 {
		query = query.Where("agent_id = ?", params.AgentID)
	}

	err := query.
		Order("id DESC").
		Find(&requests).Error

	return requests, err
}

func UpdateCustomerProvisionRequest(
	request *models.CustomerProvisionRequest,
) error {
	return database.DB.Save(request).Error
}

func GetLastCustomerProvisionRequest() (*models.CustomerProvisionRequest, error) {
	var request models.CustomerProvisionRequest

	err := database.DB.
		Order("id DESC").
		First(&request).Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}
