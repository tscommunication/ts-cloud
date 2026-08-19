package models

import (
	"time"

	"gorm.io/gorm"
)

// SubscriptionRenewalReversal records the reversal of a previously committed
// subscription renewal.
//
// The original SubscriptionRenewal is never deleted or rewritten. Together,
// the renewal and reversal records form the durable service-period audit trail.
//
// RenewalID and PaymentID are unique because one committed renewal may be
// reversed at most once.
type SubscriptionRenewalReversal struct {
	gorm.Model

	RenewalID uint                `gorm:"not null;uniqueIndex"`
	Renewal   SubscriptionRenewal `gorm:"foreignKey:RenewalID"`

	PaymentID uint    `gorm:"not null;uniqueIndex"`
	Payment   Payment `gorm:"foreignKey:PaymentID"`

	InvoiceID      uint `gorm:"not null;index"`
	CustomerID     uint `gorm:"not null;index"`
	SubscriptionID uint `gorm:"not null;index"`

	PreviousExpiryDate      time.Time `gorm:"not null"`
	RestoredExpiryDate      time.Time `gorm:"not null"`
	PreviousNextBillingDate time.Time `gorm:"not null"`
	RestoredNextBillingDate time.Time `gorm:"not null"`

	Reason string `gorm:"type:text;not null"`

	ReversedByUserID uint      `gorm:"not null;index"`
	ReversedAt       time.Time `gorm:"not null"`
}
