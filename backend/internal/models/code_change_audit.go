package models

import (
	"time"

	"gorm.io/gorm"
)

type CodeChangeAudit struct {
	gorm.Model
	EntityType string    `gorm:"size:20;not null;index"`
	EntityID   uint      `gorm:"not null;index"`
	OldCode    string    `gorm:"size:30;not null"`
	NewCode    string    `gorm:"size:30;not null"`
	ChangedBy  uint      `gorm:"not null;index"`
	Reason     string    `gorm:"size:500;not null"`
	ChangedAt  time.Time `gorm:"not null"`
}
