package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

func performCreatePaymentHandlerRequest(
	t *testing.T,
	handler gin.HandlerFunc,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST(
		"/payments",
		func(c *gin.Context) {
			c.Set("user_id", uint(91))
			c.Set("role", "superadmin")
			handler(c)
		},
	)

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodPost,
		"/payments",
		bytes.NewReader(raw),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	router.ServeHTTP(
		recorder,
		request,
	)

	return recorder
}

func paymentCreateTestResult(
	renewed bool,
	subscriptionID uint,
) services.CreatePaymentResult {
	result := services.CreatePaymentResult{
		Payment: &models.Payment{
			InvoiceID:      1,
			CustomerID:     2,
			SubscriptionID: subscriptionID,
			PaymentDate: time.Date(
				2026, time.August, 19,
				15, 0, 0, 0,
				time.UTC,
			),
			Amount: 500,
			Method: "CASH",
			Status: "SUCCESS",
		},
		Renewal: services.PaymentRenewalResult{
			Renewed: renewed,
		},
	}

	result.Payment.ID = 77

	if renewed {
		result.Renewal.Renewal =
			&models.SubscriptionRenewal{
				PaymentID:      result.Payment.ID,
				SubscriptionID: subscriptionID,
			}
	}

	return result
}

func TestCreatePaymentHandlerReconcilesRenewedSubscription(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	loaderCalls := 0
	reconcileCalls := 0

	creator := func(
		payment *models.Payment,
	) (services.CreatePaymentResult, error) {
		result := paymentCreateTestResult(
			true,
			501,
		)

		*payment = *result.Payment

		return result, nil
	}

	loader := func(
		id uint,
	) (*models.Subscription, error) {
		loaderCalls++

		if id != 501 {
			t.Fatalf(
				"subscription id = %d, want 501",
				id,
			)
		}

		subscription := &models.Subscription{
			Status: "ACTIVE",
		}
		subscription.ID = id

		return subscription, nil
	}

	reconciler := func(
		subscription *models.Subscription,
		action services.SubscriptionLifecycleAction,
		keyMaterial string,
	) (services.SubscriptionLifecycleReconciliationResult, error) {
		reconcileCalls++

		if subscription.ID != 501 {
			t.Fatalf(
				"subscription id = %d, want 501",
				subscription.ID,
			)
		}

		if action !=
			services.SubscriptionLifecycleRenew {
			t.Fatalf(
				"action = %q, want RENEW",
				action,
			)
		}

		if keyMaterial != cfg.CredentialKey {
			t.Fatal("credential key was not forwarded")
		}

		return services.SubscriptionLifecycleReconciliationResult{
			Action:         action,
			SubscriptionID: subscription.ID,
			Status:         subscription.Status,
		}, nil
	}

	handler := createPaymentHandler(
		cfg,
		creator,
		func(id uint) (*models.Invoice, error) {
			return &models.Invoice{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Payment, error) {
			return &models.Payment{
				Model: gorm.Model{ID: id},
			}, nil
		},
		loader,
		reconciler,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   1,
			"payment_date": "2026-08-19",
			"amount":       500,
			"method":       "CASH",
		},
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusCreated,
			recorder.Body.String(),
		)
	}

	if loaderCalls != 1 {
		t.Fatalf(
			"loader calls = %d, want 1",
			loaderCalls,
		)
	}

	if reconcileCalls != 1 {
		t.Fatalf(
			"reconcile calls = %d, want 1",
			reconcileCalls,
		)
	}

	var body map[string]any

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	if _, ok := body["payment"]; !ok {
		t.Fatal("payment response missing")
	}

	if _, ok := body["renewal"]; !ok {
		t.Fatal("renewal response missing")
	}

	if _, ok := body["pppoe_reconciliation"]; !ok {
		t.Fatal("PPP reconciliation response missing")
	}
}

