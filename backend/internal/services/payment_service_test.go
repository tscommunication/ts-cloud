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

func setupPaymentServiceTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open payment service test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.Invoice{},
		&models.Payment{},
		&models.AgentCollection{},
		&models.PaymentVoidAudit{},
		&models.SubscriptionRenewal{},
	); err != nil {
		t.Fatalf("migrate payment service test database: %v", err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
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

func TestCreatePaymentFullPaymentRenewsSubscriptionAtomically(
	t *testing.T,
) {
	db := setupPaymentServiceTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 0, 0, 0,
		time.UTC,
	)

	customer := models.Customer{
		CustomerCode: "CUS-E5D-1",
		FullName:     "E5D Customer",
		Mobile:       "01710000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-E5D-1",
		Name:        "E5D Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode: "SUB-E5D-1",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now.AddDate(0, -2, 0),
		NextBillingDate: time.Date(
			2026, time.August, 5,
			0, 0, 0, 0,
			time.UTC,
		),
		ExpiryDate: time.Date(
			2026, time.August, 5,
			0, 0, 0, 0,
			time.UTC,
		),
		BillingDay: 5,
		Status:     "EXPIRED",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{
		InvoiceNo:      "INV-E5D-1",
		SubscriptionID: subscription.ID,
		CustomerID:     customer.ID,
		PackageID:      pkg.ID,
		BillMonth:      int(now.Month()),
		BillYear:       now.Year(),
		IssueDate:      now,
		DueDate:        now,
		PackagePrice:   500,
		TotalAmount:    500,
		PaidAmount:     0,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	payment := &models.Payment{
		InvoiceID:   invoice.ID,
		PaymentDate: now,
		Amount:      500,
		Method:      "CASH",
		Status:      "SUCCESS",
	}

	if err := CreatePayment(payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	var savedInvoice models.Invoice
	if err := db.First(
		&savedInvoice,
		invoice.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if savedInvoice.Status != "PAID" {
		t.Fatalf(
			"invoice status = %q, want PAID",
			savedInvoice.Status,
		)
	}

	if savedInvoice.DueAmount != 0 {
		t.Fatalf(
			"invoice due = %.2f, want 0",
			savedInvoice.DueAmount,
		)
	}

	var savedSubscription models.Subscription
	if err := db.First(
		&savedSubscription,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	wantExpiry := time.Date(
		2026, time.September, 19,
		0, 0, 0, 0,
		time.UTC,
	)

	if !savedSubscription.ExpiryDate.Equal(
		wantExpiry,
	) {
		t.Fatalf(
			"expiry = %v, want %v",
			savedSubscription.ExpiryDate,
			wantExpiry,
		)
	}

	if savedSubscription.Status != "ACTIVE" {
		t.Fatalf(
			"subscription status = %q, want ACTIVE",
			savedSubscription.Status,
		)
	}

	var renewalCount int64
	if err := db.Model(
		&models.SubscriptionRenewal{},
	).Where(
		"payment_id = ?",
		payment.ID,
	).Count(
		&renewalCount,
	).Error; err != nil {
		t.Fatal(err)
	}

	if renewalCount != 1 {
		t.Fatalf(
			"renewal rows = %d, want 1",
			renewalCount,
		)
	}
}

func TestCreatePaymentPartialPaymentDoesNotRenewSubscription(
	t *testing.T,
) {
	db := setupPaymentServiceTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 0, 0, 0,
		time.UTC,
	)

	customer := models.Customer{
		CustomerCode: "CUS-E5D-2",
		FullName:     "Partial Customer",
		Mobile:       "01710000002",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-E5D-2",
		Name:        "Partial Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	oldExpiry := now.AddDate(0, 0, 10)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-E5D-2",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now.AddDate(0, -1, 0),
		NextBillingDate:  oldExpiry,
		ExpiryDate:       oldExpiry,
		BillingDay:       oldExpiry.Day(),
		Status:           "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{
		InvoiceNo:      "INV-E5D-2",
		SubscriptionID: subscription.ID,
		CustomerID:     customer.ID,
		PackageID:      pkg.ID,
		BillMonth:      int(now.Month()),
		BillYear:       now.Year(),
		IssueDate:      now,
		DueDate:        now,
		PackagePrice:   500,
		TotalAmount:    500,
		PaidAmount:     0,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	payment := &models.Payment{
		InvoiceID:   invoice.ID,
		PaymentDate: now,
		Amount:      200,
		Method:      "CASH",
		Status:      "SUCCESS",
	}

	if err := CreatePayment(payment); err != nil {
		t.Fatalf("create partial payment: %v", err)
	}

	var savedInvoice models.Invoice
	if err := db.First(
		&savedInvoice,
		invoice.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if savedInvoice.Status != "PARTIAL" {
		t.Fatalf(
			"invoice status = %q, want PARTIAL",
			savedInvoice.Status,
		)
	}

	var savedSubscription models.Subscription
	if err := db.First(
		&savedSubscription,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if !savedSubscription.ExpiryDate.Equal(
		oldExpiry,
	) {
		t.Fatal("partial payment changed subscription expiry")
	}

	var renewalCount int64
	if err := db.Model(
		&models.SubscriptionRenewal{},
	).Count(
		&renewalCount,
	).Error; err != nil {
		t.Fatal(err)
	}

	if renewalCount != 0 {
		t.Fatalf(
			"renewal rows = %d, want 0",
			renewalCount,
		)
	}
}

func TestCreatePaymentDisconnectedSubscriptionDoesNotAutoRenew(
	t *testing.T,
) {
	db := setupPaymentServiceTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 0, 0, 0,
		time.UTC,
	)

	customer := models.Customer{
		CustomerCode: "CUS-E5D-3",
		FullName:     "Disconnected Customer",
		Mobile:       "01710000003",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-E5D-3",
		Name:        "Disconnected Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode: "SUB-E5D-3",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now.AddDate(0, -2, 0),
		NextBillingDate:  now.AddDate(0, 0, -5),
		ExpiryDate:       now.AddDate(0, 0, -5),
		BillingDay:       1,
		Status:           "DISCONNECTED",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{
		InvoiceNo:      "INV-E5D-3",
		SubscriptionID: subscription.ID,
		CustomerID:     customer.ID,
		PackageID:      pkg.ID,
		BillMonth:      int(now.Month()),
		BillYear:       now.Year(),
		IssueDate:      now,
		DueDate:        now,
		PackagePrice:   500,
		TotalAmount:    500,
		PaidAmount:     0,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	payment := &models.Payment{
		InvoiceID:   invoice.ID,
		PaymentDate: now,
		Amount:      500,
		Method:      "CASH",
		Status:      "SUCCESS",
	}

	if err := CreatePayment(payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	var saved models.Subscription
	if err := db.First(
		&saved,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Status != "DISCONNECTED" {
		t.Fatalf(
			"status = %q, want DISCONNECTED",
			saved.Status,
		)
	}

	var renewalCount int64
	if err := db.Model(
		&models.SubscriptionRenewal{},
	).Count(
		&renewalCount,
	).Error; err != nil {
		t.Fatal(err)
	}

	if renewalCount != 0 {
		t.Fatalf(
			"renewal rows = %d, want 0",
			renewalCount,
		)
	}
}

func TestCreatePaymentRenewalFailureRollsBackPaymentInvoiceAndSubscription(
	t *testing.T,
) {
	db := setupPaymentServiceTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		16, 0, 0, 0,
		time.UTC,
	)

	customer := models.Customer{
		CustomerCode: "CUS-E5D2-ROLLBACK",
		FullName:     "Rollback Customer",
		Mobile:       "01710000999",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-E5D2-ROLLBACK",
		Name:        "Rollback Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	oldExpiry := time.Date(
		2026, time.August, 5,
		0, 0, 0, 0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-E5D2-ROLLBACK",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now.AddDate(0, -2, 0),
		NextBillingDate:  oldExpiry,
		ExpiryDate:       oldExpiry,
		BillingDay:       5,
		Status:           "EXPIRED",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{
		InvoiceNo:      "INV-E5D2-ROLLBACK",
		SubscriptionID: subscription.ID,
		CustomerID:     customer.ID,
		PackageID:      pkg.ID,
		BillMonth:      int(now.Month()),
		BillYear:       now.Year(),
		IssueDate:      now,
		DueDate:        now,
		PackagePrice:   500,
		TotalAmount:    500,
		PaidAmount:     0,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	// Force renewal-ledger persistence to fail.
	// CreatePayment must then roll back the entire transaction.
	if err := db.Migrator().
		DropTable(&models.SubscriptionRenewal{}); err != nil {
		t.Fatalf(
			"drop renewal ledger table: %v",
			err,
		)
	}

	payment := &models.Payment{
		InvoiceID:   invoice.ID,
		PaymentDate: now,
		Amount:      500,
		Method:      "CASH",
		Status:      "SUCCESS",
	}

	if err := CreatePayment(payment); err == nil {
		t.Fatal(
			"expected payment creation to fail when renewal ledger write fails",
		)
	}

	var paymentCount int64
	if err := db.Model(&models.Payment{}).
		Where("invoice_id = ?", invoice.ID).
		Count(&paymentCount).Error; err != nil {
		t.Fatal(err)
	}

	if paymentCount != 0 {
		t.Fatalf(
			"payment rows = %d, want 0 after rollback",
			paymentCount,
		)
	}

	var savedInvoice models.Invoice
	if err := db.First(
		&savedInvoice,
		invoice.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if savedInvoice.PaidAmount != 0 {
		t.Fatalf(
			"invoice paid amount = %.2f, want 0 after rollback",
			savedInvoice.PaidAmount,
		)
	}

	if savedInvoice.DueAmount != 500 {
		t.Fatalf(
			"invoice due amount = %.2f, want 500 after rollback",
			savedInvoice.DueAmount,
		)
	}

	if savedInvoice.Status != "UNPAID" {
		t.Fatalf(
			"invoice status = %q, want UNPAID after rollback",
			savedInvoice.Status,
		)
	}

	var savedSubscription models.Subscription
	if err := db.First(
		&savedSubscription,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if !savedSubscription.ExpiryDate.Equal(oldExpiry) {
		t.Fatalf(
			"subscription expiry = %v, want %v after rollback",
			savedSubscription.ExpiryDate,
			oldExpiry,
		)
	}

	if !savedSubscription.NextBillingDate.Equal(oldExpiry) {
		t.Fatalf(
			"next billing date = %v, want %v after rollback",
			savedSubscription.NextBillingDate,
			oldExpiry,
		)
	}

	if savedSubscription.Status != "EXPIRED" {
		t.Fatalf(
			"subscription status = %q, want EXPIRED after rollback",
			savedSubscription.Status,
		)
	}
}
