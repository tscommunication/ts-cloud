package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

const lifecycleHandlerCredentialKey = "0123456789abcdef0123456789abcdef"

func setupLifecycleHandlerDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:"+
				t.Name()+
				"?mode=memory&cache=shared",
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatalf(
			"open lifecycle handler database: %v",
			err,
		)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.NetworkRouter{},
		&models.Subscription{},
	); err != nil {
		t.Fatalf(
			"migrate lifecycle handler database: %v",
			err,
		)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func createLifecycleHandlerFixture(
	t *testing.T,
	db *gorm.DB,
	status string,
) models.Subscription {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-LIFE-001",
		FullName:     "Lifecycle Customer",
		Mobile:       "01700000001",
		NID:          "1234567890",
		Status:       "ACTIVE",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf(
			"create lifecycle customer: %v",
			err,
		)
	}

	pkg := models.Package{
		PackageCode:     "PKG-LIFE-001",
		Name:            "Lifecycle Package",
		Price:           500,
		MikroTikProfile: "Go_P25",
		Status:          "ACTIVE",
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf(
			"create lifecycle package: %v",
			err,
		)
	}

	now := time.Now()

	subscription := models.Subscription{
		SubscriptionCode: "SUB-LIFE-001",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		ActivationDate:   now,
		BillingDay:       now.Day(),
		NextBillingDate:  now.AddDate(0, 1, 0),
		ExpiryDate:       now.AddDate(0, 1, 0),
		Status:           status,
		PPPoEUsername:    "life-user",
	}

	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf(
			"create lifecycle subscription: %v",
			err,
		)
	}

	return subscription
}

func performLifecycleHandlerRequest(
	t *testing.T,
	handler gin.HandlerFunc,
	path string,
	id uint,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST(path, handler)

	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodPost,
		replaceLifecycleID(
			path,
			id,
		),
		bytes.NewReader(body),
	)

	if len(body) > 0 {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	router.ServeHTTP(
		recorder,
		request,
	)

	return recorder
}

func replaceLifecycleID(
	path string,
	id uint,
) string {
	result := path

	for index := 0; index+2 < len(result); index++ {
		if result[index:index+3] == ":id" {
			return result[:index] +
				strconv.FormatUint(
					uint64(id),
					10,
				) +
				result[index+3:]
		}
	}

	return result
}

func TestSuspendSubscriptionHandlerPostCommitReconciliation(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"ACTIVE",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	called := false

	handler := suspendSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			called = true

			if row.Status != "SUSPENDED" {
				t.Fatalf(
					"runner saw status %q",
					row.Status,
				)
			}

			if action !=
				services.SubscriptionLifecycleSuspend {
				t.Fatalf(
					"action = %q",
					action,
				)
			}

			if keyMaterial !=
				lifecycleHandlerCredentialKey {
				t.Fatal(
					"credential key was not propagated",
				)
			}

			return services.SubscriptionLifecycleReconciliationResult{
				Action:                  action,
				SubscriptionID:          row.ID,
				Status:                  row.Status,
				ReconciliationAttempted: true,
			}, nil
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/suspend",
		subscription.ID,
		nil,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if !called {
		t.Fatal(
			"reconciliation runner was not called",
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "SUSPENDED" {
		t.Fatalf(
			"stored status = %q",
			stored.Status,
		)
	}
}

func TestActivateSubscriptionHandlerPostCommitReconciliation(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"SUSPENDED",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	handler := activateSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			if row.Status != "ACTIVE" {
				t.Fatalf(
					"runner saw status %q",
					row.Status,
				)
			}

			if action !=
				services.SubscriptionLifecycleActivate {
				t.Fatalf(
					"action = %q",
					action,
				)
			}

			return services.SubscriptionLifecycleReconciliationResult{
				Action:                  action,
				SubscriptionID:          row.ID,
				Status:                  row.Status,
				ReconciliationAttempted: true,
			}, nil
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/activate",
		subscription.ID,
		nil,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "ACTIVE" {
		t.Fatalf(
			"stored status = %q",
			stored.Status,
		)
	}
}

func TestRenewSubscriptionHandlerPostCommitReconciliation(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"ACTIVE",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	handler := renewSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			if row.Status != "ACTIVE" {
				t.Fatalf(
					"runner saw status %q",
					row.Status,
				)
			}

			if action !=
				services.SubscriptionLifecycleRenew {
				t.Fatalf(
					"action = %q",
					action,
				)
			}

			return services.SubscriptionLifecycleReconciliationResult{
				Action:                  action,
				SubscriptionID:          row.ID,
				Status:                  row.Status,
				ReconciliationAttempted: true,
			}, nil
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/renew",
		subscription.ID,
		[]byte(`{"months":1}`),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestDisconnectSubscriptionHandlerPostCommitReconciliation(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"ACTIVE",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	handler := disconnectSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			if row.Status != "DISCONNECTED" {
				t.Fatalf(
					"runner saw status %q",
					row.Status,
				)
			}

			if action !=
				services.SubscriptionLifecycleDisconnect {
				t.Fatalf(
					"action = %q",
					action,
				)
			}

			return services.SubscriptionLifecycleReconciliationResult{
				Action:                  action,
				SubscriptionID:          row.ID,
				Status:                  row.Status,
				ReconciliationAttempted: true,
			}, nil
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/disconnect",
		subscription.ID,
		nil,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "DISCONNECTED" {
		t.Fatalf(
			"stored status = %q",
			stored.Status,
		)
	}
}

func TestLifecycleHandlerReconciliationFailureDoesNotRollbackMutation(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"ACTIVE",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	handler := suspendSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			return services.SubscriptionLifecycleReconciliationResult{
					Action:                  action,
					SubscriptionID:          row.ID,
					Status:                  row.Status,
					ReconciliationAttempted: true,
				},
				errors.New(
					"simulated reconciliation boundary failure",
				)
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/suspend",
		subscription.ID,
		nil,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var body map[string]any

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	if body["pppoe_reconciliation_error"] !=
		"simulated reconciliation boundary failure" {
		t.Fatalf(
			"unexpected reconciliation error body: %v",
			body,
		)
	}

	var stored models.Subscription

	if err := db.First(
		&stored,
		subscription.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != "SUSPENDED" {
		t.Fatalf(
			"mutation rolled back; stored status = %q",
			stored.Status,
		)
	}
}

func TestLifecycleHandlerNoReconciliationBeforeMutationFailure(
	t *testing.T,
) {
	db := setupLifecycleHandlerDB(t)

	subscription :=
		createLifecycleHandlerFixture(
			t,
			db,
			"SUSPENDED",
		)

	cfg := &config.Config{
		CredentialKey: lifecycleHandlerCredentialKey,
	}

	called := false

	handler := suspendSubscriptionHandler(
		cfg,
		func(
			row *models.Subscription,
			action services.SubscriptionLifecycleAction,
			keyMaterial string,
		) (
			services.SubscriptionLifecycleReconciliationResult,
			error,
		) {
			called = true
			return services.SubscriptionLifecycleReconciliationResult{},
				nil
		},
	)

	recorder := performLifecycleHandlerRequest(
		t,
		handler,
		"/subscriptions/:id/suspend",
		subscription.ID,
		nil,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusConflict,
		)
	}

	if called {
		t.Fatal(
			"reconciliation ran before/after failed lifecycle mutation",
		)
	}
}
