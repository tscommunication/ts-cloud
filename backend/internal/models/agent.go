package models

import "gorm.io/gorm"

type Agent struct {
	gorm.Model
	Code              string  `gorm:"size:30;not null;uniqueIndex"`
	Name              string  `gorm:"size:120;not null"`
	POPID             uint    `gorm:"not null;index"`
	POP               POP     `gorm:"foreignKey:POPID"`
	Mobile            string  `gorm:"size:20"`
	Address           string  `gorm:"type:text"`
	CommissionPercent float64 `gorm:"not null;default:0"`
	Status            string  `gorm:"size:20;not null;default:ACTIVE;index"`
}
