package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestListNetworkDeviceSLAAtCarriesStatusAcrossPeriodBoundary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkDevice{}, &models.NetworkDeviceHealth{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	device := models.NetworkDevice{Code: "SLA-OLT-001", Name: "SLA OLT", DeviceType: "OLT", MonitoringEnabled: true, MonitoringStatus: "ONLINE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-24 * time.Hour)
	observations := []models.NetworkDeviceHealth{
		{NetworkDeviceID: device.ID, ObservedAt: cutoff.Add(-time.Hour), Status: "ONLINE"},
		{NetworkDeviceID: device.ID, ObservedAt: cutoff.Add(6 * time.Hour), Status: "OFFLINE"},
		{NetworkDeviceID: device.ID, ObservedAt: cutoff.Add(12 * time.Hour), Status: "ONLINE"},
	}
	if err := db.Create(&observations).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := listNetworkDeviceSLAAt(1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ObservedSeconds != 24*60*60 || row.DowntimeSeconds != 6*60*60 || row.UptimePercent != 75 {
		t.Fatalf("unexpected SLA row: %+v", row)
	}
}

func TestListNetworkDeviceSLAAtDoesNotAssumeStatusWithoutObservation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkDevice{}, &models.NetworkDeviceHealth{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	device := models.NetworkDevice{Code: "SLA-SW-001", Name: "SLA Switch", DeviceType: "SWITCH", MonitoringEnabled: true, MonitoringStatus: "ONLINE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := listNetworkDeviceSLAAt(1, time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ObservedSeconds != 0 || rows[0].UptimePercent != 0 {
		t.Fatalf("unexpected SLA without observations: %+v", rows)
	}
}
