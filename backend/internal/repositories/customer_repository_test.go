package repositories

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListCustomersLifecycleViews(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_lifecycle_views?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&models.Customer{}, &models.CustomerInternetAccount{}, &models.NetworkRouterPPPoESession{}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	customers := []models.Customer{
		{CustomerCode: "CUS-ONLINE", FullName: "Online", Mobile: "01300000001", Status: "ACTIVE", CreatedAt: now.Add(-time.Hour)},
		{CustomerCode: "CUS-OFFLINE", FullName: "Offline", Mobile: "01300000002", Status: "ACTIVE", CreatedAt: now.AddDate(0, 0, -10)},
		{CustomerCode: "CUS-PENDING", FullName: "Pending", Mobile: "01300000003", Status: "ACTIVE", CreatedAt: now.Add(-time.Hour)},
	}
	for index := range customers {
		if err := db.Create(&customers[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	expired := now.Add(-time.Hour)
	accounts := []models.CustomerInternetAccount{
		{AccountCode: "NET-ONLINE", CustomerID: customers[0].ID, RouterID: 1, PPPoEUsername: "online-user", PPPoEPasswordEncrypted: "secret", Status: "ACTIVE"},
		{AccountCode: "NET-OFFLINE", CustomerID: customers[1].ID, RouterID: 1, PPPoEUsername: "offline-user", PPPoEPasswordEncrypted: "secret", Status: "EXPIRED", ExpiryDate: &expired},
	}
	for index := range accounts {
		if err := db.Create(&accounts[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.NetworkRouterPPPoESession{RouterID: 1, SessionKey: "online", Username: "online-user", Active: true, FirstSeenAt: now, LastSeenAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		view string
		code string
	}{
		{"ONLINE", "CUS-ONLINE"},
		{"OFFLINE", "CUS-OFFLINE"},
		{"EXPIRED", "CUS-OFFLINE"},
		{"PENDING", "CUS-PENDING"},
	}
	for _, test := range tests {
		rows, count, err := ListCustomers(CustomerListParams{View: test.view, Now: now, Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("view %s: %v", test.view, err)
		}
		if count != 1 || len(rows) != 1 || rows[0].CustomerCode != test.code {
			t.Fatalf("view %s: got count=%d rows=%#v", test.view, count, rows)
		}
	}

	recent, count, err := ListCustomers(CustomerListParams{View: "RECENT", Now: now, Page: 1, PageSize: 20})
	if err != nil || count != 2 || len(recent) != 2 {
		t.Fatalf("recent view: count=%d rows=%d err=%v", count, len(recent), err)
	}
}
