package models

import "time"

type NetworkRouterHealth struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	RouterID           uint           `gorm:"not null;index:idx_router_health_observed,priority:1" json:"router_id"`
	Router             *NetworkRouter `gorm:"foreignKey:RouterID;constraint:OnDelete:CASCADE" json:"-"`
	ObservedAt         time.Time      `gorm:"not null;index:idx_router_health_observed,priority:2" json:"observed_at"`
	ConnectivityStatus string         `gorm:"size:20;not null" json:"connectivity_status"`
	APIStatus          string         `gorm:"size:20;not null" json:"api_status"`
	LatencyMS          int64          `gorm:"not null;default:0" json:"latency_ms"`
	CPULoad            int            `gorm:"not null;default:0" json:"cpu_load"`
	TotalMemory        int64          `gorm:"not null;default:0" json:"total_memory"`
	FreeMemory         int64          `gorm:"not null;default:0" json:"free_memory"`
	RouterUptime       string         `gorm:"size:100" json:"router_uptime"`
	TCPError           string         `gorm:"size:500" json:"tcp_error"`
	APIError           string         `gorm:"size:500" json:"api_error"`
}
