package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

var customerInternetCredentialHandlerDBCounter atomic.Uint64

func setupCustomerInternetCredentialHandlerTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			fmt.Sprintf(
				"file:customer_internet_credential_handler_%d?mode=memory&cache=shared",
				customerInternetCredentialHandlerDBCounter.Add(1),
			),
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.User{},
		&models.CustomerInternetAccount{},
		&models.Package{},
		&models.Subscription{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	previousReconciler :=
		customerInternetCredentialPostSaveReconciler

	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
		customerInternetCredentialPostSaveReconciler =
			previousReconciler
	})

	return db
}

func createCustomerInternetCredentialHandlerFixture(
	t *testing.T,
	db *gorm.DB,
) (*models.Customer, *models.CustomerInternetAccount, *models.Subscription) {
	t.Helper()

	customer := &models.Customer{
		CustomerCode: "CUS-CRED-H01",
		FullName:     "Credential Handler Customer",
		Mobile:       "01770000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(customer).Error; err != nil {
		t.Fatal(err)
	}

	account := &models.CustomerInternetAccount{
		AccountCode:            "NET-CRED-H01",
		CustomerID:             customer.ID,
		RouterID:               7,
		PPPoEUsername:          "old-pppoe-user",
		PPPoEPasswordEncrypted: "legacy-encrypted",
		Status:                 "ACTIVE",
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}

	pkg := &models.Package{
		PackageCode: "PKG-CRED-H01",
		Name:        "Credential Handler Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := &models.Subscription{
		SubscriptionCode:  "SUB-CRED-H01",
		CustomerID:        customer.ID,
		PackageID:         pkg.ID,
		InternetAccountID: &account.ID,
		Status:            "ACTIVE",
	}
	if err := db.Create(subscription).Error; err != nil {
		t.Fatal(err)
	}

	return customer, account, subscription
}

