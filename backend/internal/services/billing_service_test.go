package services

import (
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
