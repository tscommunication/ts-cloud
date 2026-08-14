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
