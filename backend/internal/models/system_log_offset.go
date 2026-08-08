package models

import (
	"gorm.io/gorm"
)

type SystemLogOffset struct {
	gorm.Model

	ServiceName string `gorm:"size:50;uniqueIndex"`
	LogFile     string `gorm:"size:255"`
	LastOffset  int64
	Inode       uint64
}
