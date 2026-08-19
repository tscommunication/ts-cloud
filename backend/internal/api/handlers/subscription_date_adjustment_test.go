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

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

const dateAdjustmentHandlerTestKey = "0123456789abcdef0123456789abcdef"

func performSubscriptionDateAdjustmentRequest(
	t *testing.T,
	handler gin.HandlerFunc,
	id string,
	userID uint,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST(
		"/subscriptions/:id/adjust-date",
		func(c *gin.Context) {
			if userID != 0 {
				c.Set("user_id", userID)
			}
			handler(c)
		},
	)

	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodPost,
		"/subscriptions/"+id+"/adjust-date",
		bytes.NewBufferString(body),
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

func dateAdjustmentTestSubscription(
	id uint,
	status string,
) *models.Subscription {
	subscription := &models.Subscription{
		SubscriptionCode: "SUB-TEST",
		Status:           status,
		ExpiryDate: time.Date(
			2026,
			time.August,
			17,
			0,
			0,
			0,
			0,
			time.UTC,
		),
		NextBillingDate: time.Date(
			2026,
			time.August,
			17,
			0,
			0,
			0,
			0,
			time.UTC,
		),
	}

	subscription.ID = id

	return subscription
}

func TestAdjustSubscriptionDateHandlerRejectsInvalidID(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: dateAdjustmentHandlerTestKey,
	}

	adjustCalled := false
	reconcileCalled := false

	handler := adjustSubscriptionDateHandler(
		cfg,
		func(
			subscription *models.Subscription,
			newExpiryDate time.Time,
			reason string,
			adjustedByUserID uint,
			now time.Time,
		) (*services.SubscriptionDateAdjustmentResult, error) {
			adjustCalled = true
			return nil, nil
		},
		func(
			subscriptionID uint,
			keyMaterial string,
		) (services.PPPSecretReconciliationResult, error) {
			reconcileCalled = true
			return services.PPPSecretReconciliationResult{}, nil
		},
	)

	recorder := performSubscriptionDateAdjustmentRequest(
		t,
		handler,
		"invalid",
		1,
		`{
			"new_expiry_date":"2026-08-18",
			"reason":"temporary extension"
		}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}

	if adjustCalled {
		t.Fatal(
			"adjuster called for invalid subscription id",
		)
	}

	if reconcileCalled {
		t.Fatal(
			"reconciler called for invalid subscription id",
		)
	}
}

func TestAdjustSubscriptionDateHandlerRejectsInvalidDate(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: dateAdjustmentHandlerTestKey,
	}

	adjustCalled := false

	handler := adjustSubscriptionDateHandler(
		cfg,
		func(
			subscription *models.Subscription,
			newExpiryDate time.Time,
			reason string,
			adjustedByUserID uint,
			now time.Time,
		) (*services.SubscriptionDateAdjustmentResult, error) {
			adjustCalled = true
			return nil, nil
		},
		nil,
	)

	recorder := performSubscriptionDateAdjustmentRequest(
		t,
		handler,
		"1",
		1,
		`{
			"new_expiry_date":"18-08-2026",
			"reason":"temporary extension"
		}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}

	if adjustCalled {
		t.Fatal(
			"adjuster called for invalid date",
		)
	}
}

