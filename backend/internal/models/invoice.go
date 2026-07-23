package models

import (
	"time"

	"gorm.io/gorm"
)

type Invoice struct {
	gorm.Model

	InvoiceNo string `gorm:"uniqueIndex;size:30"`

	// Relations
	SubscriptionID uint         `gorm:"not null;index"`
	Subscription   Subscription `gorm:"foreignKey:SubscriptionID"`

	CustomerID uint     `gorm:"not null;index"`
	Customer   Customer `gorm:"foreignKey:CustomerID"`

	PackageID uint    `gorm:"not null;index"`
	Package   Package `gorm:"foreignKey:PackageID"`

	// Billing Period
	BillMonth int `gorm:"not null"`
	BillYear  int `gorm:"not null"`

	IssueDate time.Time `gorm:"not null"`
	DueDate   time.Time `gorm:"not null"`

	// Amount
	PackagePrice float64 `gorm:"default:0"`
	Discount     float64 `gorm:"default:0"`
	Vat          float64 `gorm:"default:0"`
	TotalAmount  float64 `gorm:"default:0"`
	PaidAmount   float64 `gorm:"default:0"`
	DueAmount    float64 `gorm:"default:0"`

	// Status
	Status string `gorm:"size:20;default:UNPAID"`
	// UNPAID
	// PARTIAL
	// PAID
	// CANCELLED

	Remarks string `gorm:"type:text"`
}
