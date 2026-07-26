package models

import "gorm.io/gorm"

type FTPServer struct {
	gorm.Model

	// Basic Information
	Name string `gorm:"size:100;not null"`

	// Driver
	Driver string `gorm:"size:30;default:VSFTPD"`

	// Connection
	Host string `gorm:"size:100;not null"`
	Port int `gorm:"default:21"`

	// Login
	Username string `gorm:"size:100;not null"`
	Password string `gorm:"size:255;not null"`

	// Storage
	RootPath string `gorm:"size:255;not null"`

	// Passive Mode
	PassivePortStart int
	PassivePortEnd   int

	// Limits
	MaxConnections int `gorm:"default:100"`

	// Status
	Status string `gorm:"size:20;default:ACTIVE"`
	// ACTIVE
	// DISABLED

	Description string `gorm:"type:text"`
}
