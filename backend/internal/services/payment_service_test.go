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
	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Agent{},
		&models.Invoice{},
		&models.Payment{},
		&models.AgentCollection{},
		&models.PaymentVoidAudit{},
	); err != nil {
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
	if _, err := VoidPayment(
		payment.ID,
		"Duplicate recharge correction",
		99,
		time.Now(),
	); err != nil {
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

	if _, err := VoidPayment(
		payment.ID,
		"Duplicate recharge correction",
		99,
		time.Now(),
	); err != nil {
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

func TestVoidPaymentCreatesAudit(
	t *testing.T,
) {
	invoice, payment := setupPaymentTest(t)

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

	audit, err := VoidPayment(
		payment.ID,
		"Customer accidentally recharged twice",
		99,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	if audit == nil {
		t.Fatal("expected payment void audit")
	}

	if audit.PaymentID != payment.ID {
		t.Fatalf(
			"audit payment id = %d, want %d",
			audit.PaymentID,
			payment.ID,
		)
	}

	if audit.InvoiceID != invoice.ID {
		t.Fatalf(
			"audit invoice id = %d, want %d",
			audit.InvoiceID,
			invoice.ID,
		)
	}

	if audit.Reason !=
		"Customer accidentally recharged twice" {
		t.Fatalf(
			"audit reason = %q",
			audit.Reason,
		)
	}

	if audit.VoidedByUserID != 99 {
		t.Fatalf(
			"audit actor = %d, want 99",
			audit.VoidedByUserID,
		)
	}

	if !audit.VoidedAt.Equal(now) {
		t.Fatalf(
			"audit time = %v, want %v",
			audit.VoidedAt,
			now,
		)
	}

	if audit.PreviousStatus != "SUCCESS" ||
		audit.NewStatus != "VOID" {
		t.Fatalf(
			"unexpected audit status transition: %s -> %s",
			audit.PreviousStatus,
			audit.NewStatus,
		)
	}

	var saved models.PaymentVoidAudit

	if err := database.DB.
		Where("payment_id = ?", payment.ID).
		First(&saved).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Reason != audit.Reason {
		t.Fatalf(
			"saved reason = %q, want %q",
			saved.Reason,
			audit.Reason,
		)
	}
}

func TestVoidPaymentRequiresReason(
	t *testing.T,
) {
	_, payment := setupPaymentTest(t)

	if _, err := VoidPayment(
		payment.ID,
		"   ",
		99,
		time.Now(),
	); err == nil {
		t.Fatal("expected void reason validation error")
	}

	var saved models.Payment

	if err := database.DB.First(
		&saved,
		payment.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Status != "SUCCESS" {
		t.Fatalf(
			"payment changed despite missing reason: %s",
			saved.Status,
		)
	}
}

func TestVoidPaymentRequiresActor(
	t *testing.T,
) {
	_, payment := setupPaymentTest(t)

	if _, err := VoidPayment(
		payment.ID,
		"Duplicate recharge correction",
		0,
		time.Now(),
	); err == nil {
		t.Fatal("expected void actor validation error")
	}

	var saved models.Payment

	if err := database.DB.First(
		&saved,
		payment.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Status != "SUCCESS" {
		t.Fatalf(
			"payment changed despite missing actor: %s",
			saved.Status,
		)
	}
}

func TestVoidPaymentSecondAttemptPreservesOriginalAudit(
	t *testing.T,
) {
	_, payment := setupPaymentTest(t)

	firstTime := time.Date(
		2026,
		time.August,
		19,
		15,
		0,
		0,
		0,
		time.UTC,
	)

	if _, err := VoidPayment(
		payment.ID,
		"Duplicate recharge correction",
		99,
		firstTime,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := VoidPayment(
		payment.ID,
		"Second attempt",
		100,
		firstTime.Add(time.Hour),
	); err == nil {
		t.Fatal("expected already voided error")
	}

	var audits []models.PaymentVoidAudit

	if err := database.DB.
		Where("payment_id = ?", payment.ID).
		Find(&audits).Error; err != nil {
		t.Fatal(err)
	}

	if len(audits) != 1 {
		t.Fatalf(
			"audit count = %d, want 1",
			len(audits),
		)
	}

	if audits[0].VoidedByUserID != 99 ||
		audits[0].Reason !=
			"Duplicate recharge correction" {
		t.Fatalf(
			"original audit was altered: %+v",
			audits[0],
		)
	}
}
