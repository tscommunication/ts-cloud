package services

import (
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionRenewalReversalServiceTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() +
		"?mode=memory&cache=shared"

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Subscription{},
		&models.SubscriptionRenewal{},
		&models.SubscriptionRenewalReversal{},
		&models.SubscriptionDateAdjustment{},
	); err != nil {
		t.Fatal(err)
	}

	return db
}

func seedRenewalReversalEligibility(
	t *testing.T,
	db *gorm.DB,
) (*models.Subscription, *models.SubscriptionRenewal) {
	t.Helper()

	now := time.Date(
		2026, time.August, 19,
		12, 0, 0, 0,
		time.UTC,
	)

	oldExpiry := now
	newExpiry := now.AddDate(0, 1, 0)

	subscription := &models.Subscription{
		SubscriptionCode: "SUB-REVERSAL",
		CustomerID:       1,
		PackageID:        1,
		Status:           "ACTIVE",
		ActivationDate:   now.AddDate(0, -1, 0),
		ExpiryDate:       newExpiry,
		NextBillingDate:  newExpiry,
		BillingDay:       19,
	}

	if err := db.Create(subscription).Error; err != nil {
		t.Fatal(err)
	}

	renewal := &models.SubscriptionRenewal{
		PaymentID:          201,
		InvoiceID:          301,
		CustomerID:         1,
		SubscriptionID:     subscription.ID,
		OldExpiryDate:      oldExpiry,
		NewExpiryDate:      newExpiry,
		OldNextBillingDate: oldExpiry,
		NewNextBillingDate: newExpiry,
		RenewalDate:        now,
		Amount:             500,
		Source:             "PAYMENT",
	}

	if err := db.Create(renewal).Error; err != nil {
		t.Fatal(err)
	}

	return subscription, renewal
}

func TestRenewalReversalEligibilityAllowsLatestMatchingRenewal(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	_, renewal :=
		seedRenewalReversalEligibility(t, db)

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			renewal.PaymentID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if !result.RenewalFound {
		t.Fatal("expected renewal to be found")
	}

	if !result.Eligible {
		t.Fatalf(
			"expected reversal eligibility, got reason %q",
			result.Reason,
		)
	}

	if result.RenewalID != renewal.ID {
		t.Fatalf(
			"renewal id = %d, want %d",
			result.RenewalID,
			renewal.ID,
		)
	}

	if !result.RestoredExpiryDate.Equal(
		renewal.OldExpiryDate,
	) {
		t.Fatal("restored expiry date mismatch")
	}
}

