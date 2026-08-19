package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/services"
)

const reconciliationHandlerTestKey = "0123456789abcdef0123456789abcdef"

func performPPPReconciliationHandlerRequest(
	t *testing.T,
	handler gin.HandlerFunc,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST(
		"/subscriptions/:id/reconcile-pppoe",
		handler,
	)

	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodPost,
		"/subscriptions/"+id+"/reconcile-pppoe",
		nil,
	)

	router.ServeHTTP(
		recorder,
		request,
	)

	return recorder
}

func TestReconcileSubscriptionPPPSecretHandlerRejectsInvalidID(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: reconciliationHandlerTestKey,
	}

	called := false

	handler := reconcileSubscriptionPPPSecretHandler(
		cfg,
		func(
			subscriptionID uint,
			keyMaterial string,
		) (services.PPPSecretReconciliationResult, error) {
			called = true

			return services.PPPSecretReconciliationResult{},
				nil
		},
	)

	recorder := performPPPReconciliationHandlerRequest(
		t,
		handler,
		"invalid",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusBadRequest,
		)
	}

	if called {
		t.Fatal(
			"runner was called for invalid subscription id",
		)
	}

	var body map[string]any

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if body["error"] !=
		"Invalid subscription ID" {
		t.Fatalf(
			"error = %v",
			body["error"],
		)
	}
}

func TestReconcileSubscriptionPPPSecretHandlerRejectsNilRunner(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: reconciliationHandlerTestKey,
	}

	handler := reconcileSubscriptionPPPSecretHandler(
		cfg,
		nil,
	)

	recorder := performPPPReconciliationHandlerRequest(
		t,
		handler,
		"25",
	)

	if recorder.Code !=
		http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	var body map[string]any

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if body["error"] !=
		"PPP reconciliation runner is not configured" {
		t.Fatalf(
			"error = %v",
			body["error"],
		)
	}
}

func TestReconcileSubscriptionPPPSecretHandlerSuccess(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: reconciliationHandlerTestKey,
	}

	var (
		gotSubscriptionID uint
		gotKeyMaterial    string
	)

	handler := reconcileSubscriptionPPPSecretHandler(
		cfg,
		func(
			subscriptionID uint,
			keyMaterial string,
		) (services.PPPSecretReconciliationResult, error) {
			gotSubscriptionID = subscriptionID
			gotKeyMaterial = keyMaterial

			return services.PPPSecretReconciliationResult{
				Plan: services.PPPSecretReconciliationPlan{
					SubscriptionID: subscriptionID,
					RouterID:       7,
					RouterCode:     "RTR-007",
					Username:       "subscriber-25",
					Profile:        "Go_P25",
					Action: services.
						PPPSecretActionCreate,
					Reason: "RouterOS PPP secret does not exist",
				},
				Execution: services.
					PPPSecretReconciliationExecution{
					Action: services.
						PPPSecretActionCreate,
					Executed: true,
					Reason:   "RouterOS PPP secret does not exist",
					SecretID: "*55",
				},
			}, nil
		},
	)

	recorder := performPPPReconciliationHandlerRequest(
		t,
		handler,
		"25",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	if gotSubscriptionID != 25 {
		t.Fatalf(
			"subscription id = %d, want 25",
			gotSubscriptionID,
		)
	}

	if gotKeyMaterial != reconciliationHandlerTestKey {
		t.Fatal(
			"handler did not pass credential key to runner",
		)
	}

	var body subscriptionPPPSecretReconciliationResponse

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if body.SubscriptionID != 25 {
		t.Fatalf(
			"subscription id = %d",
			body.SubscriptionID,
		)
	}

	if body.RouterID != 7 {
		t.Fatalf(
			"router id = %d",
			body.RouterID,
		)
	}

	if body.RouterCode != "RTR-007" {
		t.Fatalf(
			"router code = %q",
			body.RouterCode,
		)
	}

	if body.Username != "subscriber-25" {
		t.Fatalf(
			"username = %q",
			body.Username,
		)
	}

	if body.Profile != "Go_P25" {
		t.Fatalf(
			"profile = %q",
			body.Profile,
		)
	}

	if body.Action != "CREATE" {
		t.Fatalf(
			"action = %q",
			body.Action,
		)
	}

	if !body.Executed {
		t.Fatal(
			"expected reconciliation execution",
		)
	}

	if body.SecretID != "*55" {
		t.Fatalf(
			"secret id = %q",
			body.SecretID,
		)
	}
}

