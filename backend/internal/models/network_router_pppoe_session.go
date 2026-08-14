package models

import "time"

type NetworkRouterPPPoESession struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	RouterID       uint           `gorm:"not null;uniqueIndex:idx_router_pppoe_session_key,priority:1;index" json:"router_id"`
	Router         *NetworkRouter `gorm:"foreignKey:RouterID;constraint:OnDelete:CASCADE" json:"-"`
	SessionKey     string         `gorm:"size:255;not null;uniqueIndex:idx_router_pppoe_session_key,priority:2" json:"-"`
	Username       string         `gorm:"size:255;not null;index" json:"username"`
	Service        string         `gorm:"size:80" json:"service"`
	CallerID       string         `gorm:"size:255" json:"caller_id"`
	Address        string         `gorm:"size:100" json:"address"`
	Uptime         string         `gorm:"size:100" json:"uptime"`
	SessionID      string         `gorm:"size:100" json:"session_id"`
	Active         bool           `gorm:"not null;default:true;index" json:"active"`
	FirstSeenAt    time.Time      `gorm:"not null" json:"first_seen_at"`
	LastSeenAt     time.Time      `gorm:"not null;index" json:"last_seen_at"`
	DisconnectedAt *time.Time     `json:"disconnected_at"`
}

func (NetworkRouterPPPoESession) TableName() string {
	return "network_router_pppoe_sessions"
}
