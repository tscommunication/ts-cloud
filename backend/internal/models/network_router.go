package models

import (
	"time"

	"gorm.io/gorm"
)

type NetworkRouter struct {
	gorm.Model
	Code                 string     `gorm:"size:30;not null;uniqueIndex" json:"code"`
	Name                 string     `gorm:"size:120;not null" json:"name"`
	POPID                *uint      `gorm:"index" json:"pop_id"`
	POP                  *POP       `gorm:"foreignKey:POPID" json:"-"`
	Host                 string     `gorm:"size:255;not null;uniqueIndex" json:"host"`
	APIPort              int        `gorm:"not null;default:8729" json:"api_port"`
	APIUsername          string     `gorm:"size:100;not null" json:"api_username"`
	APIPasswordEncrypted string     `gorm:"type:text" json:"-"`
	UseTLS               bool       `gorm:"not null;default:true" json:"use_tls"`
	Status               string     `gorm:"size:20;not null;default:ACTIVE;index" json:"status"`
	Remarks              string     `gorm:"type:text" json:"remarks"`
	ConnectivityStatus   string     `gorm:"size:20;not null;default:UNKNOWN;index" json:"connectivity_status"`
	LastCheckedAt        *time.Time `json:"last_checked_at"`
	LastLatencyMS        int64      `gorm:"not null;default:0" json:"last_latency_ms"`
	LastConnectionError  string     `gorm:"size:500" json:"last_connection_error"`
	LastTCPError         string     `gorm:"size:500" json:"last_tcp_error"`
	LastAPIError         string     `gorm:"size:500" json:"last_api_error"`
	APIStatus            string     `gorm:"size:20;not null;default:UNKNOWN;index" json:"api_status"`
	LastAuthenticatedAt  *time.Time `json:"last_authenticated_at"`
	RouterIdentity       string     `gorm:"size:120" json:"router_identity"`
	RouterOSVersion      string     `gorm:"size:100" json:"routeros_version"`
	BoardName            string     `gorm:"size:120" json:"board_name"`
	RouterUptime         string     `gorm:"size:100" json:"router_uptime"`
	CPULoad              int        `gorm:"not null;default:0" json:"cpu_load"`
	TotalMemory          int64      `gorm:"not null;default:0" json:"total_memory"`
	FreeMemory           int64      `gorm:"not null;default:0" json:"free_memory"`
}
