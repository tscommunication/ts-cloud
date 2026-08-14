package models

import "time"

type NetworkRouterAlert struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	RouterID       uint           `gorm:"not null;index" json:"router_id"`
	Router         *NetworkRouter `gorm:"foreignKey:RouterID;constraint:OnDelete:CASCADE" json:"-"`
	Type           string         `gorm:"size:30;not null;index" json:"type"`
	Severity       string         `gorm:"size:20;not null" json:"severity"`
	Status         string         `gorm:"size:20;not null;index" json:"status"`
	Message        string         `gorm:"size:255;not null" json:"message"`
	CurrentValue   float64        `gorm:"not null" json:"current_value"`
	Threshold      float64        `gorm:"not null" json:"threshold"`
	OpenedAt       time.Time      `gorm:"not null" json:"opened_at"`
	LastObservedAt time.Time      `gorm:"not null" json:"last_observed_at"`
	ResolvedAt     *time.Time     `json:"resolved_at"`
}
