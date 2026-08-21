package models

import (
	"time"

	"gorm.io/gorm"
)

type FTPUser struct {
	gorm.Model

	// Customer is the permanent owner of this service entitlement.
	CustomerID uint     `gorm:"index"`
	Customer   Customer `gorm:"foreignKey:CustomerID"`

	// Subscription Relation (PPPoE Account)
	SubscriptionID uint         `gorm:"not null;index"`
	Subscription   Subscription `gorm:"foreignKey:SubscriptionID"`

	// FTP Server Relation
	FTPServerID uint      `gorm:"not null;index"`
	FTPServer   FTPServer `gorm:"foreignKey:FTPServerID"`

	// Login
	Username string `gorm:"size:100;not null;uniqueIndex"`

	// Encrypted Password
	Password string `gorm:"size:255;not null"`

	// Storage
	HomeDirectory string `gorm:"size:255;not null"`

	StorageQuotaGB int `gorm:"default:10"`

	// Speed Limit (Future)
	UploadLimitMbps   int `gorm:"default:0"`
	DownloadLimitMbps int `gorm:"default:0"`

	// Status
	Status string `gorm:"size:20;default:ACTIVE"`
	// ACTIVE
	// SUSPENDED
	// DISABLED

	// Monitoring
	LastLogin *time.Time

	LastIP string `gorm:"size:50"`

	TotalUploadBytes   uint64 `gorm:"default:0"`
	TotalDownloadBytes uint64 `gorm:"default:0"`

	Remarks string `gorm:"type:text"`
}
