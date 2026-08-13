package repositories

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestExpireOverdueSubscriptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 13, 15, 0, 0, 0, time.UTC)
	subscriptions := []models.Subscription{
		{SubscriptionCode: "SUB-OLD", CustomerID: 1, PackageID: 1, Status: "ACTIVE", ActivationDate: now.AddDate(0, -2, 0), NextBillingDate: now.AddDate(0, -1, 0), ExpiryDate: now.AddDate(0, 0, -1), BillingDay: 1},
		{SubscriptionCode: "SUB-TODAY", CustomerID: 1, PackageID: 1, Status: "ACTIVE", ActivationDate: now.AddDate(0, -1, 0), NextBillingDate: now, ExpiryDate: now, BillingDay: 1},
		{SubscriptionCode: "SUB-SUSPENDED", CustomerID: 1, PackageID: 1, Status: "SUSPENDED", ActivationDate: now.AddDate(0, -2, 0), NextBillingDate: now.AddDate(0, -1, 0), ExpiryDate: now.AddDate(0, 0, -1), BillingDay: 1},
	}
	for index := range subscriptions {
		if err := db.Create(&subscriptions[index]).Error; err != nil {
			t.Fatal(err)
		}
	}

	count, err := ExpireOverdueSubscriptions(now)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 expired subscription, got %d", count)
	}

	var statuses []string
	if err := db.Model(&models.Subscription{}).Order("id ASC").Pluck("status", &statuses).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{"EXPIRED", "ACTIVE", "SUSPENDED"}
	for index := range want {
		if statuses[index] != want[index] {
			t.Fatalf("status %d: expected %s, got %s", index, want[index], statuses[index])
		}
	}
}
