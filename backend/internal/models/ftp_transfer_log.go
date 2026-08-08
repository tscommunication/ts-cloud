package models

import (
	"time"

	"gorm.io/gorm"
)

type FTPTransferLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	FTPUserID uint   `json:"ftp_user_id"`
	Username  string `json:"username"`

	TransferType string `json:"transfer_type"` // UPLOAD / DOWNLOAD
	Filename     string `json:"filename"`

	FileSize int64 `json:"file_size"`

	IPAddress string `json:"ip_address"`

	TransferTime time.Time `json:"transfer_time"`
}
