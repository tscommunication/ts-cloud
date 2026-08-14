package repositories

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestAgentBillingQueriesAreScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Invoice{}, &models.Payment{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db

	agentA, agentB := uint(10), uint(20)
	customerA := models.Customer{CustomerCode: "CUS-A", FullName: "A", Mobile: "01001", AgentID: &agentA}
	customerB := models.Customer{CustomerCode: "CUS-B", FullName: "B", Mobile: "01002", AgentID: &agentB}
	if err := db.Create(&customerA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&customerB).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	invoiceA := models.Invoice{InvoiceNo: "INV-A", CustomerID: customerA.ID, SubscriptionID: 1, PackageID: 1, IssueDate: now, DueDate: now, TotalAmount: 100, DueAmount: 100}
	invoiceB := models.Invoice{InvoiceNo: "INV-B", CustomerID: customerB.ID, SubscriptionID: 2, PackageID: 1, IssueDate: now, DueDate: now, TotalAmount: 200, DueAmount: 200}
	if err := db.Create(&invoiceA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&invoiceB).Error; err != nil {
		t.Fatal(err)
	}
	paymentA := models.Payment{ReceiptNo: "RCPT-A", InvoiceID: invoiceA.ID, CustomerID: customerA.ID, SubscriptionID: 1, PaymentDate: now, Amount: 100}
	paymentB := models.Payment{ReceiptNo: "RCPT-B", InvoiceID: invoiceB.ID, CustomerID: customerB.ID, SubscriptionID: 2, PaymentDate: now, Amount: 200}
	if err := db.Create(&paymentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&paymentB).Error; err != nil {
		t.Fatal(err)
	}

	invoices, err := GetInvoicesByAgent(agentA)
	if err != nil || len(invoices) != 1 || invoices[0].ID != invoiceA.ID {
		t.Fatalf("invoice scope failed: rows=%v err=%v", invoices, err)
	}
	payments, err := GetPaymentsByAgent(agentA)
	if err != nil || len(payments) != 1 || payments[0].ID != paymentA.ID {
		t.Fatalf("payment scope failed: rows=%v err=%v", payments, err)
	}
	allowed, err := InvoiceBelongsToAgent(invoiceB.ID, agentA)
	if err != nil || allowed {
		t.Fatalf("foreign invoice allowed: allowed=%v err=%v", allowed, err)
	}
	allowed, err = PaymentBelongsToAgent(paymentB.ID, agentA)
	if err != nil || allowed {
		t.Fatalf("foreign payment allowed: allowed=%v err=%v", allowed, err)
	}
}
