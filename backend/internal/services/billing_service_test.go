package services

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestRunDueBillingCreatesInvoiceAndAdvancesDate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.Payment{}, &models.BillingRun{}, &models.BillingRunItem{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	customer := models.Customer{CustomerCode: "CUS-BILL", FullName: "Billing Customer", Mobile: "01700000000", Status: "ACTIVE"}
	pkg := models.Package{PackageCode: "PKG-BILL", Name: "Billing Package", Price: 1200, Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	dueDate := now.AddDate(0, 0, -1)
	subscription := models.Subscription{SubscriptionCode: "SUB-BILL", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", ActivationDate: now.AddDate(0, -1, 0), NextBillingDate: dueDate, ExpiryDate: now.AddDate(0, 1, 0), BillingDay: 1}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	run, err := RunDueBilling(now, 9)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "COMPLETED" || run.CreatedCount != 1 || run.FailedCount != 0 {
		t.Fatalf("unexpected run: %+v", run)
	}
	var invoice models.Invoice
	if err := db.First(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	if invoice.TotalAmount != 1200 || invoice.SubscriptionID != subscription.ID {
		t.Fatalf("unexpected invoice: %+v", invoice)
	}
	if err := db.First(&subscription, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}
	wantNext := dueDate.AddDate(0, 1, 0)
	if !subscription.NextBillingDate.Equal(wantNext) {
		t.Fatalf("expected next billing %v, got %v", wantNext, subscription.NextBillingDate)
	}
}

func TestRunDueBillingClampsMonthEndBillingDate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.Payment{}, &models.BillingRun{}, &models.BillingRunItem{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db

	now := time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC)
	customer := models.Customer{CustomerCode: "CUS-BILL-END", FullName: "Month End Customer", Mobile: "01700000001", Status: "ACTIVE"}
	pkg := models.Package{PackageCode: "PKG-BILL-END", Name: "Month End Package", Price: 1200, Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	subscription := models.Subscription{SubscriptionCode: "SUB-BILL-END", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", ActivationDate: now.AddDate(0, -1, 0), NextBillingDate: now, ExpiryDate: now.AddDate(0, 1, 0), BillingDay: 31}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := RunDueBilling(now, 9); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&subscription, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.February, 28, 12, 0, 0, 0, time.UTC)
	if !subscription.NextBillingDate.Equal(want) {
		t.Fatalf("expected next billing %v, got %v", want, subscription.NextBillingDate)
	}

	februaryRun := time.Date(2026, time.February, 28, 12, 0, 0, 0, time.UTC)
	if _, err := RunDueBilling(februaryRun, 9); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&subscription, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC)
	if !subscription.NextBillingDate.Equal(want) {
		t.Fatalf("expected billing anchor to return on %v, got %v", want, subscription.NextBillingDate)
	}
}

func TestRunDueBillingMarksRunFailedWhenNextBillingDateCannotAdvance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.Payment{}, &models.BillingRun{}, &models.BillingRunItem{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db

	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	customer := models.Customer{CustomerCode: "CUS-BILL-UPDATE", FullName: "Billing Update Customer", Mobile: "01700000002", Status: "ACTIVE"}
	pkg := models.Package{PackageCode: "PKG-BILL-UPDATE", Name: "Billing Update Package", Price: 1200, Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	dueDate := now.AddDate(0, 0, -1)
	subscription := models.Subscription{SubscriptionCode: "SUB-BILL-UPDATE", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", ActivationDate: now.AddDate(0, -1, 0), NextBillingDate: dueDate, ExpiryDate: now.AddDate(0, 1, 0), BillingDay: dueDate.Day()}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	db.Callback().Update().Before("gorm:update").Register("fail_subscription_billing_advance", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "Subscription" {
			tx.AddError(errors.New("simulated subscription update failure"))
		}
	})

	run, err := RunDueBilling(now, 9)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "FAILED" || run.CreatedCount != 0 || run.FailedCount != 1 {
		t.Fatalf("unexpected run: %+v", run)
	}

	var invoice models.Invoice
	if err := db.First(&invoice).Error; err != nil {
		t.Fatalf("expected created invoice to remain auditable: %v", err)
	}
	var item models.BillingRunItem
	if err := db.First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Status != "FAILED" || item.InvoiceID == nil || *item.InvoiceID != invoice.ID {
		t.Fatalf("unexpected billing run item: %+v", item)
	}
}