func TestCreatePaymentHandlerNoRenewalDoesNotReconcile(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	reconcileCalls := 0

	creator := func(
		payment *models.Payment,
	) (services.CreatePaymentResult, error) {
		result := paymentCreateTestResult(
			false,
			502,
		)

		*payment = *result.Payment

		return result, nil
	}

	reconciler := func(
		subscription *models.Subscription,
		action services.SubscriptionLifecycleAction,
		keyMaterial string,
	) (services.SubscriptionLifecycleReconciliationResult, error) {
		reconcileCalls++

		return services.SubscriptionLifecycleReconciliationResult{},
			nil
	}

	handler := createPaymentHandler(
		cfg,
		creator,
		func(id uint) (*models.Invoice, error) {
			return &models.Invoice{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Payment, error) {
			return &models.Payment{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Subscription, error) {
			t.Fatal("loader called for non-renewal payment")
			return nil, nil
		},
		reconciler,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   1,
			"payment_date": "2026-08-19",
			"amount":       200,
			"method":       "CASH",
		},
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusCreated,
			recorder.Body.String(),
		)
	}

	if reconcileCalls != 0 {
		t.Fatalf(
			"reconcile calls = %d, want 0",
			reconcileCalls,
		)
	}
}

func TestCreatePaymentHandlerCreatorFailureDoesNotReconcile(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	reconcileCalls := 0

	creator := func(
		payment *models.Payment,
	) (services.CreatePaymentResult, error) {
		return services.CreatePaymentResult{},
			errors.New("payment transaction failed")
	}

	reconciler := func(
		subscription *models.Subscription,
		action services.SubscriptionLifecycleAction,
		keyMaterial string,
	) (services.SubscriptionLifecycleReconciliationResult, error) {
		reconcileCalls++

		return services.SubscriptionLifecycleReconciliationResult{},
			nil
	}

	handler := createPaymentHandler(
		cfg,
		creator,
		func(id uint) (*models.Invoice, error) {
			return &models.Invoice{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Payment, error) {
			return &models.Payment{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Subscription, error) {
			t.Fatal("loader called after creator failure")
			return nil, nil
		},
		reconciler,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   1,
			"payment_date": "2026-08-19",
			"amount":       500,
			"method":       "CASH",
		},
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}

	if reconcileCalls != 0 {
		t.Fatalf(
			"reconcile calls = %d, want 0",
			reconcileCalls,
		)
	}
}

func TestCreatePaymentHandlerRejectsNilCreator(
	t *testing.T,
) {
	cfg := &config.Config{}

	handler := createPaymentHandler(
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   1,
			"payment_date": "2026-08-19",
			"amount":       500,
			"method":       "CASH",
		},
	)

	if recorder.Code !=
		http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}
}

func TestCreatePaymentHandlerReconciliationFailurePreservesCreatedPayment(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: "0123456789abcdef0123456789abcdef",
	}

	creator := func(
		payment *models.Payment,
	) (services.CreatePaymentResult, error) {
		result := paymentCreateTestResult(
			true,
			503,
		)

		*payment = *result.Payment

		return result, nil
	}

	loader := func(
		id uint,
	) (*models.Subscription, error) {
		subscription := &models.Subscription{
			Status: "ACTIVE",
		}
		subscription.ID = id

		return subscription, nil
	}

	reconciler := func(
		subscription *models.Subscription,
		action services.SubscriptionLifecycleAction,
		keyMaterial string,
	) (services.SubscriptionLifecycleReconciliationResult, error) {
		return services.SubscriptionLifecycleReconciliationResult{
			Action:         action,
			SubscriptionID: subscription.ID,
			Status:         subscription.Status,
		}, errors.New("router unavailable")
	}

	handler := createPaymentHandler(
		cfg,
		creator,
		func(id uint) (*models.Invoice, error) {
			return &models.Invoice{
				Model: gorm.Model{ID: id},
			}, nil
		},
		func(id uint) (*models.Payment, error) {
			return &models.Payment{
				Model: gorm.Model{ID: id},
			}, nil
		},
		loader,
		reconciler,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   1,
			"payment_date": "2026-08-19",
			"amount":       500,
			"method":       "CASH",
		},
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusCreated,
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

	if _, ok := body["payment"]; !ok {
		t.Fatal("payment response missing after reconciliation failure")
	}

	errorValue, ok :=
		body["pppoe_reconciliation_error"]

	if !ok {
		t.Fatal("PPP reconciliation error missing")
	}

	if errorValue != "router unavailable" {
		t.Fatalf(
			"reconciliation error = %v, want router unavailable",
			errorValue,
		)
	}
}

