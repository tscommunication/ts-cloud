package models

import "time"

type NetworkRouterPPPSecret struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	RouterID      uint           `gorm:"not null;uniqueIndex:idx_router_ppp_secret_key,priority:1;index" json:"router_id"`
	Router        *NetworkRouter `gorm:"foreignKey:RouterID;constraint:OnDelete:CASCADE" json:"-"`
	RouterOSID    string         `gorm:"size:100;not null;uniqueIndex:idx_router_ppp_secret_key,priority:2" json:"routeros_id"`
	Username      string         `gorm:"size:255;not null;index" json:"username"`
	Service       string         `gorm:"size:80" json:"service"`
	Profile       string         `gorm:"size:255" json:"profile"`
	CallerID      string         `gorm:"size:255" json:"caller_id"`
	RemoteAddress string         `gorm:"size:100" json:"remote_address"`
	Disabled      bool           `gorm:"not null;default:false" json:"disabled"`
	Present       bool           `gorm:"not null;default:true;index" json:"present"`
	FirstSeenAt   time.Time      `gorm:"not null" json:"first_seen_at"`
	LastSeenAt    time.Time      `gorm:"not null;index" json:"last_seen_at"`
}

type NetworkRouterPPPSecretView struct {
	ID                 uint      `json:"id"`
	RouterID           uint      `json:"router_id"`
	RouterCode         string    `json:"router_code"`
	RouterName         string    `json:"router_name"`
	Username           string    `json:"username"`
	Service            string    `json:"service"`
	Profile            string    `json:"profile"`
	CallerID           string    `json:"caller_id"`
	RemoteAddress      string    `json:"remote_address"`
	Disabled           bool      `json:"disabled"`
	Present            bool      `json:"present"`
	LastSeenAt         time.Time `json:"last_seen_at"`
	SubscriptionID     *uint     `json:"subscription_id"`
	SubscriptionCode   string    `json:"subscription_code"`
	SubscriptionStatus string    `json:"subscription_status"`
	CustomerID         *uint     `json:"customer_id"`
	CustomerCode       string    `json:"customer_code"`
	CustomerName       string    `json:"customer_name"`
}
