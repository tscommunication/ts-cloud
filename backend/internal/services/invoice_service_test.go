package services

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupInvoiceTest(t *testing.T) *models.Subscription {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.Payment{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	customer := models.Customer{CustomerCode: "CUS-INVOICE", FullName: "Invoice Customer", Status: "ACTIVE"}
	pkg := models.Package{PackageCode: "PKG-INVOICE", Name: "Invoice Package", Price: 800, Status: "ACTIVE"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	subscription := &models.Subscription{SubscriptionCode: "SUB-INVOICE", CustomerID: customer.ID, PackageID: pkg.ID, Status: "ACTIVE", ActivationDate: now, NextBillingDate: now, ExpiryDate: now.AddDate(0, 1, 0), BillingDay: 1}
	if err := db.Create(subscription).Error; err != nil {
		t.Fatal(err)
	}
	return subscription
}

func TestCreateInvoiceUsesPackagePriceAndRejectsDuplicatePeriod(t *testing.T) {
	subscription := setupInvoiceTest(t)
	now := time.Now()
	invoice := &models.Invoice{SubscriptionID: subscription.ID, BillMonth: 8, BillYear: 2026, IssueDate: now, DueDate: now.AddDate(0, 0, 7), PackagePrice: 1, Discount: 50, Vat: 10}
	if err := CreateInvoice(invoice); err != nil {
		t.Fatal(err)
	}
	if invoice.PackagePrice != 800 || invoice.TotalAmount != 760 || invoice.InvoiceNo == "" {
		t.Fatalf("unexpected invoice values: %+v", invoice)
	}
	duplicate := &models.Invoice{SubscriptionID: subscription.ID, BillMonth: 8, BillYear: 2026, IssueDate: now, DueDate: now.AddDate(0, 0, 7), PackagePrice: 800}
	if err := CreateInvoice(duplicate); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate period error, got %v", err)
	}
}

func TestCancelInvoicePreservesRecord(t *testing.T) {
	subscription := setupInvoiceTest(t)
	now := time.Now()
	invoice := &models.Invoice{SubscriptionID: subscription.ID, BillMonth: 9, BillYear: 2026, IssueDate: now, DueDate: now.AddDate(0, 0, 7), PackagePrice: 800}
	if err := CreateInvoice(invoice); err != nil {
		t.Fatal(err)
	}
	if err := CancelInvoice(invoice); err != nil {
		t.Fatal(err)
	}
	var saved models.Invoice
	if err := database.DB.First(&saved, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "CANCELLED" || saved.DueAmount != 0 {
		t.Fatalf("unexpected cancelled invoice: %+v", saved)
	}
}

func TestProcessInvoiceOverdues(t *testing.T) {
	subscription := setupInvoiceTest(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	invoice := &models.Invoice{SubscriptionID: subscription.ID, BillMonth: 7, BillYear: 2026, IssueDate: now.AddDate(0, 0, -10), DueDate: now.AddDate(0, 0, -1), PackagePrice: 800}
	if err := CreateInvoice(invoice); err != nil {
		t.Fatal(err)
	}
	count, err := ProcessInvoiceOverdues(now)
	if err != nil || count != 1 {
		t.Fatalf("expected one overdue invoice, count=%d err=%v", count, err)
	}
	if err := database.DB.First(invoice, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if invoice.Status != "OVERDUE" {
		t.Fatalf("expected OVERDUE, got %s", invoice.Status)
	}
}
