package models

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionDateAdjustment struct {
	gorm.Model

	SubscriptionID uint `gorm:"not null;index"`

	OldExpiryDate      time.Time `gorm:"not null"`
	NewExpiryDate      time.Time `gorm:"not null"`
	OldNextBillingDate time.Time `gorm:"not null"`
	NewNextBillingDate time.Time `gorm:"not null"`

	OldStatus string `gorm:"size:20;not null"`
	NewStatus string `gorm:"size:20;not null"`

	Reason string `gorm:"type:text;not null"`

	AdjustedByUserID uint      `gorm:"not null;index"`
	AdjustedAt       time.Time `gorm:"not null"`

	// Explicitly records that this was an administrative
	// date correction/extension and not a billed renewal.
	WithoutBilling bool `gorm:"not null;default:true"`
}
