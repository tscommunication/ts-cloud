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
	if err := db.AutoMigrate(&models.NetworkRouter{}, &models.POP{}, &models.Agent{}, &models.Customer{}, &models.Package{}, &models.Subscription{}, &models.NetworkRouterPPPoESession{}, &models.NetworkRouterPPPoEDailyUsage{}); err != nil {
		t.Fatal(err)
	}
	router := models.NetworkRouter{Code: "R-PPPOE", Name: "Test", Host: "10.0.0.1", APIPort: 8729, APIUsername: "reader"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	pop := models.POP{Code: "POP-PPPOE", Name: "Test POP"}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}
	agent := models.Agent{Code: "AGENT-PPPOE", Name: "Mapped Agent", POPID: pop.ID}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}
	customer := models.Customer{CustomerCode: "CUS-PPPOE", FullName: "Mapped Customer", Mobile: "01000000000", AgentID: &agent.ID}
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
		{SessionKey: "one", Username: "user-1", Address: "10.1.0.1", RxRateBps: 1_500_000, TxRateBps: 750_000, RxBytes: 1000, TxBytes: 2000},
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
	second := []models.NetworkRouterPPPoESession{{SessionKey: "one", Username: "user-1", Address: "10.1.0.1", Uptime: "2m", RxRateBps: 2_500_000, TxRateBps: 1_250_000, RxBytes: 3000, TxBytes: 4000}}
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
	if active[0].RxRateBps != 266 || active[0].TxRateBps != 266 || active[0].RxBytes != 3000 || active[0].TxBytes != 4000 {
		t.Fatalf("session traffic was not updated: %#v", active[0])
	}
	var usage models.NetworkRouterPPPoEDailyUsage
	if err := db.Where("router_id = ? AND session_key = ?", router.ID, "one").First(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if usage.RxBytes != 2000 || usage.TxBytes != 2000 {
		t.Fatalf("unexpected daily usage: %+v", usage)
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
	if global[0].AgentCode != "AGENT-PPPOE" || global[0].AgentName != "Mapped Agent" {
		t.Fatalf("expected assigned agent in PPPoE view, got %#v", global[0])
	}
	agentSessions, err := ListNetworkPPPoESessionsForAgent(agent.ID, true, 10)
	if err != nil || len(agentSessions) != 1 || agentSessions[0].Username != "user-1" {
		t.Fatalf("agent must only see own mapped session: %#v, %v", agentSessions, err)
	}
	agentSummary, err := GetNetworkPPPoESummaryForAgent(agent.ID)
	if err != nil || agentSummary.ActiveSessions != 1 || agentSummary.MappedSessions != 1 || agentSummary.UnmappedSessions != 0 {
		t.Fatalf("unexpected agent PPPoE summary: %+v, %v", agentSummary, err)
	}
	agentUsage, err := GetNetworkPPPoEDailyUsageSummaryForAgent(agent.ID, 1, started.Add(time.Minute))
	if err != nil || agentUsage.RxBytes != 2000 || agentUsage.TxBytes != 2000 {
		t.Fatalf("unexpected agent PPPoE usage: %+v, %v", agentUsage, err)
	}
	agentUserUsage, err := ListNetworkPPPoEUserUsageForAgent(agent.ID, 1, 10, started.Add(time.Minute))
	if err != nil || len(agentUserUsage) != 1 || agentUserUsage[0].Username != "user-1" || agentUserUsage[0].RxBytes != 2000 || agentUserUsage[0].TxBytes != 2000 {
		t.Fatalf("unexpected agent PPPoE user usage: %#v, %v", agentUserUsage, err)
	}
	allowed, err := NetworkPPPoESessionBelongsToAgent(agentSessions[0].ID, agent.ID)
	if err != nil || !allowed {
		t.Fatalf("agent should be allowed to view own live traffic: %v", err)
	}
	allowed, err = NetworkPPPoESessionBelongsToAgent(agentSessions[0].ID, agent.ID+1)
	if err != nil || allowed {
		t.Fatalf("another agent must not be allowed to view live traffic: %v", err)
	}
	all, err := ListNetworkRouterPPPoESessions(router.ID, false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 session records, got %d", len(all))
	}
	for _, row := range all {
		if row.Username == "user-2" && (row.Active || row.DisconnectedAt == nil || row.DisconnectReason != "NOT_OBSERVED_ON_SYNC") {
			t.Fatalf("disconnected session was not closed: %#v", row)
		}
	}
}
