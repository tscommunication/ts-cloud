package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type FTPUserResponse struct {
	ID uint `json:"id"`

	SubscriptionID uint `json:"subscription_id"`
	FTPServerID    uint `json:"ftp_server_id"`

	Username string `json:"username"`

	HomeDirectory string `json:"home_directory"`

	StorageQuotaGB int `json:"storage_quota_gb"`

	UploadLimitMbps   int `json:"upload_limit_mbps"`
	DownloadLimitMbps int `json:"download_limit_mbps"`

	Status string `json:"status"`

	LastLogin *time.Time `json:"last_login,omitempty"`

	LastIP string `json:"last_ip"`

	TotalUploadBytes   uint64 `json:"total_upload_bytes"`
	TotalDownloadBytes uint64 `json:"total_download_bytes"`

	Remarks string `json:"remarks"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToFTPUserResponse(user models.FTPUser) FTPUserResponse {
	return FTPUserResponse{
		ID:                 user.ID,
		SubscriptionID:     user.SubscriptionID,
		FTPServerID:        user.FTPServerID,
		Username:           user.Username,
		HomeDirectory:      user.HomeDirectory,
		StorageQuotaGB:     user.StorageQuotaGB,
		UploadLimitMbps:    user.UploadLimitMbps,
		DownloadLimitMbps:  user.DownloadLimitMbps,
		Status:             user.Status,
		LastLogin:          user.LastLogin,
		LastIP:             user.LastIP,
		TotalUploadBytes:   user.TotalUploadBytes,
		TotalDownloadBytes: user.TotalDownloadBytes,
		Remarks:            user.Remarks,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}
}
