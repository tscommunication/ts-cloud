package models

import "time"

type CustomerImportBatch struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Filename        string    `gorm:"size:255;not null" json:"filename"`
	Status          string    `gorm:"size:30;not null;index" json:"status"`
	RouterID        uint      `gorm:"not null;index" json:"router_id"`
	TotalRows       int       `gorm:"not null" json:"total_rows"`
	ImportedRows    int       `gorm:"not null" json:"imported_rows"`
	CreatedPackages int       `gorm:"not null" json:"created_packages"`
	CreatedPOPs     int       `gorm:"not null" json:"created_pops"`
	CreatedAt       time.Time `gorm:"not null" json:"created_at"`
}

type CustomerImportItem struct {
	ID             uint   `gorm:"primaryKey"`
	BatchID        uint   `gorm:"not null;index"`
	SourceID       string `gorm:"size:100;not null"`
	Username       string `gorm:"size:255;not null;index"`
	CustomerID     uint   `gorm:"not null;index"`
	SubscriptionID uint   `gorm:"not null;index"`
}
