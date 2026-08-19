package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

const reconciliationPlanTestKey = "0123456789abcdef0123456789abcdef"

type fakePPPSecretReader struct {
	Rows []mikrotik.PPPSecret
	Err  error

	Calls       int
	RouterID    uint
	Requested   string
	KeyMaterial string
}

func (reader *fakePPPSecretReader) ListPPPSecrets(
	router *models.NetworkRouter,
	name string,
	keyMaterial string,
) ([]mikrotik.PPPSecret, error) {
	reader.Calls++
	reader.Requested = name
	reader.KeyMaterial = keyMaterial

	if router != nil {
		reader.RouterID = router.ID
	}

	if reader.Err != nil {
		return nil, reader.Err
	}

	return reader.Rows, nil
}

func setupPPPReconciliationPlanDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	dsn := "file:" +
		t.Name() +
		"?mode=memory&cache=shared"

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf(
			"open reconciliation database: %v",
			err,
		)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.NetworkRouter{},
		&models.Subscription{},
	); err != nil {
		t.Fatalf(
			"migrate reconciliation database: %v",
			err,
		)
	}

	return db
}

func createPPPReconciliationPlanFixture(
	t *testing.T,
	db *gorm.DB,
	status string,
) (
	models.Subscription,
	models.NetworkRouter,
	models.Package,
) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-RECON-001",
		FullName:     "Reconciliation Customer",
		Mobile:       "01700000001",
		NID:          "1234567890",
		Status:       "ACTIVE",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	pkg := models.Package{
		PackageCode:     "PKG-RECON",
		Name:            "Reconciliation Package",
		MikroTikProfile: "Go_P25",
		Status:          "ACTIVE",
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	routerPassword, err := security.EncryptSecret(
		"router-secret",
		reconciliationPlanTestKey,
	)
	if err != nil {
		t.Fatalf(
			"encrypt router password: %v",
			err,
		)
	}

	router := models.NetworkRouter{
		Code:                 "RTR-RECON-01",
		Name:                 "Reconciliation Router",
		Host:                 "192.0.2.20",
		APIPort:              8729,
		APIUsername:          "api-user",
		APIPasswordEncrypted: routerPassword,
		UseTLS:               true,
		Status:               "ACTIVE",
	}

	if err := db.Create(&router).Error; err != nil {
		t.Fatalf("create router: %v", err)
	}

	pppoePassword, err := security.EncryptSecret(
		"subscriber-secret",
		reconciliationPlanTestKey,
	)
	if err != nil {
		t.Fatalf(
			"encrypt subscriber password: %v",
			err,
		)
	}

	now := time.Date(
		2026,
		8,
		19,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := models.Subscription{
		SubscriptionCode:       "SUB-RECON-001",
		CustomerID:             customer.ID,
		PackageID:              pkg.ID,
		ActivationDate:         now,
		BillingDay:             19,
		NextBillingDate:        now.AddDate(0, 1, 0),
		ExpiryDate:             now.AddDate(0, 1, 0),
		Status:                 status,
		RouterID:               router.ID,
		PPPoEUsername:          "subscriber-1",
		PPPoEPasswordEncrypted: pppoePassword,
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf(
			"create subscription: %v",
			err,
		)
	}

	return subscription, router, pkg
}

func TestBuildSubscriptionPPPSecretReconciliationPlanCreate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{}

	plan, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			reader,
		)
	if err != nil {
		t.Fatalf(
			"build reconciliation plan: %v",
			err,
		)
	}

	if plan.Action != PPPSecretActionCreate {
		t.Fatalf(
			"action = %q, want CREATE",
			plan.Action,
		)
	}

	if plan.SubscriptionID != subscription.ID {
		t.Fatalf(
			"subscription id = %d",
			plan.SubscriptionID,
		)
	}

	if plan.RouterID != router.ID {
		t.Fatalf(
			"router id = %d",
			plan.RouterID,
		)
	}

	if plan.Username != "subscriber-1" {
		t.Fatalf(
			"username = %q",
			plan.Username,
		)
	}

	if plan.Profile != "Go_P25" {
		t.Fatalf(
			"profile = %q",
			plan.Profile,
		)
	}

	if reader.Calls != 1 {
		t.Fatalf(
			"reader calls = %d, want 1",
			reader.Calls,
		)
	}

	if reader.RouterID != router.ID {
		t.Fatalf(
			"reader router id = %d",
			reader.RouterID,
		)
	}

	if reader.Requested != "subscriber-1" {
		t.Fatalf(
			"reader username = %q",
			reader.Requested,
		)
	}
}

func TestBuildSubscriptionPPPSecretReconciliationPlanDisable(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"SUSPENDED",
		)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*1",
				Name:     "subscriber-1",
				Service:  "pppoe",
				Profile:  "Go_P25",
				Disabled: false,
			},
		},
	}

	plan, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			reader,
		)
	if err != nil {
		t.Fatalf(
			"build reconciliation plan: %v",
			err,
		)
	}

	if plan.Action != PPPSecretActionDisable {
		t.Fatalf(
			"action = %q, want DISABLE",
			plan.Action,
		)
	}

	if plan.CurrentSecret == nil {
		t.Fatal(
			"expected current PPP secret",
		)
	}

	if plan.CurrentSecret.ID != "*1" {
		t.Fatalf(
			"current id = %q",
			plan.CurrentSecret.ID,
		)
	}
}

func TestBuildSubscriptionPPPSecretReconciliationPlanPropagatesConflict(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:   "*1",
				Name: "subscriber-1",
			},
			{
				ID:   "*2",
				Name: "SUBSCRIBER-1",
			},
		},
	}

	plan, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			reader,
		)
	if err != nil {
		t.Fatalf(
			"build reconciliation plan: %v",
			err,
		)
	}

	if plan.Action != PPPSecretActionConflict {
		t.Fatalf(
			"action = %q, want CONFLICT",
			plan.Action,
		)
	}
}

func TestBuildSubscriptionPPPSecretReconciliationPlanRejectsInactiveRouter(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	router.Status = "MAINTENANCE"

	if err := db.Save(&router).Error; err != nil {
		t.Fatalf(
			"save inactive router: %v",
			err,
		)
	}

	reader := &fakePPPSecretReader{}

	_, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			reader,
		)
	if err == nil {
		t.Fatal(
			"expected inactive router to fail",
		)
	}

	if !strings.Contains(
		err.Error(),
		"active router",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if reader.Calls != 0 {
		t.Fatalf(
			"reader unexpectedly called %d times",
			reader.Calls,
		)
	}
}

func TestBuildSubscriptionPPPSecretReconciliationPlanPropagatesReaderFailure(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Err: errors.New("router read failed"),
	}

	_, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			reader,
		)
	if err == nil {
		t.Fatal(
			"expected reader failure",
		)
	}

	if !strings.Contains(
		err.Error(),
		"read RouterOS PPP secrets",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestBuildSubscriptionPPPSecretReconciliationPlanValidation(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	reader := &fakePPPSecretReader{}

	if _, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			0,
			reconciliationPlanTestKey,
			reader,
		); err == nil {
		t.Fatal(
			"expected zero subscription id to fail",
		)
	}

	if _, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			999,
			reconciliationPlanTestKey,
			reader,
		); err == nil {
		t.Fatal(
			"expected missing subscription to fail",
		)
	}

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	if _, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscription.ID,
			reconciliationPlanTestKey,
			nil,
		); err == nil {
		t.Fatal(
			"expected nil reader to fail",
		)
	}
}
