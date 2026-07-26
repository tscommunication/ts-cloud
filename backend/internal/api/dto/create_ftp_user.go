package dto

type CreateFTPUserRequest struct {
	SubscriptionID uint `json:"subscription_id" binding:"required"`

	FTPServerID uint `json:"ftp_server_id" binding:"required"`

	Username string `json:"username" binding:"required"`

	Password string `json:"password" binding:"required"`

	HomeDirectory string `json:"home_directory" binding:"required"`

	StorageQuotaGB int `json:"storage_quota_gb"`

	UploadLimitMbps int `json:"upload_limit_mbps"`

	DownloadLimitMbps int `json:"download_limit_mbps"`

	Status string `json:"status"`

	Remarks string `json:"remarks"`
}
