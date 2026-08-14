package models

import "gorm.io/gorm"

type NetworkRouter struct {
	gorm.Model
	Code        string `gorm:"size:30;not null;uniqueIndex" json:"code"`
	Name        string `gorm:"size:120;not null" json:"name"`
	POPID       *uint  `gorm:"index" json:"pop_id"`
	POP         *POP   `gorm:"foreignKey:POPID" json:"-"`
	Host        string `gorm:"size:255;not null;uniqueIndex" json:"host"`
	APIPort     int    `gorm:"not null;default:8729" json:"api_port"`
	APIUsername string `gorm:"size:100;not null" json:"api_username"`
	UseTLS      bool   `gorm:"not null;default:true" json:"use_tls"`
	Status      string `gorm:"size:20;not null;default:ACTIVE;index" json:"status"`
	Remarks     string `gorm:"type:text" json:"remarks"`
}
