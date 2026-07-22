package models

import "gorm.io/gorm"

type Package struct {
	gorm.Model

	PackageCode string `gorm:"uniqueIndex;size:30"`
	Name        string `gorm:"size:100;not null"`

	Price float64

	DownloadSpeed int
	UploadSpeed   int

	BurstDownload int
	BurstUpload   int

	ValidityDays int

	MikroTikProfile string `gorm:"size:100"`
	RadiusProfile   string `gorm:"size:100"`

	Status string `gorm:"default:ACTIVE"`

	Description string `gorm:"type:text"`
}
