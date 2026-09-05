package models

import (
	"gorm.io/gorm"
	"time"
)

type ServiceEntitlement struct {
	gorm.Model
	CustomerID     uint          `gorm:"not null;index" json:"customer_id"`
	Customer       Customer      `gorm:"foreignKey:CustomerID" json:"-"`
	SubscriptionID *uint         `gorm:"index" json:"subscription_id,omitempty"`
	Subscription   *Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`

	// ManagedKey identifies system-managed entitlements without restricting
	// manual entitlements. NULL means the entitlement is manually managed.
	ManagedKey *string `gorm:"size:150;uniqueIndex" json:"-"`

	ServiceType       string     `gorm:"size:30;not null;index" json:"service_type"`
	ServiceName       string     `gorm:"size:150;not null" json:"service_name"`
	Username          string     `gorm:"size:255" json:"username"`
	PasswordEncrypted string     `gorm:"type:text" json:"-"`
	Endpoint          string     `gorm:"size:500" json:"endpoint"`
	Status            string     `gorm:"size:20;not null;default:ACTIVE;index" json:"status"`
	ExpiryAt          *time.Time `json:"expiry_at,omitempty"`
	QuotaGB           int        `gorm:"not null;default:0" json:"quota_gb"`
	Remarks           string     `gorm:"type:text" json:"remarks"`
}
