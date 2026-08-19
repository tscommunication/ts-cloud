package models

import (
	"time"

	"gorm.io/gorm"
)

// SubscriptionRenewal is the durable service-period ledger for a billed
// subscription renewal.
//
// A payment may produce at most one renewal. PaymentID is therefore unique
// and acts as the database-level idempotency boundary.
//
// Financial history remains owned by Payment/Invoice. This record captures
// the resulting service-period transition.
type SubscriptionRenewal struct {
	gorm.Model

	PaymentID uint    `gorm:"not null;uniqueIndex"`
	Payment   Payment `gorm:"foreignKey:PaymentID"`

	InvoiceID uint    `gorm:"not null;index"`
	Invoice   Invoice `gorm:"foreignKey:InvoiceID"`

	CustomerID uint     `gorm:"not null;index"`
	Customer   Customer `gorm:"foreignKey:CustomerID"`

	SubscriptionID uint         `gorm:"not null;index"`
	Subscription   Subscription `gorm:"foreignKey:SubscriptionID"`

	OldExpiryDate      time.Time `gorm:"not null"`
	NewExpiryDate      time.Time `gorm:"not null"`
	OldNextBillingDate time.Time `gorm:"not null"`
	NewNextBillingDate time.Time `gorm:"not null"`

	RenewalDate time.Time `gorm:"not null;index"`
	Amount      float64   `gorm:"not null"`

	// PAYMENT is the initial source. Keeping Source explicit allows future
	// renewal origins without changing the ledger structure.
	Source string `gorm:"size:30;not null;default:PAYMENT"`
}
