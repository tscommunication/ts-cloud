package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupSubscriptionExpiryWorkerTestDB(
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
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func seedExpiryWorkerSubscription(
	t *testing.T,
	db *gorm.DB,
	code string,
	status string,
	expiryDate time.Time,
) models.Subscription {
	t.Helper()

	subscription := models.Subscription{
		SubscriptionCode: code,
		CustomerID:       1,
		PackageID:        1,
		Status:           status,
		ActivationDate:   expiryDate.AddDate(0, -1, 0),
		NextBillingDate:  expiryDate,
		ExpiryDate:       expiryDate,
		BillingDay:       1,
		RouterID:         1,
		PPPoEUsername:    code + "-user",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	return subscription
}

func TestProcessSubscriptionExpiriesReconcilesExpiredSubscriptions(
	t *testing.T,
) {
	db := setupSubscriptionExpiryWorkerTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		15,
		0,
		0,
		0,
		time.UTC,
	)

	expired := seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-EXP-001",
		"ACTIVE",
		now.AddDate(0, 0, -1),
	)

	seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-TODAY-001",
		"ACTIVE",
		now,
	)

	calls := 0

	runner := func(
		subscription *models.Subscription,
		action SubscriptionLifecycleAction,
		keyMaterial string,
	) (SubscriptionLifecycleReconciliationResult, error) {
		calls++

		if subscription == nil {
			t.Fatal("subscription is nil")
		}

		if subscription.ID != expired.ID {
			t.Fatalf(
				"subscription id = %d, want %d",
				subscription.ID,
				expired.ID,
			)
		}

		if subscription.Status != "EXPIRED" {
			t.Fatalf(
				"status = %q, want EXPIRED",
				subscription.Status,
			)
		}

		if action != SubscriptionLifecycleExpire {
			t.Fatalf(
				"action = %q, want EXPIRE",
				action,
			)
		}

		if keyMaterial != lifecycleReconciliationTestKey {
			t.Fatal("credential key was not forwarded")
		}

		return SubscriptionLifecycleReconciliationResult{
			Action:                  action,
			SubscriptionID:          subscription.ID,
			Status:                  subscription.Status,
			ReconciliationAttempted: true,
			Reconciliation: PPPSecretReconciliationResult{
				Plan: PPPSecretReconciliationPlan{
					SubscriptionID: subscription.ID,
					Action:         PPPSecretActionDisable,
				},
				Execution: PPPSecretReconciliationExecution{
					Action:   PPPSecretActionDisable,
					Executed: true,
					SecretID: "*20",
				},
			},
		}, nil
	}

	processSubscriptionExpiries(
		now,
		lifecycleReconciliationTestKey,
		runner,
	)

	if calls != 1 {
		t.Fatalf(
			"runner calls = %d, want 1",
			calls,
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		expired.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "EXPIRED" {
		t.Fatalf(
			"stored status = %q, want EXPIRED",
			stored.Status,
		)
	}
}

func TestProcessSubscriptionExpiriesDoesNotReconcileNonExpiredSubscriptions(
	t *testing.T,
) {
	db := setupSubscriptionExpiryWorkerTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		15,
		0,
		0,
		0,
		time.UTC,
	)

	seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-ACTIVE-001",
		"ACTIVE",
		now,
	)

	seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-SUSPENDED-001",
		"SUSPENDED",
		now.AddDate(0, 0, -2),
	)

	calls := 0

	runner := func(
		subscription *models.Subscription,
		action SubscriptionLifecycleAction,
		keyMaterial string,
	) (SubscriptionLifecycleReconciliationResult, error) {
		calls++

		return SubscriptionLifecycleReconciliationResult{},
			nil
	}

	processSubscriptionExpiries(
		now,
		lifecycleReconciliationTestKey,
		runner,
	)

	if calls != 0 {
		t.Fatalf(
			"runner calls = %d, want 0",
			calls,
		)
	}
}

func TestProcessSubscriptionExpiriesNilRunnerKeepsExpiredState(
	t *testing.T,
) {
	db := setupSubscriptionExpiryWorkerTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		15,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-NIL-RUNNER",
		"ACTIVE",
		now.AddDate(0, 0, -1),
	)

	processSubscriptionExpiries(
		now,
		lifecycleReconciliationTestKey,
		nil,
	)

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "EXPIRED" {
		t.Fatalf(
			"stored status = %q, want EXPIRED",
			stored.Status,
		)
	}
}

func TestProcessSubscriptionExpiriesReconciliationFailureDoesNotRollbackExpiry(
	t *testing.T,
) {
	db := setupSubscriptionExpiryWorkerTestDB(t)

	now := time.Date(
		2026,
		time.August,
		19,
		15,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := seedExpiryWorkerSubscription(
		t,
		db,
		"SUB-RECON-FAIL",
		"ACTIVE",
		now.AddDate(0, 0, -1),
	)

	calls := 0

	runner := func(
		subscription *models.Subscription,
		action SubscriptionLifecycleAction,
		keyMaterial string,
	) (SubscriptionLifecycleReconciliationResult, error) {
		calls++

		return SubscriptionLifecycleReconciliationResult{
			Action:                  action,
			SubscriptionID:          subscription.ID,
			Status:                  subscription.Status,
			ReconciliationAttempted: true,
			ReconciliationError:     "router unavailable",
		}, nil
	}

	processSubscriptionExpiries(
		now,
		lifecycleReconciliationTestKey,
		runner,
	)

	if calls != 1 {
		t.Fatalf(
			"runner calls = %d, want 1",
			calls,
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "EXPIRED" {
		t.Fatalf(
			"stored status = %q, want EXPIRED",
			stored.Status,
		)
	}
}
