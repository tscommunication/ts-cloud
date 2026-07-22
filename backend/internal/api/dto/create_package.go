package dto

type CreatePackageRequest struct {
	Name             string  `json:"name" binding:"required"`
	Price            float64 `json:"price"`
	DownloadSpeed    int     `json:"download_speed"`
	UploadSpeed      int     `json:"upload_speed"`
	BurstDownload    int     `json:"burst_download"`
	BurstUpload      int     `json:"burst_upload"`
	ValidityDays     int     `json:"validity_days"`
	MikroTikProfile  string  `json:"mikrotik_profile"`
	RadiusProfile    string  `json:"radius_profile"`
	Description      string  `json:"description"`
}