func TestCreatePaymentHandlerStableJSONContract(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: "test-key",
	}

	creator := func(
		payment *models.Payment,
	) (services.CreatePaymentResult, error) {
		renewal := &models.SubscriptionRenewal{
			PaymentID:      77,
			InvoiceID:      payment.InvoiceID,
			CustomerID:     21,
			SubscriptionID: 31,
			Source:         "PAYMENT",
		}

		result := services.CreatePaymentResult{
			Payment: payment,
			Renewal: services.PaymentRenewalResult{
				Renewed: true,
				Reason:  "invoice paid",
				Renewal: renewal,
			},
		}

		payment.ID = 77
		payment.CustomerID = 21
		payment.SubscriptionID = 31

		return result, nil
	}

	invoiceLoader := func(
		id uint,
	) (*models.Invoice, error) {
		return &models.Invoice{
			Model: gorm.Model{ID: id},
		}, nil
	}

	paymentLoader := func(
		id uint,
	) (*models.Payment, error) {
		return &models.Payment{
			Model:          gorm.Model{ID: id},
			InvoiceID:      11,
			CustomerID:     21,
			SubscriptionID: 31,
			Amount:         500,
			Method:         "CASH",
			Status:         "SUCCESS",
		}, nil
	}

	subscriptionLoader := func(
		id uint,
	) (*models.Subscription, error) {
		return &models.Subscription{
			Model:  gorm.Model{ID: id},
			Status: "ACTIVE",
		}, nil
	}

	reconciler := func(
		subscription *models.Subscription,
		action services.SubscriptionLifecycleAction,
		keyMaterial string,
	) (
		services.SubscriptionLifecycleReconciliationResult,
		error,
	) {
		return services.SubscriptionLifecycleReconciliationResult{
			Action:                  action,
			SubscriptionID:          subscription.ID,
			Status:                  subscription.Status,
			ReconciliationAttempted: true,
		}, nil
	}

	handler := createPaymentHandler(
		cfg,
		creator,
		invoiceLoader,
		paymentLoader,
		subscriptionLoader,
		reconciler,
	)

	recorder := performCreatePaymentHandlerRequest(
		t,
		handler,
		map[string]any{
			"invoice_id":   11,
			"payment_date": "2026-08-19",
			"amount":       500,
			"method":       "CASH",
		},
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusCreated,
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

	renewalRaw, ok := body["renewal"]
	if !ok {
		t.Fatal("renewal response missing")
	}

	renewal, ok := renewalRaw.(map[string]any)
	if !ok {
		t.Fatalf(
			"renewal response has unexpected type %T",
			renewalRaw,
		)
	}

	if renewed, ok := renewal["renewed"].(bool); !ok || !renewed {
		t.Fatalf(
			"expected renewal.renewed=true, got %#v",
			renewal["renewed"],
		)
	}

	if got := renewal["reason"]; got != "invoice paid" {
		t.Fatalf(
			"expected stable renewal reason, got %#v",
			got,
		)
	}

	if _, exists := renewal["Renewed"]; exists {
		t.Fatal("legacy Go field name Renewed leaked into JSON")
	}

	if _, exists := renewal["Reason"]; exists {
		t.Fatal("legacy Go field name Reason leaked into JSON")
	}

	if _, exists := renewal["Renewal"]; exists {
		t.Fatal("legacy Go field name Renewal leaked into JSON")
	}

	reconciliationRaw, ok :=
		body["pppoe_reconciliation"]
	if !ok {
		t.Fatal("pppoe_reconciliation response missing")
	}

	reconciliation, ok :=
		reconciliationRaw.(map[string]any)
	if !ok {
		t.Fatalf(
			"pppoe_reconciliation has unexpected type %T",
			reconciliationRaw,
		)
	}

	if got := reconciliation["subscription_id"]; got != float64(31) {
		t.Fatalf(
			"expected subscription_id 31, got %#v",
			got,
		)
	}

	if attempted, ok :=
		reconciliation["reconciliation_attempted"].(bool); !ok || !attempted {
		t.Fatalf(
			"expected reconciliation_attempted=true, got %#v",
			reconciliation["reconciliation_attempted"],
		)
	}

	for _, forbidden := range []string{
		"Action",
		"SubscriptionID",
		"ReconciliationAttempted",
		"Reconciliation",
		"ReconciliationError",
	} {
		if _, exists := reconciliation[forbidden]; exists {
			t.Fatalf(
				"legacy Go field name %q leaked into JSON",
				forbidden,
			)
		}
	}
}
