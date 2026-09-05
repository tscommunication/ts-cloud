package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type PackageResponse struct {
	ID              uint    `json:"id"`
	PackageCode     string  `json:"package_code"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	Commission      float64 `json:"commission"`
	DownloadSpeed   int     `json:"download_speed"`
	UploadSpeed     int     `json:"upload_speed"`
	BurstDownload   int     `json:"burst_download"`
	BurstUpload     int     `json:"burst_upload"`
	ValidityDays    int     `json:"validity_days"`
	MikroTikProfile string  `json:"mikrotik_profile"`
	RadiusProfile   string  `json:"radius_profile"`
	Status          string  `json:"status"`
	Description     string  `json:"description"`

	FTPEnabled bool `json:"ftp_enabled"`
	FTPQuotaGB int  `json:"ftp_quota_gb"`
}

func ToPackageResponse(pkg models.Package) PackageResponse {
	return PackageResponse{
		ID:              pkg.ID,
		PackageCode:     pkg.PackageCode,
		Name:            pkg.Name,
		Price:           pkg.Price,
		Commission:      pkg.Commission,
		DownloadSpeed:   pkg.DownloadSpeed,
		UploadSpeed:     pkg.UploadSpeed,
		BurstDownload:   pkg.BurstDownload,
		BurstUpload:     pkg.BurstUpload,
		ValidityDays:    pkg.ValidityDays,
		MikroTikProfile: pkg.MikroTikProfile,
		RadiusProfile:   pkg.RadiusProfile,
		Status:          pkg.Status,
		Description:     pkg.Description,
	}
}