func performCustomerInternetCredentialSave(
	t *testing.T,
	cfg *config.Config,
	customerID uint,
	payload string,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT(
		"/customers/:id/internet-credential",
		func(c *gin.Context) {
			c.Set("role", "superadmin")
			c.Next()
		},
		SaveCustomerInternetCredential(cfg),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf(
			"/customers/%d/internet-credential",
			customerID,
		),
		bytes.NewBufferString(payload),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	return recorder
}

func TestSaveCustomerInternetCredentialRunsManagedCoordinator(
	t *testing.T,
) {
	db := setupCustomerInternetCredentialHandlerTestDB(t)
	customer, _, subscription :=
		createCustomerInternetCredentialHandlerFixture(t, db)

	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	calls := 0

	customerInternetCredentialPostSaveReconciler = func(
		gotSubscription *models.Subscription,
		keyMaterial string,
		identityChanged bool,
		oldRouterID uint,
		oldUsername string,
	) (
		services.CustomerInternetCredentialReconciliationResult,
		error,
	) {
		calls++

		if gotSubscription.ID != subscription.ID {
			t.Fatalf(
				"subscription = %d, want %d",
				gotSubscription.ID,
				subscription.ID,
			)
		}

		if !identityChanged {
			t.Fatal("expected identityChanged=true")
		}

		if oldRouterID != 7 ||
			oldUsername != "old-pppoe-user" {
			t.Fatalf(
				"unexpected old identity: router=%d username=%q",
				oldRouterID,
				oldUsername,
			)
		}

		return services.CustomerInternetCredentialReconciliationResult{
			PPPAttempted: true,
			FTPAttempted: true,
			FTP: services.ManagedFTPReconciliationResult{
				SubscriptionID: subscription.ID,
				Action:         "PASSWORD_SYNC_AND_UNLOCK",
				Executed:       true,
			},
		}, nil
	}

	recorder := performCustomerInternetCredentialSave(
		t,
		cfg,
		customer.ID,
		`{
			"router_id":8,
			"pppoe_username":"new-pppoe-user",
			"pppoe_password":"new-secret"
		}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want 200; body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if calls != 1 {
		t.Fatalf("coordinator calls = %d, want 1", calls)
	}

	var body map[string]any
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	if _, ok := body["pppoe_reconciliation"]; !ok {
		t.Fatal("PPP reconciliation missing")
	}

	if _, ok := body["ftp_reconciliation"]; !ok {
		t.Fatal("FTP reconciliation missing")
	}
}

func TestSaveCustomerInternetCredentialPPPFailureStillReturnsFTPResult(
	t *testing.T,
) {
	db := setupCustomerInternetCredentialHandlerTestDB(t)
	customer, _, subscription :=
		createCustomerInternetCredentialHandlerFixture(t, db)

	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	customerInternetCredentialPostSaveReconciler = func(
		*models.Subscription,
		string,
		bool,
		uint,
		string,
	) (
		services.CustomerInternetCredentialReconciliationResult,
		error,
	) {
		return services.CustomerInternetCredentialReconciliationResult{
			PPPAttempted: true,
			PPPError:     "router unavailable",
			FTPAttempted: true,
			FTP: services.ManagedFTPReconciliationResult{
				SubscriptionID: subscription.ID,
				Action:         "PASSWORD_SYNC_AND_UNLOCK",
				Executed:       true,
			},
		}, nil
	}

	recorder := performCustomerInternetCredentialSave(
		t,
		cfg,
		customer.ID,
		`{
			"router_id":7,
			"pppoe_username":"old-pppoe-user",
			"pppoe_password":"rotated-secret"
		}`,
	)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf(
			"status = %d, want 502; body=%s",
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
		"router unavailable" {
		t.Fatalf(
			"PPP error = %#v",
			body["pppoe_reconciliation_error"],
		)
	}

	if _, ok := body["ftp_reconciliation"]; !ok {
		t.Fatal(
			"FTP reconciliation missing after PPP failure",
		)
	}

	if _, ok := body["internet_credential"]; !ok {
		t.Fatal(
			"saved internet credential missing from failure response",
		)
	}
}

func TestSaveCustomerInternetCredentialFTPFailurePreservesPPPSuccess(
	t *testing.T,
) {
	db := setupCustomerInternetCredentialHandlerTestDB(t)
	customer, _, _ :=
		createCustomerInternetCredentialHandlerFixture(t, db)

	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	customerInternetCredentialPostSaveReconciler = func(
		*models.Subscription,
		string,
		bool,
		uint,
		string,
	) (
		services.CustomerInternetCredentialReconciliationResult,
		error,
	) {
		return services.CustomerInternetCredentialReconciliationResult{
			PPPAttempted: true,
			FTPAttempted: true,
			FTPError:     "FTP server unavailable",
		}, nil
	}

	recorder := performCustomerInternetCredentialSave(
		t,
		cfg,
		customer.ID,
		`{
			"router_id":7,
			"pppoe_username":"old-pppoe-user",
			"pppoe_password":"rotated-secret"
		}`,
	)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf(
			"status = %d, want 502; body=%s",
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

	if body["ftp_reconciliation_error"] !=
		"FTP server unavailable" {
		t.Fatalf(
			"FTP error = %#v",
			body["ftp_reconciliation_error"],
		)
	}

	if _, ok := body["pppoe_reconciliation"]; !ok {
		t.Fatal(
			"PPP reconciliation missing after FTP failure",
		)
	}
}

func TestSaveCustomerInternetCredentialCoordinatorBoundaryFailure(
	t *testing.T,
) {
	db := setupCustomerInternetCredentialHandlerTestDB(t)
	customer, _, _ :=
		createCustomerInternetCredentialHandlerFixture(t, db)

	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	customerInternetCredentialPostSaveReconciler = func(
		*models.Subscription,
		string,
		bool,
		uint,
		string,
	) (
		services.CustomerInternetCredentialReconciliationResult,
		error,
	) {
		return services.CustomerInternetCredentialReconciliationResult{},
			errors.New("coordinator unavailable")
	}

	recorder := performCustomerInternetCredentialSave(
		t,
		cfg,
		customer.ID,
		`{
			"router_id":7,
			"pppoe_username":"old-pppoe-user",
			"pppoe_password":"rotated-secret"
		}`,
	)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf(
			"status = %d, want 502; body=%s",
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

	if _, ok := body["internet_credential"]; !ok {
		t.Fatal(
			"saved credential missing after coordinator failure",
		)
	}
}
