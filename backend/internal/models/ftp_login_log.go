package models

import (
	"time"

	"gorm.io/gorm"
)

type FTPLoginLog struct {
	gorm.Model

	// FTP User Relation
	FTPUserID uint `gorm:"not null;index"`
	FTPUser   FTPUser `gorm:"foreignKey:FTPUserID"`

	// Login Information
	Username string `gorm:"size:100;not null"`

	IPAddress string `gorm:"size:50"`

	// SUCCESS / FAILED
	LoginStatus string `gorm:"size:20;not null;index"`

	// Login Time
	LoginTime time.Time `gorm:"autoCreateTime"`

	// Client Information
	UserAgent string `gorm:"size:255"`
}
