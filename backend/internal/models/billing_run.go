package models

import (
	"time"

	"gorm.io/gorm"
)

type BillingRun struct {
	gorm.Model
	RunDate      time.Time `gorm:"not null;index" json:"run_date"`
	TriggeredBy  uint      `gorm:"default:0" json:"triggered_by"`
	Status       string    `gorm:"size:20;not null" json:"status"`
	Total        int       `gorm:"default:0" json:"total"`
	CreatedCount int       `gorm:"default:0" json:"created_count"`
	SkippedCount int       `gorm:"default:0" json:"skipped_count"`
	FailedCount  int       `gorm:"default:0" json:"failed_count"`
}

type BillingRunItem struct {
	gorm.Model
	BillingRunID   uint   `gorm:"not null;index"`
	SubscriptionID uint   `gorm:"not null;index"`
	InvoiceID      *uint  `gorm:"index"`
	Status         string `gorm:"size:20;not null"`
	ErrorMessage   string `gorm:"type:text"`
}
