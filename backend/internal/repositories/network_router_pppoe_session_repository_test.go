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
	if err := db.AutoMigrate(&models.NetworkRouter{}, &models.Customer{}, &models.Package{}, &models.Subscription{}, &models.NetworkRouterPPPoESession{}); err != nil {
		t.Fatal(err)
	}
	router := models.NetworkRouter{Code: "R-PPPOE", Name: "Test", Host: "10.0.0.1", APIPort: 8729, APIUsername: "reader"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "CUS-PPPOE", FullName: "Mapped Customer", Mobile: "01000000000"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	packageRow := models.Package{PackageCode: "PKG-PPPOE", Name: "Mapped Package"}
	if err := db.Create(&packageRow).Error; err != nil {
		t.Fatal(err)
	}
	subscription := models.Subscription{SubscriptionCode: "SUB-PPPOE", CustomerID: customer.ID, PackageID: packageRow.ID, RouterID: router.ID, PPPoEUsername: "USER-1"}
	if err := db.Create(&subscription).Error; err != nil {
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
	summary, err := GetNetworkPPPoESummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary.ActiveSessions != 2 || summary.MappedSessions != 1 || summary.UnmappedSessions != 1 {
		t.Fatalf("unexpected PPPoE summary: %+v", summary)
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
	if active[0].SubscriptionID == nil || active[0].CustomerCode != "CUS-PPPOE" || active[0].PackageCode != "PKG-PPPOE" {
		t.Fatalf("expected customer and package mapping, got %#v", active[0])
	}
	global, err := ListNetworkPPPoESessions(true, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(global) != 1 || global[0].RouterCode != "R-PPPOE" || global[0].CustomerCode != "CUS-PPPOE" {
		t.Fatalf("unexpected global PPPoE sessions: %#v", global)
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
