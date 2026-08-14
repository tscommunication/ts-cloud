package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRouterHealthHistoryRecordsChangesAndFiveMinuteSamples(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkRouterHealth{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	router := &models.NetworkRouter{ConnectivityStatus: "ONLINE", APIStatus: "AUTHENTICATED", CPULoad: 20}
	router.ID = 1
	started := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	if err := recordNetworkRouterHealth(router, started); err != nil {
		t.Fatal(err)
	}
	if err := recordNetworkRouterHealth(router, started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	router.APIStatus = "AUTH_FAILED"
	router.LastAPIError = "invalid credentials"
	if err := recordNetworkRouterHealth(router, started.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := recordNetworkRouterHealth(router, started.Add(8*time.Minute)); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&models.NetworkRouterHealth{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3 snapshots, got %d", count)
	}
}

func TestRouterResourceAlertsOpenUpdateAndResolve(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkRouterAlert{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	router := &models.NetworkRouter{Code: "R1", CPULoad: 90, TotalMemory: 1000, FreeMemory: 50}
	router.ID = 1
	started := time.Date(2026, 8, 14, 11, 0, 0, 0, time.UTC)
	if err := evaluateNetworkRouterAlerts(router, 85, 90, started); err != nil {
		t.Fatal(err)
	}
	router.CPULoad = 95
	if err := evaluateNetworkRouterAlerts(router, 85, 90, started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var activeCount int64
	if err := db.Model(&models.NetworkRouterAlert{}).Where("status = ?", "ACTIVE").Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != 2 {
		t.Fatalf("expected 2 active alerts without duplicates, got %d", activeCount)
	}
	router.CPULoad = 20
	router.FreeMemory = 800
	if err := evaluateNetworkRouterAlerts(router, 85, 90, started.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.NetworkRouterAlert{}).Where("status = ?", "ACTIVE").Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != 0 {
		t.Fatalf("expected alerts to resolve, got %d active", activeCount)
	}
}

func TestRouterStateAlertsTransitionWithoutDuplicates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NetworkRouterAlert{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	router := &models.NetworkRouter{Code: "R1", ConnectivityStatus: "OFFLINE", APIStatus: "AUTH_FAILED"}
	router.ID = 1
	started := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := evaluateNetworkRouterStateAlerts(router, started); err != nil {
		t.Fatal(err)
	}
	if err := evaluateNetworkRouterStateAlerts(router, started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	router.ConnectivityStatus = "ONLINE"
	if err := evaluateNetworkRouterStateAlerts(router, started.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	router.APIStatus = "AUTHENTICATED"
	if err := evaluateNetworkRouterStateAlerts(router, started.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}

	var total, active int64
	if err := db.Model(&models.NetworkRouterAlert{}).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.NetworkRouterAlert{}).Where("status = ?", "ACTIVE").Count(&active).Error; err != nil {
		t.Fatal(err)
	}
	if total != 2 || active != 0 {
		t.Fatalf("expected two resolved state alerts, got total=%d active=%d", total, active)
	}
}
