package models

import (
	"time"

	"gorm.io/gorm"
)

// TemporaryInternetAccess is the durable promise-to-pay access ledger.
// GrantedDurationSeconds becomes a debt against the next regular recharge;
// SettlementPaymentID guarantees that the same grant is deducted only once.
type TemporaryInternetAccess struct {
	gorm.Model

	CustomerID        uint                    `gorm:"not null;index" json:"customer_id"`
	Customer          Customer                `gorm:"foreignKey:CustomerID" json:"-"`
	InternetAccountID uint                    `gorm:"not null;index" json:"internet_account_id"`
	InternetAccount   CustomerInternetAccount `gorm:"foreignKey:InternetAccountID" json:"-"`
	SubscriptionID    uint                    `gorm:"not null;index" json:"subscription_id"`
	Subscription      Subscription            `gorm:"foreignKey:SubscriptionID" json:"-"`

	Status                 string    `gorm:"size:30;not null;index" json:"status"`
	PreSettlementStatus    string    `gorm:"size:30" json:"pre_settlement_status"`
	StartsAt               time.Time `gorm:"not null;index" json:"starts_at"`
	EndsAt                 time.Time `gorm:"not null;index" json:"ends_at"`
	GrantedDurationSeconds int64     `gorm:"not null" json:"granted_duration_seconds"`

	PromisedPaymentAt *time.Time `json:"promised_payment_at"`
	PromisedAmount    float64    `gorm:"not null;default:0" json:"promised_amount"`
	RequestSource     string     `gorm:"size:30;not null" json:"request_source"`
	Reason            string     `gorm:"type:text;not null" json:"reason"`

	GrantedByUserID uint      `gorm:"not null;index" json:"granted_by_user_id"`
	GrantedByUser   User      `gorm:"foreignKey:GrantedByUserID" json:"-"`
	GrantedAt       time.Time `gorm:"not null" json:"granted_at"`

	ExpiredAt          *time.Time `json:"expired_at"`
	CancelledAt        *time.Time `json:"cancelled_at"`
	CancelledByUserID  *uint      `gorm:"index" json:"cancelled_by_user_id"`
	CancellationReason string     `gorm:"type:text" json:"cancellation_reason"`

	SettlementPaymentID *uint      `gorm:"index" json:"settlement_payment_id"`
	SettledAt           *time.Time `json:"settled_at"`
}
