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
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.CustomerInternetAccount{}, &models.Subscription{}); err != nil {
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

	expiredSubscriptions, err := ExpireOverdueSubscriptions(now)
	if err != nil {
		t.Fatal(err)
	}

	if len(expiredSubscriptions) != 1 {
		t.Fatalf(
			"expected 1 expired subscription, got %d",
			len(expiredSubscriptions),
		)
	}

	if expiredSubscriptions[0].SubscriptionCode != "SUB-OLD" {
		t.Fatalf(
			"expired subscription = %q, want SUB-OLD",
			expiredSubscriptions[0].SubscriptionCode,
		)
	}

	if expiredSubscriptions[0].Status != "EXPIRED" {
		t.Fatalf(
			"returned expired subscription status = %q, want EXPIRED",
			expiredSubscriptions[0].Status,
		)
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

func TestUpdateSubscriptionPersistsChangedPackageWithOldPackagePreloaded(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}

	oldPackage := models.Package{PackageCode: "PKG-OLD", Name: "Old", Status: "ACTIVE"}
	newPackage := models.Package{PackageCode: "PKG-NEW", Name: "New", Status: "ACTIVE"}
	if err := db.Create(&oldPackage).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&newPackage).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)
	account := models.CustomerInternetAccount{
		AccountCode: "NET-PACKAGE-CHANGE", CustomerID: 1, RouterID: 1,
		PPPoEUsername: "package-change-user", PPPoEPasswordEncrypted: "encrypted",
		PackageID: oldPackage.ID, BillingDay: 21, Status: "ACTIVE",
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	created := models.Subscription{
		SubscriptionCode: "SUB-PACKAGE-CHANGE", CustomerID: 1,
		PackageID: oldPackage.ID, Status: "ACTIVE", ActivationDate: now,
		NextBillingDate: now.AddDate(0, 1, 0), ExpiryDate: now.AddDate(0, 1, 0),
		BillingDay: 21, InternetAccountID: &account.ID,
	}
	if err := db.Create(&created).Error; err != nil {
		t.Fatal(err)
	}

	loaded, err := GetSubscriptionByID(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Package.ID != oldPackage.ID {
		t.Fatalf("preloaded package = %d, want %d", loaded.Package.ID, oldPackage.ID)
	}

	loaded.PackageID = newPackage.ID
	if err := UpdateSubscription(loaded); err != nil {
		t.Fatal(err)
	}

	var saved models.Subscription
	if err := db.First(&saved, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.PackageID != newPackage.ID {
		t.Fatalf("saved package_id = %d, want %d", saved.PackageID, newPackage.ID)
	}
	if err := db.First(&account, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if account.PackageID != newPackage.ID {
		t.Fatalf("internet account package_id = %d, want %d", account.PackageID, newPackage.ID)
	}
}
