package models

import (
	"time"

	"gorm.io/gorm"
)

type NetworkDevice struct {
	gorm.Model
	Code                string          `gorm:"size:30;not null;uniqueIndex" json:"code"`
	Name                string          `gorm:"size:120;not null" json:"name"`
	DeviceType          string          `gorm:"size:20;not null;index" json:"device_type"`
	Vendor              string          `gorm:"size:40;not null;index" json:"vendor"`
	DeviceModel         string          `gorm:"column:model;size:120;not null" json:"model"`
	OLTType             string          `gorm:"size:30" json:"olt_type"`
	POPID               *uint           `gorm:"index" json:"pop_id"`
	POP                 *POP            `gorm:"foreignKey:POPID" json:"-"`
	ManagementIP        string          `gorm:"size:255;not null;uniqueIndex" json:"management_ip"`
	MonitoringProtocol  string          `gorm:"size:20;not null;index" json:"monitoring_protocol"`
	SNMPVersion         string          `gorm:"size:10" json:"snmp_version"`
	SNMPPort            int             `gorm:"not null;default:161" json:"snmp_port"`
	ManagementPort      int             `gorm:"not null;default:0" json:"management_port"`
	SNMPUsername        string          `gorm:"size:100" json:"snmp_username"`
	SNMPSecretEncrypted string          `gorm:"type:text" json:"-"`
	PollingInterval     int             `gorm:"not null;default:300" json:"polling_interval_seconds"`
	MonitoringEnabled   bool            `gorm:"not null;default:true;index" json:"monitoring_enabled"`
	MonitoringStatus    string          `gorm:"size:20;not null;default:UNKNOWN;index" json:"monitoring_status"`
	LastPolledAt        *time.Time      `json:"last_polled_at"`
	LastError           string          `gorm:"size:500" json:"last_error"`
	Remarks             string          `gorm:"type:text" json:"remarks"`
	Routers             []NetworkRouter `gorm:"many2many:network_device_routers" json:"-"`
}
