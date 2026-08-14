package repositories

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncNetworkRouterPPPoESessionsTracksDisconnects(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pppoe_session_repository?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&models.NetworkRouter{}, &models.NetworkRouterPPPoESession{}); err != nil {
		t.Fatal(err)
	}
	router := models.NetworkRouter{Code: "R-PPPOE", Name: "Test", Host: "10.0.0.1", APIPort: 8729, APIUsername: "reader"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	first := []models.NetworkRouterPPPoESession{
		{SessionKey: "one", Username: "user-1", Address: "10.1.0.1"},
		{SessionKey: "two", Username: "user-2", Address: "10.1.0.2"},
	}
	if err := SyncNetworkRouterPPPoESessions(router.ID, first, started); err != nil {
		t.Fatal(err)
	}
	second := []models.NetworkRouterPPPoESession{{SessionKey: "one", Username: "user-1", Address: "10.1.0.1", Uptime: "2m"}}
	if err := SyncNetworkRouterPPPoESessions(router.ID, second, started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	active, err := ListNetworkRouterPPPoESessions(router.ID, true, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Username != "user-1" || active[0].Uptime != "2m" {
		t.Fatalf("unexpected active sessions: %#v", active)
	}
	all, err := ListNetworkRouterPPPoESessions(router.ID, false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 session records, got %d", len(all))
	}
	for _, row := range all {
		if row.Username == "user-2" && (row.Active || row.DisconnectedAt == nil) {
			t.Fatalf("disconnected session was not closed: %#v", row)
		}
	}
}
