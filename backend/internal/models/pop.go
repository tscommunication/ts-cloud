package models

import "gorm.io/gorm"

type POP struct {
	gorm.Model
	Code            string `gorm:"size:30;not null;uniqueIndex"`
	Name            string `gorm:"size:120;not null"`
	ManagerName     string `gorm:"size:120"`
	Mobile          string `gorm:"size:20"`
	Address         string `gorm:"type:text"`
	SourceReference string `gorm:"size:255"`
	Status          string `gorm:"size:20;not null;default:ACTIVE;index"`
}
