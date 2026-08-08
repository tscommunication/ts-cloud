package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

// CreateFTPLoginLog inserts a login event.
func CreateFTPLoginLog(log *models.FTPLoginLog) error {

	return database.DB.Create(log).Error
}

// GetFTPLoginHistory returns login history of a user.
func GetFTPLoginHistory(
	ftpUserID uint,
) ([]models.FTPLoginLog, error) {

	var logs []models.FTPLoginLog

	err := database.DB.
		Where("ftp_user_id = ?", ftpUserID).
		Order("login_time DESC").
		Find(&logs).Error

	return logs, err
}

// GetLatestFTPLogin returns latest login event.
func GetLatestFTPLogin(
	ftpUserID uint,
) (*models.FTPLoginLog, error) {

	var log models.FTPLoginLog

	err := database.DB.
		Where("ftp_user_id = ?", ftpUserID).
		Order("login_time DESC").
		First(&log).Error

	if err != nil {
		return nil, err
	}

	return &log, nil
}
