package models

import (
	"time"

	"gorm.io/gorm"
)

type FTPLoginLog struct {
	gorm.Model

	// FTP User Relation
	FTPUserID uint
	FTPUser   FTPUser `gorm:"foreignKey:FTPUserID"`

	// Login Info
	Username string `gorm:"size:100;not null"`

	IPAddress string `gorm:"size:50"`

	LoginStatus string `gorm:"size:20;not null"`
	// SUCCESS
	// FAILED

	LoginTime time.Time

	UserAgent string `gorm:"size:255"`
}
