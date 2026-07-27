package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

// Create FTP Login Log
func CreateFTPLoginLog(
	log *models.FTPLoginLog,
) error {

	return database.DB.Create(log).Error
}


// Get Login History By FTP User
func GetFTPLoginLogsByUserID(
	userID uint,
) ([]models.FTPLoginLog, error) {

	var logs []models.FTPLoginLog

	err := database.DB.
		Where("ftp_user_id = ?", userID).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, err
}


// Get Latest Login
func GetLastFTPLogin(
	userID uint,
) (*models.FTPLoginLog, error) {

	var log models.FTPLoginLog

	err := database.DB.
		Where(
			"ftp_user_id = ? AND login_status = ?",
			userID,
			"SUCCESS",
		).
		Order("created_at DESC").
	First(&log).Error

	return &log, err
}
