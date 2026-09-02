package models

import "time"

type NetworkRouterPPPoESession struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	RouterID         uint           `gorm:"not null;uniqueIndex:idx_router_pppoe_session_key,priority:1;index" json:"router_id"`
	Router           *NetworkRouter `gorm:"foreignKey:RouterID;constraint:OnDelete:CASCADE" json:"-"`
	SessionKey       string         `gorm:"size:255;not null;uniqueIndex:idx_router_pppoe_session_key,priority:2" json:"-"`
	Username         string         `gorm:"size:255;not null;index" json:"username"`
	Service          string         `gorm:"size:80" json:"service"`
	CallerID         string         `gorm:"size:255" json:"caller_id"`
	Address          string         `gorm:"size:100" json:"address"`
	Uptime           string         `gorm:"size:100" json:"uptime"`
	SessionID        string         `gorm:"size:100" json:"session_id"`
	RxRateBps        int64          `gorm:"not null;default:0" json:"rx_rate_bps"`
	TxRateBps        int64          `gorm:"not null;default:0" json:"tx_rate_bps"`
	RxBytes          int64          `gorm:"not null;default:0" json:"rx_bytes"`
	TxBytes          int64          `gorm:"not null;default:0" json:"tx_bytes"`
	Active           bool           `gorm:"not null;default:true;index" json:"active"`
	FirstSeenAt      time.Time      `gorm:"not null" json:"first_seen_at"`
	LastSeenAt       time.Time      `gorm:"not null;index" json:"last_seen_at"`
	DisconnectedAt   *time.Time     `json:"disconnected_at"`
	DisconnectReason string         `gorm:"size:100" json:"disconnect_reason"`
}

func (NetworkRouterPPPoESession) TableName() string {
	return "network_router_pppoe_sessions"
}

type NetworkRouterPPPoESessionView struct {
	ID                 uint       `json:"id"`
	RouterID           uint       `json:"router_id"`
	RouterCode         string     `json:"router_code"`
	RouterName         string     `json:"router_name"`
	Username           string     `json:"username"`
	Service            string     `json:"service"`
	CallerID           string     `json:"caller_id"`
	Address            string     `json:"address"`
	Uptime             string     `json:"uptime"`
	SessionID          string     `json:"session_id"`
	RxRateBps          int64      `json:"rx_rate_bps"`
	TxRateBps          int64      `json:"tx_rate_bps"`
	RxBytes            int64      `json:"rx_bytes"`
	TxBytes            int64      `json:"tx_bytes"`
	Active             bool       `json:"active"`
	FirstSeenAt        time.Time  `json:"first_seen_at"`
	LastSeenAt         time.Time  `json:"last_seen_at"`
	DisconnectedAt     *time.Time `json:"disconnected_at"`
	DisconnectReason   string     `json:"disconnect_reason"`
	SubscriptionID     *uint      `json:"subscription_id"`
	SubscriptionCode   string     `json:"subscription_code"`
	SubscriptionStatus string     `json:"subscription_status"`
	CustomerID         *uint      `json:"customer_id"`
	CustomerCode       string     `json:"customer_code"`
	CustomerName       string     `json:"customer_name"`
	AgentID            *uint      `json:"agent_id"`
	AgentCode          string     `json:"agent_code"`
	AgentName          string     `json:"agent_name"`
	PackageID          *uint      `json:"package_id"`
	PackageCode        string     `json:"package_code"`
	PackageName        string     `json:"package_name"`
	ONURxPowerDBM      *float64   `json:"onu_rx_power_dbm"`
}
