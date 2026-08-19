package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const backfillCredentialKey = "0123456789abcdef0123456789abcdef"

func setupPPPoEBackfillTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.CustomerProvisionRequest{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	return db
}

func createBackfillCustomerAndPackage(
	t *testing.T,
	db *gorm.DB,
) (models.Customer, models.Package) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-BACKFILL-001",
		FullName:     "Backfill Customer",
		Mobile:       "01712345678",
		NID:          "1234567890",
		Status:       "ACTIVE",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	pkg := models.Package{
		PackageCode: "PKG-BACKFILL",
		Name:        "Backfill Package",
		Status:      "ACTIVE",
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	return customer, pkg
}

func TestBackfillLegacyPPPoECredentialsEncryptsPlaintext(
	t *testing.T,
) {
	db := setupPPPoEBackfillTestDB(t)

	customer, pkg := createBackfillCustomerAndPackage(
		t,
		db,
	)

	now := time.Date(
		2026,
		8,
		19,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-BACKFILL-001",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now,
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 1, 0),
		ExpiryDate:       now.AddDate(0, 1, 0),
		Status:           "ACTIVE",
		PPPoEUsername:    "legacy-sub-user",
		PPPoEPassword:    "legacy-sub-secret",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	request := models.CustomerProvisionRequest{
		RequestCode:       "CPR-BACKFILL-001",
		Source:            "AGENT",
		Status:            "PENDING",
		FullName:          "Legacy Provision User",
		Mobile:            "01812345678",
		NID:               "1234567891",
		PackageID:         pkg.ID,
		PPPoEUsername:     "legacy-request-user",
		PPPoEPassword:     "legacy-request-secret",
		BillingDay:        19,
		ActivationDate:    now,
		RequestedByUserID: 1,
		RequestedAt:       now,
	}

	if err := db.Create(&request).Error; err != nil {
		t.Fatalf("create provision request: %v", err)
	}

	result, err := BackfillLegacyPPPoECredentials(
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf("backfill credentials: %v", err)
	}

	if result.SubscriptionsUpdated != 1 {
		t.Fatalf(
			"subscriptions updated = %d, want 1",
			result.SubscriptionsUpdated,
		)
	}

	if result.ProvisionRequestsUpdated != 1 {
		t.Fatalf(
			"provision requests updated = %d, want 1",
			result.ProvisionRequestsUpdated,
		)
	}

	var savedSubscription models.Subscription

	if err := db.First(
		&savedSubscription,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload subscription: %v",
			err,
		)
	}

	if savedSubscription.PPPoEPassword != "" {
		t.Fatalf(
			"subscription plaintext password not cleared; length=%d",
			len(savedSubscription.PPPoEPassword),
		)
	}

	if savedSubscription.PPPoEPasswordEncrypted == "" {
		t.Fatal(
			"subscription encrypted PPPoE password is blank",
		)
	}

	subscriptionPassword, err := security.DecryptSecret(
		savedSubscription.PPPoEPasswordEncrypted,
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf(
			"decrypt subscription PPPoE password: %v",
			err,
		)
	}

	if subscriptionPassword != "legacy-sub-secret" {
		t.Fatalf(
			"unexpected subscription password %q",
			subscriptionPassword,
		)
	}

	var savedRequest models.CustomerProvisionRequest

	if err := db.First(
		&savedRequest,
		request.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload provision request: %v",
			err,
		)
	}

	if savedRequest.PPPoEPassword != "" {
		t.Fatalf(
			"provision plaintext password not cleared; length=%d",
			len(savedRequest.PPPoEPassword),
		)
	}

	if savedRequest.PPPoEPasswordEncrypted == "" {
		t.Fatal(
			"provision encrypted PPPoE password is blank",
		)
	}

	requestPassword, err := security.DecryptSecret(
		savedRequest.PPPoEPasswordEncrypted,
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf(
			"decrypt provision PPPoE password: %v",
			err,
		)
	}

	if requestPassword != "legacy-request-secret" {
		t.Fatalf(
			"unexpected provision password %q",
			requestPassword,
		)
	}
}

