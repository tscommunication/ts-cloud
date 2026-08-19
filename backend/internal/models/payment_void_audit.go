package models

import (
	"time"

	"gorm.io/gorm"
)

type PaymentVoidAudit struct {
	gorm.Model

	PaymentID uint `gorm:"not null;uniqueIndex"`

	InvoiceID      uint `gorm:"not null;index"`
	CustomerID     uint `gorm:"not null;index"`
	SubscriptionID uint `gorm:"not null;index"`

	ReceiptNo string `gorm:"size:30;not null"`

	Amount float64 `gorm:"not null"`

	PreviousStatus string `gorm:"size:20;not null"`
	NewStatus      string `gorm:"size:20;not null"`

	Reason string `gorm:"type:text;not null"`

	VoidedByUserID uint      `gorm:"not null;index"`
	VoidedAt       time.Time `gorm:"not null"`
}
