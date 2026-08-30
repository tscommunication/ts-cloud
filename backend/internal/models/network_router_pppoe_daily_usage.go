package models

import "time"

// NetworkRouterPPPoEDailyUsage stores counter deltas from read-only RouterOS
// active-session polling. It is deliberately independent of billing records.
type NetworkRouterPPPoEDailyUsage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RouterID  uint      `gorm:"not null;uniqueIndex:idx_pppoe_daily_usage,priority:1;index" json:"router_id"`
	SessionKey string   `gorm:"size:255;not null;uniqueIndex:idx_pppoe_daily_usage,priority:2" json:"-"`
	UsageDate time.Time `gorm:"type:date;not null;uniqueIndex:idx_pppoe_daily_usage,priority:3;index" json:"usage_date"`
	Username  string    `gorm:"size:255;not null;index" json:"username"`
	RxBytes   int64     `gorm:"not null;default:0" json:"rx_bytes"`
	TxBytes   int64     `gorm:"not null;default:0" json:"tx_bytes"`
}

func (NetworkRouterPPPoEDailyUsage) TableName() string { return "network_router_pppoe_daily_usage" }