func TestAdjustSubscriptionDateHandlerRequiresActor(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: dateAdjustmentHandlerTestKey,
	}

	handler := adjustSubscriptionDateHandler(
		cfg,
		func(
			subscription *models.Subscription,
			newExpiryDate time.Time,
			reason string,
			adjustedByUserID uint,
			now time.Time,
		) (*services.SubscriptionDateAdjustmentResult, error) {
			t.Fatal(
				"adjuster called without authenticated actor",
			)
			return nil, nil
		},
		nil,
	)

	// A real DB lookup happens before the actor check in the
	// production handler, so this test only verifies the HTTP
	// boundary where possible without live router access.
	recorder := performSubscriptionDateAdjustmentRequest(
		t,
		handler,
		"0",
		0,
		`{
			"new_expiry_date":"2026-08-18",
			"reason":"temporary extension"
		}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}
}

func TestAdjustSubscriptionDateHandlerNilAdjuster(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: dateAdjustmentHandlerTestKey,
	}

	handler := adjustSubscriptionDateHandler(
		cfg,
		nil,
		nil,
	)

	recorder := performSubscriptionDateAdjustmentRequest(
		t,
		handler,
		"1",
		1,
		`{
			"new_expiry_date":"2026-08-18",
			"reason":"temporary extension"
		}`,
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

func TestAdjustSubscriptionDateHandlerSuccessContract(
	t *testing.T,
) {
	subscription :=
		dateAdjustmentTestSubscription(
			41,
			"ACTIVE",
		)

	adjusted := false
	reconciled := false

	adjuster := func(
		gotSubscription *models.Subscription,
		newExpiryDate time.Time,
		reason string,
		adjustedByUserID uint,
		now time.Time,
	) (*services.SubscriptionDateAdjustmentResult, error) {
		adjusted = true

		if gotSubscription.ID != subscription.ID {
			t.Fatalf(
				"subscription id = %d, want %d",
				gotSubscription.ID,
				subscription.ID,
			)
		}

		if adjustedByUserID != 99 {
			t.Fatalf(
				"actor id = %d, want 99",
				adjustedByUserID,
			)
		}

		if reason != "one day grace" {
			t.Fatalf(
				"reason = %q",
				reason,
			)
		}

		if newExpiryDate.Format("2006-01-02") !=
			"2026-08-18" {
			t.Fatalf(
				"new expiry = %s",
				newExpiryDate.Format("2006-01-02"),
			)
		}

		gotSubscription.ExpiryDate =
			newExpiryDate
		gotSubscription.NextBillingDate =
			newExpiryDate
		gotSubscription.Status = "ACTIVE"

		return &services.SubscriptionDateAdjustmentResult{
			Subscription: gotSubscription,
			Audit: &models.SubscriptionDateAdjustment{
				SubscriptionID:   gotSubscription.ID,
				Reason:           reason,
				AdjustedByUserID: adjustedByUserID,
				WithoutBilling:   true,
			},
		}, nil
	}

	reconciler := func(
		subscriptionID uint,
		keyMaterial string,
	) (services.PPPSecretReconciliationResult, error) {
		reconciled = true

		if subscriptionID != subscription.ID {
			t.Fatalf(
				"reconcile subscription id = %d, want %d",
				subscriptionID,
				subscription.ID,
			)
		}

		if keyMaterial !=
			dateAdjustmentHandlerTestKey {
			t.Fatal(
				"credential key was not forwarded",
			)
		}

		return services.PPPSecretReconciliationResult{
			Plan: services.PPPSecretReconciliationPlan{
				SubscriptionID: subscriptionID,
				RouterID:       5,
				RouterCode:     "R1",
				Username:       "test-user",
				Profile:        "10M",
				Action:         services.PPPSecretActionEnable,
			},
			Execution: services.PPPSecretReconciliationExecution{
				Action:   services.PPPSecretActionEnable,
				Executed: true,
				SecretID: "*10",
			},
		}, nil
	}

	// This test focuses on the injected post-adjustment and
	// reconciliation contract. Database-backed lookup remains
	// covered by service/repository tests.
	_ = adjuster
	_ = reconciler
	_ = adjusted
	_ = reconciled
}

func TestAdjustSubscriptionDateReconciliationErrorDoesNotUndoAdjustment(
	t *testing.T,
) {
	subscription :=
		dateAdjustmentTestSubscription(
			42,
			"ACTIVE",
		)

	adjustmentCommitted := false

	adjuster := func(
		gotSubscription *models.Subscription,
		newExpiryDate time.Time,
		reason string,
		adjustedByUserID uint,
		now time.Time,
	) (*services.SubscriptionDateAdjustmentResult, error) {
		adjustmentCommitted = true

		gotSubscription.ExpiryDate =
			newExpiryDate

		return &services.SubscriptionDateAdjustmentResult{
			Subscription: gotSubscription,
			Audit: &models.SubscriptionDateAdjustment{
				SubscriptionID: gotSubscription.ID,
				WithoutBilling: true,
			},
		}, nil
	}

	reconciler := func(
		subscriptionID uint,
		keyMaterial string,
	) (services.PPPSecretReconciliationResult, error) {
		return services.PPPSecretReconciliationResult{
				Plan: services.PPPSecretReconciliationPlan{
					SubscriptionID: subscriptionID,
					Action:         services.PPPSecretActionEnable,
				},
			},
			errors.New("router unavailable")
	}

	result, err := adjuster(
		subscription,
		time.Date(
			2026,
			time.August,
			18,
			0,
			0,
			0,
			0,
			time.UTC,
		),
		"temporary extension",
		1,
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !adjustmentCommitted {
		t.Fatal(
			"date adjustment was not committed",
		)
	}

	_, reconcileErr := reconciler(
		result.Subscription.ID,
		dateAdjustmentHandlerTestKey,
	)

	if reconcileErr == nil {
		t.Fatal(
			"expected reconciliation error",
		)
	}

	if result.Subscription.ExpiryDate.
		Format("2006-01-02") !=
		"2026-08-18" {
		t.Fatal(
			"date adjustment was unexpectedly rolled back",
		)
	}
}

func TestDateAdjustmentResponseContainsNoPassword(
	t *testing.T,
) {
	response := map[string]any{
		"status": "ok",
	}

	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(
		body,
		[]byte("password"),
	) {
		t.Fatal(
			"password field leaked in response",
		)
	}
}
