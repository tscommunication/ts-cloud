package dto

type FTPUserStatsResponse struct {
	ID                 uint    `json:"id"`
	Username           string  `json:"username"`
	Status             string  `json:"status"`

	UsedStorageBytes   uint64  `json:"used_storage_bytes"`
	QuotaBytes         uint64  `json:"quota_bytes"`
	QuotaGB            int     `json:"quota_gb"`

	UsagePercent       float64 `json:"usage_percent"`

	LastLogin          *string `json:"last_login,omitempty"`
	LastIP             string  `json:"last_ip"`

	TotalUploadBytes   uint64  `json:"total_upload_bytes"`
	TotalDownloadBytes uint64  `json:"total_download_bytes"`
}
