package services

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

// CreateFTPLoginLog records FTP login activity.
func CreateFTPLoginLog(
	ftpUserID uint,
	username string,
	ipAddress string,
	status string,
	userAgent string,
) error {

	log := &models.FTPLoginLog{
		FTPUserID:   ftpUserID,
		Username:    username,
		IPAddress:   ipAddress,
		LoginStatus: status,
		LoginTime:   time.Now(),
		UserAgent:   userAgent,
	}

	return repositories.CreateFTPLoginLog(log)
}

// GetFTPLoginHistory returns login history.
func GetFTPLoginHistory(
	userID uint,
) ([]models.FTPLoginLog, error) {

	return repositories.GetFTPLoginHistory(userID)
}

// GetLatestFTPLogin returns latest login event.
func GetLatestFTPLogin(
	userID uint,
) (*models.FTPLoginLog, error) {

	return repositories.GetLatestFTPLogin(userID)
}
