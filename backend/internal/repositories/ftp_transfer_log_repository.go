package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateFTPTransferLog(log *models.FTPTransferLog) error {
	return database.DB.Create(log).Error
}

func GetFTPTransferLogs(limit int) ([]models.FTPTransferLog, error) {

	var logs []models.FTPTransferLog

	query := database.DB.
		Order("id DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error

	return logs, err
}
