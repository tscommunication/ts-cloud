package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupSubscriptionDateAdjustmentTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.SubscriptionDateAdjustment{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func TestAdjustSubscriptionDateWithoutBillingExtendsExpiredSubscription(
	t *testing.T,
) {
	db := setupSubscriptionDateAdjustmentTestDB(t)

	now := time.Date(
		2026,
		time.August,
		17,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-MANUAL-001",
		CustomerID:       1,
		PackageID:        1,
		ActivationDate:   now.AddDate(0, -1, 0),
		BillingDay:       17,
		NextBillingDate:  now,
		ExpiryDate:       now,
		Status:           "EXPIRED",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	newDate := now.AddDate(0, 0, 1)

	result, err := AdjustSubscriptionDateWithoutBilling(
		&subscription,
		newDate,
		"One day goodwill extension",
		99,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Subscription.Status != "ACTIVE" {
		t.Fatalf(
			"status = %q, want ACTIVE",
			result.Subscription.Status,
		)
	}

	if !result.Subscription.ExpiryDate.Equal(
		startOfDay(newDate),
	) {
		t.Fatalf(
			"expiry = %v, want %v",
			result.Subscription.ExpiryDate,
			startOfDay(newDate),
		)
	}

	if !result.Subscription.NextBillingDate.Equal(
		startOfDay(newDate),
	) {
		t.Fatal("next billing date was not aligned")
	}

	if !result.Audit.WithoutBilling {
		t.Fatal("manual adjustment must be without billing")
	}

	if result.Audit.AdjustedByUserID != 99 {
		t.Fatal("adjusted by user was not recorded")
	}
}

func TestAdjustSubscriptionDateWithoutBillingAllowsDateReduction(
	t *testing.T,
) {
	db := setupSubscriptionDateAdjustmentTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-MANUAL-002",
		CustomerID:       1,
		PackageID:        1,
		ActivationDate:   now.AddDate(0, -1, 0),
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 0, 10),
		ExpiryDate:       now.AddDate(0, 0, 10),
		Status:           "ACTIVE",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	newDate := now.AddDate(0, 0, 5)

	result, err := AdjustSubscriptionDateWithoutBilling(
		&subscription,
		newDate,
		"Correction after duplicate recharge",
		99,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Subscription.ExpiryDate.Equal(
		startOfDay(newDate),
	) {
		t.Fatal("expiry date was not reduced")
	}

	if result.Subscription.Status != "ACTIVE" {
		t.Fatal("active subscription status changed unexpectedly")
	}
}

func TestAdjustSubscriptionDateWithoutBillingRejectsDisconnected(
	t *testing.T,
) {
	setupSubscriptionDateAdjustmentTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := &models.Subscription{
		Status: "DISCONNECTED",
	}
	subscription.ID = 10

	_, err := AdjustSubscriptionDateWithoutBilling(
		subscription,
		now.AddDate(0, 0, 1),
		"should not reconnect",
		99,
		now,
	)
	if err == nil {
		t.Fatal("expected disconnected adjustment rejection")
	}
}

func TestAdjustSubscriptionDateWithoutBillingRequiresReason(
	t *testing.T,
) {
	setupSubscriptionDateAdjustmentTestDB(t)

	subscription := &models.Subscription{
		Status: "ACTIVE",
	}
	subscription.ID = 20

	_, err := AdjustSubscriptionDateWithoutBilling(
		subscription,
		time.Now(),
		"",
		99,
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected reason validation error")
	}
}

func TestAdjustSubscriptionDateWithoutBillingPastDateExpiresActiveSubscription(
	t *testing.T,
) {
	setupSubscriptionDateAdjustmentTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		0, 0, 0, 0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-DATE-PAST",
		CustomerID:       1,
		PackageID:        1,
		ActivationDate:   now.AddDate(0, -1, 0),
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 0, 5),
		ExpiryDate:       now.AddDate(0, 0, 5),
		Status:           "ACTIVE",
	}

	if err := database.DB.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	newExpiry := now.AddDate(0, 0, -1)

	result, err := AdjustSubscriptionDateWithoutBilling(
		&subscription,
		newExpiry,
		"Super Admin manual correction",
		99,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Subscription.Status != "EXPIRED" {
		t.Fatalf(
			"expected EXPIRED status, got %s",
			result.Subscription.Status,
		)
	}

	if !result.Subscription.ExpiryDate.Equal(newExpiry) {
		t.Fatalf(
			"expected expiry %v, got %v",
			newExpiry,
			result.Subscription.ExpiryDate,
		)
	}

	if !result.Subscription.NextBillingDate.Equal(newExpiry) {
		t.Fatalf(
			"expected next billing date %v, got %v",
			newExpiry,
			result.Subscription.NextBillingDate,
		)
	}

	if result.Audit.OldStatus != "ACTIVE" {
		t.Fatalf(
			"expected audit old status ACTIVE, got %s",
			result.Audit.OldStatus,
		)
	}

	if result.Audit.NewStatus != "EXPIRED" {
		t.Fatalf(
			"expected audit new status EXPIRED, got %s",
			result.Audit.NewStatus,
		)
	}

	if !result.Audit.WithoutBilling {
		t.Fatal("expected without_billing audit flag")
	}
}
