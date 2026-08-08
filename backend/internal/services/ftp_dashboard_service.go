package services

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type FTPDashboardSummary struct {
	TotalUsers         int64 `json:"total_users"`
	OnlineUsers        int64 `json:"online_users"`
	TodayLogins        int64 `json:"today_logins"`
	TodayUploads       int64 `json:"today_uploads"`
	TodayDownloads     int64 `json:"today_downloads"`
	TodayUploadBytes   int64 `json:"today_upload_bytes"`
	TodayDownloadBytes int64 `json:"today_download_bytes"`
}

func GetFTPDashboardSummary() (*FTPDashboardSummary, error) {

	repo := repositories.NewFTPDashboardRepository(database.DB)

	totalUsers, err := repo.GetTotalUsers()
	if err != nil {
		return nil, err
	}

	onlineUsers, err := repo.GetOnlineUsers()
	if err != nil {
		return nil, err
	}

	todayLogins, err := repo.GetTodayLogins()
	if err != nil {
		return nil, err
	}

	todayUploads, err := repo.GetTodayUploads()
	if err != nil {
		return nil, err
	}

	todayDownloads, err := repo.GetTodayDownloads()
	if err != nil {
		return nil, err
	}

	todayUploadBytes, err := repo.GetTodayUploadBytes()
	if err != nil {
		return nil, err
	}

	todayDownloadBytes, err := repo.GetTodayDownloadBytes()
	if err != nil {
		return nil, err
	}

	return &FTPDashboardSummary{
		TotalUsers:         totalUsers,
		OnlineUsers:        onlineUsers,
		TodayLogins:        todayLogins,
		TodayUploads:       todayUploads,
		TodayDownloads:     todayDownloads,
		TodayUploadBytes:   todayUploadBytes,
		TodayDownloadBytes: todayDownloadBytes,
	}, nil
}

// Backward compatibility for handlers
func GetFTPDashboard() (*FTPDashboardSummary, error) {
	return GetFTPDashboardSummary()
}