func TestReconcileSubscriptionPPPSecretHandlerFallsBackToPlanFields(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: reconciliationHandlerTestKey,
	}

	handler := reconcileSubscriptionPPPSecretHandler(
		cfg,
		func(
			subscriptionID uint,
			keyMaterial string,
		) (services.PPPSecretReconciliationResult, error) {
			return services.PPPSecretReconciliationResult{
				Plan: services.PPPSecretReconciliationPlan{
					SubscriptionID: subscriptionID,
					RouterID:       8,
					RouterCode:     "RTR-008",
					Username:       "subscriber-noop",
					Profile:        "Go_P50",
					Action: services.
						PPPSecretActionNoop,
					Reason: "RouterOS PPP secret already matches subscription state",
					CurrentSecret: &mikrotik.PPPSecret{
						ID:      "*88",
						Name:    "subscriber-noop",
						Service: "pppoe",
						Profile: "Go_P50",
					},
				},
				Execution: services.
					PPPSecretReconciliationExecution{
					Executed: false,
				},
			}, nil
		},
	)

	recorder := performPPPReconciliationHandlerRequest(
		t,
		handler,
		"31",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	var body subscriptionPPPSecretReconciliationResponse

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if body.Action != "NOOP" {
		t.Fatalf(
			"action = %q, want NOOP",
			body.Action,
		)
	}

	if body.Reason !=
		"RouterOS PPP secret already matches subscription state" {
		t.Fatalf(
			"reason = %q",
			body.Reason,
		)
	}

	if body.Executed {
		t.Fatal(
			"NOOP response must not report execution",
		)
	}

	if body.SecretID != "" {
		t.Fatalf(
			"secret id = %q, want empty",
			body.SecretID,
		)
	}
}

func TestReconcileSubscriptionPPPSecretHandlerReturnsReconciliationOnError(
	t *testing.T,
) {
	cfg := &config.Config{
		CredentialKey: reconciliationHandlerTestKey,
	}

	handler := reconcileSubscriptionPPPSecretHandler(
		cfg,
		func(
			subscriptionID uint,
			keyMaterial string,
		) (services.PPPSecretReconciliationResult, error) {
			return services.PPPSecretReconciliationResult{
					Plan: services.PPPSecretReconciliationPlan{
						SubscriptionID: subscriptionID,
						RouterID:       9,
						RouterCode:     "RTR-009",
						Username:       "subscriber-conflict",
						Profile:        "Go_P25",
						Action: services.
							PPPSecretActionConflict,
						Reason: "multiple RouterOS PPP secrets match subscription username",
					},
					Execution: services.
						PPPSecretReconciliationExecution{
						Action: services.
							PPPSecretActionConflict,
						Executed: false,
						Reason:   "multiple RouterOS PPP secrets match subscription username",
					},
				},
				errors.New(
					"execute PPP secret reconciliation plan: PPP secret reconciliation conflict",
				)
		},
	)

	recorder := performPPPReconciliationHandlerRequest(
		t,
		handler,
		"42",
	)

	if recorder.Code !=
		http.StatusUnprocessableEntity {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusUnprocessableEntity,
			recorder.Body.String(),
		)
	}

	var body struct {
		Error          string                                      `json:"error"`
		Reconciliation subscriptionPPPSecretReconciliationResponse `json:"reconciliation"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if body.Error == "" {
		t.Fatal(
			"expected reconciliation error",
		)
	}

	if body.Reconciliation.Action != "CONFLICT" {
		t.Fatalf(
			"action = %q",
			body.Reconciliation.Action,
		)
	}

	if body.Reconciliation.Executed {
		t.Fatal(
			"conflict must not report mutation",
		)
	}

	if body.Reconciliation.SubscriptionID != 42 {
		t.Fatalf(
			"subscription id = %d",
			body.Reconciliation.SubscriptionID,
		)
	}

	if body.Reconciliation.RouterCode != "RTR-009" {
		t.Fatalf(
			"router code = %q",
			body.Reconciliation.RouterCode,
		)
	}
}
