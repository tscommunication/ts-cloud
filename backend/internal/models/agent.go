package models

import "gorm.io/gorm"

type Agent struct {
	gorm.Model
	Code                string               `gorm:"size:30;not null;uniqueIndex"`
	Name                string               `gorm:"size:120;not null"`
	POPID               uint                 `gorm:"not null;index"`
	POP                 POP                  `gorm:"foreignKey:POPID"`
	AgentPOPs           []AgentPOP           `gorm:"foreignKey:AgentID"`
	AgentPackages       []AgentPackage       `gorm:"foreignKey:AgentID"`
	AgentRouters        []AgentRouter        `gorm:"foreignKey:AgentID"`
	AgentNetworkDevices []AgentNetworkDevice `gorm:"foreignKey:AgentID"`
	Mobile              string               `gorm:"size:20"`
	Address             string               `gorm:"type:text"`
	CommissionPercent   float64              `gorm:"not null;default:0"`
	OpeningBalance      float64              `gorm:"not null;default:0"`
	SourceReference     string               `gorm:"size:255"`
	Status              string               `gorm:"size:20;not null;default:ACTIVE;index"`
}

type AgentRouter struct {
	AgentID  uint          `gorm:"primaryKey"`
	RouterID uint          `gorm:"primaryKey;index"`
	Router   NetworkRouter `gorm:"foreignKey:RouterID"`
}

type AgentNetworkDevice struct {
	AgentID         uint          `gorm:"primaryKey"`
	NetworkDeviceID uint          `gorm:"primaryKey;index"`
	NetworkDevice   NetworkDevice `gorm:"foreignKey:NetworkDeviceID;constraint:OnDelete:CASCADE"`
}

type AgentPackage struct {
	AgentID   uint    `gorm:"primaryKey"`
	PackageID uint    `gorm:"primaryKey;index"`
	Package   Package `gorm:"foreignKey:PackageID"`
}

type AgentPOP struct {
	AgentID uint `gorm:"primaryKey"`
	POPID   uint `gorm:"primaryKey;index"`
	POP     POP  `gorm:"foreignKey:POPID"`
}