func TestRenewalReversalEligibilityRejectsNewerRenewal(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	subscription, renewal :=
		seedRenewalReversalEligibility(t, db)

	later := *renewal
	later.ID = 0
	later.PaymentID = 202
	later.OldExpiryDate = renewal.NewExpiryDate
	later.NewExpiryDate =
		renewal.NewExpiryDate.AddDate(0, 1, 0)
	later.OldNextBillingDate =
		renewal.NewNextBillingDate
	later.NewNextBillingDate =
		renewal.NewNextBillingDate.AddDate(0, 1, 0)
	later.RenewalDate =
		renewal.RenewalDate.Add(time.Hour)

	if err := db.Create(&later).Error; err != nil {
		t.Fatal(err)
	}

	subscription.ExpiryDate = later.NewExpiryDate
	subscription.NextBillingDate =
		later.NewNextBillingDate

	if err := db.Save(subscription).Error; err != nil {
		t.Fatal(err)
	}

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			renewal.PaymentID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Eligible {
		t.Fatal("older renewal must not be automatically reversed")
	}

	if result.Reason !=
		"a newer subscription renewal exists" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestRenewalReversalEligibilityRejectsManualDateAdjustment(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	subscription, renewal :=
		seedRenewalReversalEligibility(t, db)

	adjustment := &models.SubscriptionDateAdjustment{
		SubscriptionID: subscription.ID,

		OldExpiryDate: renewal.NewExpiryDate,
		NewExpiryDate: renewal.NewExpiryDate,

		OldNextBillingDate: renewal.NewNextBillingDate,
		NewNextBillingDate: renewal.NewNextBillingDate,

		OldStatus: "ACTIVE",
		NewStatus: "ACTIVE",

		Reason:           "manual correction",
		AdjustedByUserID: 1,
		AdjustedAt:       renewal.RenewalDate.Add(time.Minute),

		WithoutBilling: true,
	}

	if err := db.Create(adjustment).Error; err != nil {
		t.Fatal(err)
	}

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			renewal.PaymentID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Eligible {
		t.Fatal(
			"manual post-renewal date adjustment must block automatic reversal",
		)
	}

	if result.Reason !=
		"subscription date was manually adjusted after renewal" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestRenewalReversalEligibilityRejectsDateDrift(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	subscription, renewal :=
		seedRenewalReversalEligibility(t, db)

	subscription.ExpiryDate =
		subscription.ExpiryDate.AddDate(0, 0, 1)

	if err := db.Save(subscription).Error; err != nil {
		t.Fatal(err)
	}

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			renewal.PaymentID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Eligible {
		t.Fatal("date drift must block automatic reversal")
	}

	if result.Reason !=
		"current subscription dates no longer match renewal state" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestRenewalReversalEligibilityRejectsAlreadyReversed(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	_, renewal :=
		seedRenewalReversalEligibility(t, db)

	reversal := &models.SubscriptionRenewalReversal{
		RenewalID:      renewal.ID,
		PaymentID:      renewal.PaymentID,
		InvoiceID:      renewal.InvoiceID,
		CustomerID:     renewal.CustomerID,
		SubscriptionID: renewal.SubscriptionID,

		PreviousExpiryDate: renewal.NewExpiryDate,
		RestoredExpiryDate: renewal.OldExpiryDate,

		PreviousNextBillingDate: renewal.NewNextBillingDate,
		RestoredNextBillingDate: renewal.OldNextBillingDate,

		Reason:           "payment void",
		ReversedByUserID: 1,
		ReversedAt:       renewal.RenewalDate.Add(time.Hour),
	}

	if err := db.Create(reversal).Error; err != nil {
		t.Fatal(err)
	}

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			renewal.PaymentID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Eligible {
		t.Fatal("already reversed renewal must not be eligible")
	}

	if result.Reason !=
		"subscription renewal is already reversed" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestRenewalReversalEligibilityNoRenewalIsSafeNoop(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	result, err :=
		EvaluateSubscriptionRenewalReversalTx(
			db,
			999999,
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.RenewalFound {
		t.Fatal("unexpected renewal")
	}

	if result.Eligible {
		t.Fatal("missing renewal cannot be eligible")
	}

	if result.Reason !=
		"payment has no subscription renewal" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestReverseSubscriptionRenewalForPaymentTxRestoresDates(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	subscription, renewal :=
		seedRenewalReversalEligibility(t, db)

	reversalTime :=
		renewal.RenewalDate.Add(time.Hour)

	result, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			renewal.PaymentID,
			"payment void",
			9,
			reversalTime,
		)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Reversed {
		t.Fatalf(
			"expected reversal, got reason %q",
			result.Reason,
		)
	}

	var saved models.Subscription
	if err := db.First(
		&saved,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if !saved.ExpiryDate.Equal(
		renewal.OldExpiryDate,
	) {
		t.Fatal("expiry date was not restored")
	}

	if !saved.NextBillingDate.Equal(
		renewal.OldNextBillingDate,
	) {
		t.Fatal("next billing date was not restored")
	}

	var reversal models.SubscriptionRenewalReversal
	if err := db.Where(
		"payment_id = ?",
		renewal.PaymentID,
	).First(&reversal).Error; err != nil {
		t.Fatal(err)
	}

	if reversal.ReversedByUserID != 9 {
		t.Fatalf(
			"reversal actor = %d, want 9",
			reversal.ReversedByUserID,
		)
	}

	if reversal.Reason != "payment void" {
		t.Fatalf(
			"reversal reason = %q",
			reversal.Reason,
		)
	}
}

func TestReverseSubscriptionRenewalForPaymentTxRestoresExpiredStatus(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	_, renewal :=
		seedRenewalReversalEligibility(t, db)

	now :=
		renewal.OldExpiryDate.AddDate(0, 0, 2)

	result, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			renewal.PaymentID,
			"late payment correction",
			1,
			now,
		)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Reversed {
		t.Fatalf(
			"expected reversal, got reason %q",
			result.Reason,
		)
	}

	var saved models.Subscription
	if err := db.First(
		&saved,
		renewal.SubscriptionID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if saved.Status != "EXPIRED" {
		t.Fatalf(
			"status = %q, want EXPIRED",
			saved.Status,
		)
	}
}

func TestReverseSubscriptionRenewalForPaymentTxBlocksUnsafeRenewal(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	subscription, renewal :=
		seedRenewalReversalEligibility(t, db)

	later := *renewal
	later.ID = 0
	later.PaymentID = 202
	later.OldExpiryDate =
		renewal.NewExpiryDate
	later.NewExpiryDate =
		renewal.NewExpiryDate.AddDate(0, 1, 0)
	later.OldNextBillingDate =
		renewal.NewNextBillingDate
	later.NewNextBillingDate =
		renewal.NewNextBillingDate.AddDate(0, 1, 0)
	later.RenewalDate =
		renewal.RenewalDate.Add(time.Hour)

	if err := db.Create(&later).Error; err != nil {
		t.Fatal(err)
	}

	subscription.ExpiryDate =
		later.NewExpiryDate
	subscription.NextBillingDate =
		later.NewNextBillingDate

	if err := db.Save(subscription).Error; err != nil {
		t.Fatal(err)
	}

	result, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			renewal.PaymentID,
			"payment void",
			1,
			later.RenewalDate.Add(time.Hour),
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Reversed {
		t.Fatal("unsafe historical renewal was reversed")
	}

	if result.Reason !=
		"a newer subscription renewal exists" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}

	var reversalCount int64
	if err := db.Model(
		&models.SubscriptionRenewalReversal{},
	).Count(&reversalCount).Error; err != nil {
		t.Fatal(err)
	}

	if reversalCount != 0 {
		t.Fatalf(
			"expected no reversal audit, got %d",
			reversalCount,
		)
	}
}

func TestReverseSubscriptionRenewalForPaymentTxNoRenewalIsNoop(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	result, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			999999,
			"payment void",
			1,
			time.Now(),
		)
	if err != nil {
		t.Fatal(err)
	}

	if result.Reversed {
		t.Fatal("missing renewal unexpectedly reversed")
	}

	if result.Reason !=
		"payment has no subscription renewal" {
		t.Fatalf(
			"unexpected reason %q",
			result.Reason,
		)
	}
}

func TestReverseSubscriptionRenewalForPaymentTxSecondAttemptIsNoop(
	t *testing.T,
) {
	db := setupSubscriptionRenewalReversalServiceTestDB(t)

	_, renewal :=
		seedRenewalReversalEligibility(t, db)

	first, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			renewal.PaymentID,
			"payment void",
			1,
			renewal.RenewalDate.Add(time.Hour),
		)
	if err != nil {
		t.Fatal(err)
	}

	if !first.Reversed {
		t.Fatal("first reversal did not occur")
	}

	second, err :=
		ReverseSubscriptionRenewalForPaymentTx(
			db,
			renewal.PaymentID,
			"payment void retry",
			1,
			renewal.RenewalDate.Add(2*time.Hour),
		)
	if err != nil {
		t.Fatal(err)
	}

	if second.Reversed {
		t.Fatal("second reversal unexpectedly occurred")
	}

	if second.Reason !=
		"subscription renewal is already reversed" {
		t.Fatalf(
			"unexpected second-attempt reason %q",
			second.Reason,
		)
	}
}
