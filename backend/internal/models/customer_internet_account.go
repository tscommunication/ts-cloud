package models

import (
	"time"

	"gorm.io/gorm"
)

// CustomerInternetAccount is the customer-owned technical identity used for
// MikroTik PPPoE access. Billing subscriptions link to this account but do not
// own its username, password, router, or network bindings.
type CustomerInternetAccount struct {
	gorm.Model

	AccountCode string   `gorm:"size:30;uniqueIndex;not null" json:"account_code"`
	CustomerID  uint     `gorm:"not null;uniqueIndex" json:"customer_id"`
	Customer    Customer `gorm:"foreignKey:CustomerID" json:"-"`

	RouterID uint `gorm:"not null;index" json:"router_id"`

	// Internet lifecycle belongs to the customer-owned PPPoE account. These
	// fields are backfilled from the linked legacy internet subscription while
	// the billing APIs transition to this canonical source of truth.
	PackageID       uint       `gorm:"index" json:"package_id"`
	Package         Package    `gorm:"foreignKey:PackageID" json:"-"`
	ActivationDate  *time.Time `json:"activation_date"`
	BillingDay      int        `gorm:"not null;default:1" json:"billing_day"`
	NextBillingDate *time.Time `json:"next_billing_date"`
	ExpiryDate      *time.Time `json:"expiry_date"`

	PPPoEUsername          string `gorm:"size:100;uniqueIndex;not null" json:"pppoe_username"`
	PPPoEPasswordEncrypted string `gorm:"column:pp_po_e_password_encrypted;type:text;not null" json:"-"`

	Status string `gorm:"size:20;not null;default:ACTIVE" json:"status"`

	MACAddress      string `gorm:"size:50" json:"mac_address"`
	StaticIPAddress string `gorm:"size:64" json:"static_ip_address"`

	SyncIntervalMinutes int        `gorm:"not null;default:30" json:"sync_interval_minutes"`
	LastSyncedAt        *time.Time `json:"last_synced_at"`
	LastSyncError       string     `gorm:"type:text" json:"last_sync_error"`

	LegacySubscriptionID *uint `gorm:"uniqueIndex" json:"-"`
}
