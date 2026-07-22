package models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model

	SubscriptionCode string `gorm:"uniqueIndex;size:30"`

	// Relations
	CustomerID uint `gorm:"not null;index"`
	Customer   Customer `gorm:"foreignKey:CustomerID"`

	PackageID uint `gorm:"not null;index"`
	Package   Package `gorm:"foreignKey:PackageID"`

	// Service Dates
	ActivationDate time.Time `gorm:"not null"`
	BillingDay    int       `gorm:"not null;default:1"`

	NextBillingDate time.Time `gorm:"not null"`
	ExpiryDate      time.Time `gorm:"not null"`

	// Status
	Status string `gorm:"size:20;default:ACTIVE"`
	// ACTIVE, SUSPENDED, EXPIRED, DISCONNECTED

	// Router & PPPoE (Future MikroTik Integration)
	RouterID uint `gorm:"default:0"`

	PPPoEUsername string `gorm:"size:100"`
	PPPoEPassword string `gorm:"size:100"`

	// Billing Info
	LastPaymentDate *time.Time
	LastPaidAmount  float64 `gorm:"default:0"`
	DueAmount       float64 `gorm:"default:0"`

	// Extra Info
	Remarks string `gorm:"type:text"`
}
