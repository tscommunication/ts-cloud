package models

import "gorm.io/gorm"

type Agent struct {
	gorm.Model
	Code              string     `gorm:"size:30;not null;uniqueIndex"`
	Name              string     `gorm:"size:120;not null"`
	POPID             uint       `gorm:"not null;index"`
	POP               POP        `gorm:"foreignKey:POPID"`
	AgentPOPs         []AgentPOP `gorm:"foreignKey:AgentID"`
	Mobile            string     `gorm:"size:20"`
	Address           string     `gorm:"type:text"`
	CommissionPercent float64    `gorm:"not null;default:0"`
	OpeningBalance    float64    `gorm:"not null;default:0"`
	SourceReference   string     `gorm:"size:255"`
	Status            string     `gorm:"size:20;not null;default:ACTIVE;index"`
}

type AgentPOP struct {
	AgentID uint `gorm:"primaryKey"`
	POPID   uint `gorm:"primaryKey;index"`
	POP     POP  `gorm:"foreignKey:POPID"`
}
