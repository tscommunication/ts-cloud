package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func setupPaymentTest(t *testing.T) (*models.Invoice, *models.Payment) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Agent{}, &models.Invoice{}, &models.Payment{}, &models.AgentCollection{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
	customer := &models.Customer{CustomerCode: "CUS-TEST", FullName: "Test Customer", Mobile: "01000", Status: "ACTIVE"}
	if err := db.Create(customer).Error; err != nil {
		t.Fatal(err)
	}

	invoice := &models.Invoice{
		InvoiceNo: "INV-TEST", TotalAmount: 500, PaidAmount: 200,
		DueAmount: 300, Status: "PARTIAL", IssueDate: time.Now(), DueDate: time.Now(), CustomerID: customer.ID,
	}
	if err := db.Create(invoice).Error; err != nil {
		t.Fatal(err)
	}
	payment := &models.Payment{
		InvoiceID: invoice.ID, CustomerID: customer.ID, PaymentDate: time.Now(), Amount: 200,
		Method: "CASH", Status: "SUCCESS", ReceiptNo: "RCPT-TEST",
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatal(err)
	}
	return invoice, payment
}

func TestPaymentUpdateCreatesAgentCollectionAndVoidPreservesIt(t *testing.T) {
	_, payment := setupPaymentTest(t)
	agent := &models.Agent{Code: "AG-TEST", Name: "Test Agent", POPID: 1, CommissionPercent: 10, Status: "ACTIVE"}
	if err := database.DB.Create(agent).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Model(&models.Customer{}).Where("id = ?", payment.CustomerID).Update("agent_id", agent.ID).Error; err != nil {
		t.Fatal(err)
	}
	payment.Amount = 250
	if err := UpdatePayment(payment); err != nil {
		t.Fatal(err)
	}
	var collection models.AgentCollection
	if err := database.DB.Where("payment_id = ?", payment.ID).First(&collection).Error; err != nil {
		t.Fatal(err)
	}
	if collection.Amount != 250 || collection.CommissionRate != 10 || collection.CommissionAmount != 25 || collection.Status != "ACTIVE" {
		t.Fatalf("unexpected collection: %+v", collection)
	}
	if err := VoidPayment(payment.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.First(&collection, collection.ID).Error; err != nil {
		t.Fatal(err)
	}
	if collection.Status != "VOID" || collection.Amount != 250 {
		t.Fatalf("unexpected void collection: %+v", collection)
	}
	_, summary, err := repositories.ListAgentCollections(agent.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 0 || summary.TotalAmount != 0 || summary.TotalCommission != 0 || summary.VoidCount != 1 || summary.VoidAmount != 250 {
		t.Fatalf("void collection leaked into active totals: %+v", summary)
	}
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
