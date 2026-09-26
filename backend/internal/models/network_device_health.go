package models

import "time"

type NetworkDeviceHealth struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	NetworkDeviceID uint      `gorm:"not null;index:idx_device_health_observed,priority:1" json:"network_device_id"`
	ObservedAt      time.Time `gorm:"not null;index:idx_device_health_observed,priority:2" json:"observed_at"`
	Status          string    `gorm:"size:20;not null" json:"status"`
	Error           string    `gorm:"size:500" json:"error"`
}
