package repositories

import (
	"fmt"
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionRenewalRepositoryTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			fmt.Sprintf(
				"file:subscription-renewal-%d?mode=memory&cache=shared",
				time.Now().UnixNano(),
			),
		),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.Invoice{},
		&models.Payment{},
		&models.SubscriptionRenewal{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	database.DB = db
	return db
}

func TestSubscriptionRenewalPaymentIDIsUnique(t *testing.T) {
	db := setupSubscriptionRenewalRepositoryTestDB(t)

	now := time.Now()

	first := models.SubscriptionRenewal{
		PaymentID:          101,
		InvoiceID:          201,
		CustomerID:         301,
		SubscriptionID:     401,
		OldExpiryDate:      now,
		NewExpiryDate:      now.AddDate(0, 1, 0),
		OldNextBillingDate: now,
		NewNextBillingDate: now.AddDate(0, 1, 0),
		RenewalDate:        now,
		Amount:             500,
		Source:             "PAYMENT",
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first renewal: %v", err)
	}

	second := first
	second.Model = gorm.Model{}

	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate payment renewal to be rejected")
	}
}

func TestSubscriptionRenewalExistsByPaymentIDTx(t *testing.T) {
	db := setupSubscriptionRenewalRepositoryTestDB(t)

	now := time.Now()

	renewal := models.SubscriptionRenewal{
		PaymentID:          111,
		InvoiceID:          211,
		CustomerID:         311,
		SubscriptionID:     411,
		OldExpiryDate:      now,
		NewExpiryDate:      now.AddDate(0, 1, 0),
		OldNextBillingDate: now,
		NewNextBillingDate: now.AddDate(0, 1, 0),
		RenewalDate:        now,
		Amount:             600,
		Source:             "PAYMENT",
	}

	if err := db.Create(&renewal).Error; err != nil {
		t.Fatalf("create renewal: %v", err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		exists, err := SubscriptionRenewalExistsByPaymentIDTx(
			tx,
			renewal.PaymentID,
		)
		if err != nil {
			return err
		}
		if !exists {
			t.Fatal("expected renewal to exist")
		}

		missing, err := SubscriptionRenewalExistsByPaymentIDTx(tx, 999999)
		if err != nil {
			return err
		}
		if missing {
			t.Fatal("unexpected renewal for missing payment")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}
}
