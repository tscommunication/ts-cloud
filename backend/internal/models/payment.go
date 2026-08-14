package models

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model

	ReceiptNo string `gorm:"uniqueIndex;size:30"`

	// Relations
	InvoiceID uint    `gorm:"not null;index"`
	Invoice   Invoice `gorm:"foreignKey:InvoiceID"`

	CustomerID uint     `gorm:"not null;index"`
	Customer   Customer `gorm:"foreignKey:CustomerID"`

	SubscriptionID uint         `gorm:"not null;index"`
	Subscription   Subscription `gorm:"foreignKey:SubscriptionID"`

	CollectedByUserID  *uint  `gorm:"index"`
	CollectedByUser    *User  `gorm:"foreignKey:CollectedByUserID"`
	CollectedByAgentID *uint  `gorm:"index"`
	CollectedByAgent   *Agent `gorm:"foreignKey:CollectedByAgentID"`

	// Payment Information
	PaymentDate time.Time `gorm:"not null"`

	Amount float64 `gorm:"not null"`

	Method string `gorm:"size:30;default:CASH"`
	// CASH
	// BKASH
	// NAGAD
	// ROCKET
	// BANK

	TransactionID string `gorm:"size:100;index"`

	Status string `gorm:"size:20;default:SUCCESS"`
	// SUCCESS
	// PENDING
	// FAILED
	// CANCELLED
	// VOID

	Reference string `gorm:"size:100"`

	Remarks string `gorm:"type:text"`
}