func TestBackfillLegacyPPPoECredentialsIsIdempotent(
	t *testing.T,
) {
	db := setupPPPoEBackfillTestDB(t)

	customer, pkg := createBackfillCustomerAndPackage(
		t,
		db,
	)

	now := time.Date(
		2026,
		8,
		19,
		10,
		30,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-BACKFILL-IDEMPOTENT",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now,
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 1, 0),
		ExpiryDate:       now.AddDate(0, 1, 0),
		Status:           "ACTIVE",
		PPPoEUsername:    "idempotent-user",
		PPPoEPassword:    "idempotent-secret",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	first, err := BackfillLegacyPPPoECredentials(
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf("first backfill: %v", err)
	}

	if first.SubscriptionsUpdated != 1 {
		t.Fatalf(
			"first subscriptions updated = %d, want 1",
			first.SubscriptionsUpdated,
		)
	}

	var afterFirst models.Subscription

	if err := db.First(
		&afterFirst,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload after first backfill: %v",
			err,
		)
	}

	firstCiphertext := afterFirst.PPPoEPasswordEncrypted

	second, err := BackfillLegacyPPPoECredentials(
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}

	if second.SubscriptionsUpdated != 0 {
		t.Fatalf(
			"second subscriptions updated = %d, want 0",
			second.SubscriptionsUpdated,
		)
	}

	if second.ProvisionRequestsUpdated != 0 {
		t.Fatalf(
			"second provision requests updated = %d, want 0",
			second.ProvisionRequestsUpdated,
		)
	}

	var afterSecond models.Subscription

	if err := db.First(
		&afterSecond,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload after second backfill: %v",
			err,
		)
	}

	if afterSecond.PPPoEPasswordEncrypted != firstCiphertext {
		t.Fatal(
			"idempotent backfill replaced existing ciphertext",
		)
	}
}

func TestBackfillLegacyPPPoECredentialsPreservesEncryptedValue(
	t *testing.T,
) {
	db := setupPPPoEBackfillTestDB(t)

	customer, pkg := createBackfillCustomerAndPackage(
		t,
		db,
	)

	encrypted, err := security.EncryptSecret(
		"already-encrypted-secret",
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf(
			"prepare encrypted secret: %v",
			err,
		)
	}

	now := time.Date(
		2026,
		8,
		19,
		11,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode:       "SUB-BACKFILL-PRESERVE",
		CustomerID:             customer.ID,
		PackageID:              pkg.ID,
		ActivationDate:         now,
		BillingDay:             19,
		NextBillingDate:        now.AddDate(0, 1, 0),
		ExpiryDate:             now.AddDate(0, 1, 0),
		Status:                 "ACTIVE",
		PPPoEUsername:          "preserve-user",
		PPPoEPassword:          "legacy-should-not-overwrite",
		PPPoEPasswordEncrypted: encrypted,
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	result, err := BackfillLegacyPPPoECredentials(
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if result.SubscriptionsUpdated != 0 {
		t.Fatalf(
			"subscriptions updated = %d, want 0",
			result.SubscriptionsUpdated,
		)
	}

	var saved models.Subscription

	if err := db.First(
		&saved,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload subscription: %v",
			err,
		)
	}

	if saved.PPPoEPasswordEncrypted != encrypted {
		t.Fatal(
			"existing encrypted credential was changed",
		)
	}
}

func TestBackfillLegacyPPPoECredentialsSkipsBlankPasswords(
	t *testing.T,
) {
	db := setupPPPoEBackfillTestDB(t)

	customer, pkg := createBackfillCustomerAndPackage(
		t,
		db,
	)

	now := time.Date(
		2026,
		8,
		19,
		11,
		10,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-BACKFILL-BLANK",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now,
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 1, 0),
		ExpiryDate:       now.AddDate(0, 1, 0),
		Status:           "ACTIVE",
		PPPoEUsername:    "blank-user",
		PPPoEPassword:    "   ",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	result, err := BackfillLegacyPPPoECredentials(
		backfillCredentialKey,
	)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if result.SubscriptionsUpdated != 0 {
		t.Fatalf(
			"subscriptions updated = %d, want 0",
			result.SubscriptionsUpdated,
		)
	}
}

func TestBackfillLegacyPPPoECredentialsRollsBackOnInvalidKey(
	t *testing.T,
) {
	db := setupPPPoEBackfillTestDB(t)

	customer, pkg := createBackfillCustomerAndPackage(
		t,
		db,
	)

	now := time.Date(
		2026,
		8,
		19,
		11,
		20,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode: "SUB-BACKFILL-ROLLBACK",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now,
		BillingDay:       19,
		NextBillingDate:  now.AddDate(0, 1, 0),
		ExpiryDate:       now.AddDate(0, 1, 0),
		Status:           "ACTIVE",
		PPPoEUsername:    "rollback-user",
		PPPoEPassword:    "rollback-secret",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	if _, err := BackfillLegacyPPPoECredentials(
		"short-key",
	); err == nil {
		t.Fatal(
			"expected invalid credential key to fail",
		)
	}

	var saved models.Subscription

	if err := db.First(
		&saved,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"reload subscription: %v",
			err,
		)
	}

	if saved.PPPoEPassword != "rollback-secret" {
		t.Fatal(
			"plaintext credential changed despite transaction rollback",
		)
	}

	if saved.PPPoEPasswordEncrypted != "" {
		t.Fatal(
			"encrypted credential persisted despite transaction rollback",
		)
	}
}
