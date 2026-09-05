package models

import "gorm.io/gorm"

// PackageServicePolicy defines which optional services an Internet package
// grants and the default limits used when customer service entitlements are
// created from that package.
//
// Package remains Internet-focused. Service-specific capabilities live here
// so FTP, Jellyfin, IPTV, Cloud Storage and future services can evolve without
// adding dedicated columns to the packages table.
type PackageServicePolicy struct {
	gorm.Model

	PackageID uint    `gorm:"not null;uniqueIndex:idx_package_service_policy" json:"package_id"`
	Package   Package `gorm:"foreignKey:PackageID" json:"-"`

	ServiceType string `gorm:"size:30;not null;uniqueIndex:idx_package_service_policy" json:"service_type"`

	Enabled bool `gorm:"not null;default:false" json:"enabled"`

	// QuotaGB is a service policy value. Its exact meaning is interpreted by
	// the service implementation. FTP and Cloud Storage use it as storage GB.
	QuotaGB int `gorm:"not null;default:0" json:"quota_gb"`

	// ConfigJSON is intentionally generic for service-specific options that
	// should not require another packages-table schema change.
	ConfigJSON string `gorm:"type:text" json:"config_json"`

	Remarks string `gorm:"type:text" json:"remarks"`
}
