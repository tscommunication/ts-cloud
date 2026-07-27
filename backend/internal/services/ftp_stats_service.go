package services

import (
	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func GetFTPUserStats(id uint) (*dto.FTPUserStatsResponse, error) {

	user, err := repositories.GetFTPUserByID(id)
	if err != nil {
		return nil, err
	}

	usedBytes, err := linux.GetDirectorySize(user.HomeDirectory)
	if err != nil {
		return nil, err
	}

	quotaBytes := uint64(user.StorageQuotaGB) * 1024 * 1024 * 1024

	var usagePercent float64

	if quotaBytes > 0 {
		usagePercent = (float64(usedBytes) / float64(quotaBytes)) * 100
	}

	var lastLogin *string

	if user.LastLogin != nil {
		t := user.LastLogin.Format("2006-01-02 15:04:05")
		lastLogin = &t
	}

	return &dto.FTPUserStatsResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Status:             user.Status,

		UsedStorageBytes:   usedBytes,
		QuotaBytes:         quotaBytes,
		QuotaGB:            user.StorageQuotaGB,
		UsagePercent:       usagePercent,

		LastLogin:          lastLogin,
		LastIP:             user.LastIP,

		TotalUploadBytes:   user.TotalUploadBytes,
		TotalDownloadBytes: user.TotalDownloadBytes,
	}, nil
}
