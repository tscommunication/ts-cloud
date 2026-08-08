package repositories

import (
	"gorm.io/gorm"
)

type FTPDashboardRepository interface {
	GetTotalUsers() (int64, error)
	GetOnlineUsers() (int64, error)
	GetTodayLogins() (int64, error)
	GetTodayUploads() (int64, error)
	GetTodayDownloads() (int64, error)
	GetTodayUploadBytes() (int64, error)
	GetTodayDownloadBytes() (int64, error)
}

type ftpDashboardRepository struct {
	db *gorm.DB
}

func NewFTPDashboardRepository(db *gorm.DB) FTPDashboardRepository {
	return &ftpDashboardRepository{
		db: db,
	}
}

func (r *ftpDashboardRepository) GetTotalUsers() (int64, error) {

	var count int64

	err := r.db.
		Table("ftp_users").
		Where("deleted_at IS NULL").
		Count(&count).Error

	return count, err
}

func (r *ftpDashboardRepository) GetOnlineUsers() (int64, error) {

	// NOTE:
	// Current schema does not store logout information
	// or active FTP sessions.
	//
	// Returning zero is safer than returning the total
	// number of successful logins, which is misleading.
	//
	// This method will be upgraded in a future sprint
	// after real session tracking is implemented.

	return 0, nil
}

func (r *ftpDashboardRepository) GetTodayLogins() (int64, error) {

	var count int64

	err := r.db.
		Table("ftp_login_logs").
		Where("login_status = ?", "SUCCESS").
		Where("DATE(login_time)=DATE('now','localtime')").
		Count(&count).Error

	return count, err
}

func (r *ftpDashboardRepository) GetTodayUploads() (int64, error) {

	var count int64

	err := r.db.
		Table("ftp_transfer_logs").
		Where("transfer_type = ?", "UPLOAD").
		Where("DATE(created_at)=DATE('now','localtime')").
		Count(&count).Error

	return count, err
}

func (r *ftpDashboardRepository) GetTodayDownloads() (int64, error) {

	var count int64

	err := r.db.
		Table("ftp_transfer_logs").
		Where("transfer_type = ?", "DOWNLOAD").
		Where("DATE(created_at)=DATE('now','localtime')").
		Count(&count).Error

	return count, err
}

func (r *ftpDashboardRepository) GetTodayUploadBytes() (int64, error) {

	var total int64

	err := r.db.
		Table("ftp_transfer_logs").
		Select("COALESCE(SUM(file_size),0)").
		Where("transfer_type = ?", "UPLOAD").
		Where("DATE(created_at)=DATE('now','localtime')").
		Scan(&total).Error

	return total, err
}

func (r *ftpDashboardRepository) GetTodayDownloadBytes() (int64, error) {

	var total int64

	err := r.db.
		Table("ftp_transfer_logs").
		Select("COALESCE(SUM(file_size),0)").
		Where("transfer_type = ?", "DOWNLOAD").
		Where("DATE(created_at)=DATE('now','localtime')").
		Scan(&total).Error

	return total, err
}
