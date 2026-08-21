package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupPaymentRenewalTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.Invoice{},
		&models.Payment{},
		&models.SubscriptionRenewal{},
		&models.CustomerInternetAccount{},
		&models.TemporaryInternetAccess{},
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func TestPaidInvoiceDeductsGrantedTemporaryAccessFromNormalCycle(t *testing.T) {
	db := setupPaymentRenewalTestDB(t)
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	subscription, invoice, payment := seedPaymentRenewalFixture(
		t, db, now, "EXPIRED", time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC), 0, "PAID",
	)
	account := models.CustomerInternetAccount{
		AccountCode: "NET-RENEW", CustomerID: subscription.CustomerID, RouterID: 1,
		PPPoEUsername: "renew-user", PPPoEPasswordEncrypted: "encrypted",
		PackageID: subscription.PackageID, BillingDay: subscription.BillingDay, Status: TemporaryInternetStatusActive,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&subscription).Update("internet_account_id", account.ID).Error; err != nil {
		t.Fatal(err)
	}
	grant := models.TemporaryInternetAccess{
		CustomerID: subscription.CustomerID, InternetAccountID: account.ID, SubscriptionID: subscription.ID,
		Status: TemporaryInternetAccessExpired, StartsAt: now.Add(-48 * time.Hour), EndsAt: now,
		GrantedDurationSeconds: int64(48 * time.Hour / time.Second), RequestSource: "CUSTOMER",
		Reason: "two day promise", GrantedByUserID: 1, GrantedAt: now.Add(-48 * time.Hour),
	}
	if err := db.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}

	var result PaymentRenewalResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = RenewSubscriptionForPaidInvoiceTx(tx, &payment, &invoice, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC)
	if !result.Renewed || result.Renewal == nil || !result.Renewal.NewExpiryDate.Equal(want) {
		t.Fatalf("unexpected renewal: %+v", result)
	}
	if result.Renewal.TemporaryAccessDeductedSeconds != int64(48*time.Hour/time.Second) {
		t.Fatalf("deducted seconds = %d", result.Renewal.TemporaryAccessDeductedSeconds)
	}
	if err := db.First(&grant, grant.ID).Error; err != nil {
		t.Fatal(err)
	}
	if grant.Status != TemporaryInternetAccessSettled || grant.SettlementPaymentID == nil || *grant.SettlementPaymentID != payment.ID {
		t.Fatalf("temporary access was not settled: %+v", grant)
	}
}

func seedPaymentRenewalFixture(
	t *testing.T,
	db *gorm.DB,
	now time.Time,
	status string,
	expiry time.Time,
	invoiceDue float64,
	invoiceStatus string,
) (
	models.Subscription,
	models.Invoice,
	models.Payment,
) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-RENEW",
		FullName:     "Renewal Customer",
		Mobile:       "01700000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-RENEW",
		Name:        "Renewal Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode: "SUB-RENEW",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now.AddDate(0, -1, 0),
		NextBillingDate:  expiry,
		ExpiryDate:       expiry,
		BillingDay:       expiry.Day(),
		Status:           status,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{
		InvoiceNo:      "INV-RENEW",
		SubscriptionID: subscription.ID,
		CustomerID:     customer.ID,
		PackageID:      pkg.ID,
		BillMonth:      int(now.Month()),
		BillYear:       now.Year(),
		IssueDate:      now,
		DueDate:        now,
		PackagePrice:   500,
		TotalAmount:    500,
		PaidAmount:     500 - invoiceDue,
		DueAmount:      invoiceDue,
		Status:         invoiceStatus,
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	payment := models.Payment{
		ReceiptNo:      "RCPT-RENEW",
		InvoiceID:      invoice.ID,
		CustomerID:     customer.ID,
		SubscriptionID: subscription.ID,
		PaymentDate:    now,
		Amount:         500,
		Method:         "CASH",
		Status:         "SUCCESS",
	}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatal(err)
	}

	return subscription, invoice, payment
}

