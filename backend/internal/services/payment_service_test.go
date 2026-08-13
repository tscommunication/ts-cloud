package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupPaymentTest(t *testing.T) (*models.Invoice, *models.Payment) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Invoice{}, &models.Payment{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db

	invoice := &models.Invoice{
		InvoiceNo: "INV-TEST", TotalAmount: 500, PaidAmount: 200,
		DueAmount: 300, Status: "PARTIAL", IssueDate: time.Now(), DueDate: time.Now(),
	}
	if err := db.Create(invoice).Error; err != nil {
		t.Fatal(err)
	}
	payment := &models.Payment{
		InvoiceID: invoice.ID, PaymentDate: time.Now(), Amount: 200,
		Method: "CASH", Status: "SUCCESS", ReceiptNo: "RCPT-TEST",
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatal(err)
	}
	return invoice, payment
}

func TestUpdatePaymentReconcilesInvoice(t *testing.T) {
	invoice, payment := setupPaymentTest(t)
	payment.Amount = 350
	payment.Method = "cash"

	if err := UpdatePayment(payment); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.First(invoice, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if invoice.PaidAmount != 350 || invoice.DueAmount != 150 || invoice.Status != "PARTIAL" {
		t.Fatalf("unexpected invoice after update: %+v", invoice)
	}
}

func TestUpdatePaymentRejectsOverpayment(t *testing.T) {
	_, payment := setupPaymentTest(t)
	payment.Amount = 501

	if err := UpdatePayment(payment); err == nil {
		t.Fatal("expected overpayment error")
	}
}

func TestVoidPaymentReconcilesInvoiceAndPreservesRecord(t *testing.T) {
	invoice, payment := setupPaymentTest(t)

	if err := VoidPayment(payment.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.First(invoice, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if invoice.PaidAmount != 0 || invoice.DueAmount != 500 || invoice.Status != "UNPAID" {
		t.Fatalf("unexpected invoice after delete: %+v", invoice)
	}
	if err := database.DB.First(payment, payment.ID).Error; err != nil {
		t.Fatal(err)
	}
	if payment.Status != "VOID" {
		t.Fatalf("expected VOID payment, got %s", payment.Status)
	}
}
