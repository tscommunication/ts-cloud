package repositories

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionRenewalReversalRepositoryTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.SubscriptionRenewal{},
		&models.SubscriptionRenewalReversal{},
	); err != nil {
		t.Fatal(err)
	}

	database.DB = db

	return db
}

func TestSubscriptionRenewalReversalPaymentIDIsUnique(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalRepositoryTestDB(t)

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	first := &models.SubscriptionRenewalReversal{
		RenewalID:               101,
		PaymentID:               201,
		InvoiceID:               301,
		CustomerID:              401,
		SubscriptionID:          501,
		PreviousExpiryDate:      now.AddDate(0, 1, 0),
		RestoredExpiryDate:      now,
		PreviousNextBillingDate: now.AddDate(0, 1, 0),
		RestoredNextBillingDate: now,
		Reason:                  "payment void",
		ReversedByUserID:        1,
		ReversedAt:              now,
	}

	if err := db.Create(first).Error; err != nil {
		t.Fatal(err)
	}

	second := *first
	second.ID = 0
	second.RenewalID = 102

	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate payment reversal to fail")
	}
}

func TestSubscriptionRenewalReversalRenewalIDIsUnique(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalRepositoryTestDB(t)

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	first := &models.SubscriptionRenewalReversal{
		RenewalID:               101,
		PaymentID:               201,
		InvoiceID:               301,
		CustomerID:              401,
		SubscriptionID:          501,
		PreviousExpiryDate:      now.AddDate(0, 1, 0),
		RestoredExpiryDate:      now,
		PreviousNextBillingDate: now.AddDate(0, 1, 0),
		RestoredNextBillingDate: now,
		Reason:                  "payment void",
		ReversedByUserID:        1,
		ReversedAt:              now,
	}

	if err := db.Create(first).Error; err != nil {
		t.Fatal(err)
	}

	second := *first
	second.ID = 0
	second.PaymentID = 202

	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate renewal reversal to fail")
	}
}

func TestSubscriptionRenewalReversalExistsByPaymentIDTx(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalRepositoryTestDB(t)

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	reversal := &models.SubscriptionRenewalReversal{
		RenewalID:               101,
		PaymentID:               201,
		InvoiceID:               301,
		CustomerID:              401,
		SubscriptionID:          501,
		PreviousExpiryDate:      now.AddDate(0, 1, 0),
		RestoredExpiryDate:      now,
		PreviousNextBillingDate: now.AddDate(0, 1, 0),
		RestoredNextBillingDate: now,
		Reason:                  "payment void",
		ReversedByUserID:        1,
		ReversedAt:              now,
	}

	if err := CreateSubscriptionRenewalReversalTx(
		db,
		reversal,
	); err != nil {
		t.Fatal(err)
	}

	exists, err := SubscriptionRenewalReversalExistsByPaymentIDTx(
		db,
		reversal.PaymentID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("expected reversal to exist")
	}

	missing, err := SubscriptionRenewalReversalExistsByPaymentIDTx(
		db,
		999999,
	)
	if err != nil {
		t.Fatal(err)
	}

	if missing {
		t.Fatal("unexpected reversal for missing payment")
	}
}