func TestPaidInvoiceRenewsExpiredSubscriptionFromRechargeDate(
	t *testing.T,
) {
	db := setupPaymentRenewalTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 30, 0, 0,
		time.UTC,
	)

	subscription, invoice, payment :=
		seedPaymentRenewalFixture(
			t,
			db,
			now,
			"EXPIRED",
			time.Date(
				2026, time.August, 5,
				0, 0, 0, 0,
				time.UTC,
			),
			0,
			"PAID",
		)

	var result PaymentRenewalResult

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = RenewSubscriptionForPaidInvoiceTx(
			tx,
			&payment,
			&invoice,
			now,
		)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.Renewed {
		t.Fatalf("renewed = false, reason=%q", result.Reason)
	}

	var saved models.Subscription
	if err := db.First(&saved, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}

	want := time.Date(
		2026, time.September, 19,
		0, 0, 0, 0,
		time.UTC,
	)

	if !saved.ExpiryDate.Equal(want) {
		t.Fatalf(
			"expiry = %v, want %v",
			saved.ExpiryDate,
			want,
		)
	}

	if saved.Status != "ACTIVE" {
		t.Fatalf(
			"status = %q, want ACTIVE",
			saved.Status,
		)
	}
}

func TestPartialPaymentDoesNotRenewSubscription(
	t *testing.T,
) {
	db := setupPaymentRenewalTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 30, 0, 0,
		time.UTC,
	)

	subscription, invoice, payment :=
		seedPaymentRenewalFixture(
			t,
			db,
			now,
			"ACTIVE",
			now.AddDate(0, 0, 10),
			200,
			"PARTIAL",
		)

	oldExpiry := subscription.ExpiryDate

	err := db.Transaction(func(tx *gorm.DB) error {
		result, err := RenewSubscriptionForPaidInvoiceTx(
			tx,
			&payment,
			&invoice,
			now,
		)
		if err != nil {
			return err
		}

		if result.Renewed {
			t.Fatal("partial payment unexpectedly renewed subscription")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var saved models.Subscription
	if err := db.First(&saved, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}

	if !saved.ExpiryDate.Equal(oldExpiry) {
		t.Fatal("partial payment changed expiry date")
	}

	var count int64
	if err := db.Model(&models.SubscriptionRenewal{}).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("renewal ledger rows = %d, want 0", count)
	}
}

func TestPaidInvoiceRenewalIsIdempotentByPayment(
	t *testing.T,
) {
	db := setupPaymentRenewalTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 30, 0, 0,
		time.UTC,
	)

	subscription, invoice, payment :=
		seedPaymentRenewalFixture(
			t,
			db,
			now,
			"ACTIVE",
			time.Date(
				2026, time.September, 19,
				0, 0, 0, 0,
				time.UTC,
			),
			0,
			"PAID",
		)

	run := func() PaymentRenewalResult {
		var result PaymentRenewalResult

		err := db.Transaction(func(tx *gorm.DB) error {
			var err error
			result, err =
				RenewSubscriptionForPaidInvoiceTx(
					tx,
					&payment,
					&invoice,
					now,
				)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}

		return result
	}

	first := run()

	if !first.Renewed {
		t.Fatalf(
			"first renewal failed: %q",
			first.Reason,
		)
	}

	var afterFirst models.Subscription
	if err := db.First(
		&afterFirst,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	second := run()

	if second.Renewed {
		t.Fatal("same payment renewed subscription twice")
	}

	var afterSecond models.Subscription
	if err := db.First(
		&afterSecond,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if !afterSecond.ExpiryDate.Equal(
		afterFirst.ExpiryDate,
	) {
		t.Fatal("second execution changed expiry date")
	}

	var count int64

	if err := db.Model(
		&models.SubscriptionRenewal{},
	).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf(
			"renewal rows = %d, want 1",
			count,
		)
	}
}

func TestDisconnectedSubscriptionIsNotAutoRenewed(
	t *testing.T,
) {
	db := setupPaymentRenewalTestDB(t)

	now := time.Date(
		2026, time.August, 19,
		15, 30, 0, 0,
		time.UTC,
	)

	subscription, invoice, payment :=
		seedPaymentRenewalFixture(
			t,
			db,
			now,
			"DISCONNECTED",
			now.AddDate(0, 0, -3),
			0,
			"PAID",
		)

	err := db.Transaction(func(tx *gorm.DB) error {
		result, err :=
			RenewSubscriptionForPaidInvoiceTx(
				tx,
				&payment,
				&invoice,
				now,
			)

		if err != nil {
			return err
		}

		if result.Renewed {
			t.Fatal("disconnected subscription was renewed")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
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
}
