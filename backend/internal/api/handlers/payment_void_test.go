package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
	"gorm.io/gorm"
)

func performPaymentVoidRequest(
	t *testing.T,
	handler gin.HandlerFunc,
	id string,
	body string,
	actorID uint,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST(
		"/payments/:id/void",
		func(c *gin.Context) {
			if actorID != 0 {
				c.Set("user_id", actorID)
			}
		},
		handler,
	)

	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodPost,
		"/payments/"+id+"/void",
		strings.NewReader(body),
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

func TestVoidPaymentHandlerRejectsInvalidID(
	t *testing.T,
) {
	called := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			called = true
			return nil, nil
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"invalid",
		`{"reason":"duplicate recharge"}`,
		99,
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
			"void runner called for invalid payment id",
		)
	}
}

func TestVoidPaymentHandlerRequiresReason(
	t *testing.T,
) {
	called := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			called = true
			return nil, nil
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{}`,
		99,
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
			"void runner called without reason",
		)
	}
}

func TestVoidPaymentHandlerRequiresActor(
	t *testing.T,
) {
	called := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			called = true
			return nil, nil
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		0,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusUnauthorized,
		)
	}

	if called {
		t.Fatal(
			"void runner called without actor",
		)
	}
}

func TestVoidPaymentHandlerRejectsNilRunner(
	t *testing.T,
) {
	handler := voidPaymentHandler(nil)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
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

func TestVoidPaymentHandlerSuccessContract(
	t *testing.T,
) {
	called := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			called = true

			if id != 41 {
				t.Fatalf(
					"id = %d, want 41",
					id,
				)
			}

			if reason != "duplicate recharge" {
				t.Fatalf(
					"reason = %q",
					reason,
				)
			}

			if voidedByUserID != 99 {
				t.Fatalf(
					"actor = %d, want 99",
					voidedByUserID,
				)
			}

			return &models.PaymentVoidAudit{
				PaymentID:      id,
				InvoiceID:      12,
				CustomerID:     7,
				SubscriptionID: 9,
				ReceiptNo:      "RCPT-TEST",
				Amount:         500,
				PreviousStatus: "SUCCESS",
				NewStatus:      "VOID",
				Reason:         reason,
				VoidedByUserID: voidedByUserID,
				VoidedAt:       now,
			}, nil
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	if !called {
		t.Fatal("void runner was not called")
	}

	var body map[string]any

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	if body["message"] !=
		"Payment voided successfully" {
		t.Fatalf(
			"unexpected message: %v",
			body["message"],
		)
	}

	audit, ok := body["audit"].(map[string]any)
	if !ok {
		t.Fatalf(
			"missing audit response: %+v",
			body,
		)
	}

	if audit["Reason"] != "duplicate recharge" {
		t.Fatalf(
			"audit reason = %v",
			audit["Reason"],
		)
	}
}

func TestVoidPaymentHandlerServiceFailure(
	t *testing.T,
) {
	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			return nil,
				errors.New("payment is already voided")
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusConflict,
		)
	}
}

func TestVoidPaymentHandlerReconcilesPPPAfterRenewalReversal(
	t *testing.T,
) {
	reconcilerCalled := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			return &models.PaymentVoidAudit{
				PaymentID:      id,
				SubscriptionID: 9,
				Reason:         reason,
			}, nil
		},
		paymentVoidPostCommitDependencies{
			reversalLoader: func(
				paymentID uint,
			) (*models.SubscriptionRenewalReversal, bool, error) {
				if paymentID != 41 {
					t.Fatalf(
						"paymentID = %d, want 41",
						paymentID,
					)
				}

				return &models.SubscriptionRenewalReversal{
					PaymentID:      paymentID,
					SubscriptionID: 9,
				}, true, nil
			},
			subscriptionLoader: func(
				id uint,
			) (*models.Subscription, error) {
				if id != 9 {
					t.Fatalf(
						"subscription id = %d, want 9",
						id,
					)
				}

				return &models.Subscription{
					Model:  gorm.Model{ID: id},
					Status: "EXPIRED",
				}, nil
			},
			reconciler: func(
				subscription *models.Subscription,
				action services.SubscriptionLifecycleAction,
				keyMaterial string,
			) (services.SubscriptionLifecycleReconciliationResult, error) {
				reconcilerCalled = true

				if action != services.SubscriptionLifecyclePaymentVoid {
					t.Fatalf(
						"action = %q, want PAYMENT_VOID",
						action,
					)
				}

				if keyMaterial != "test-key" {
					t.Fatalf(
						"key material = %q, want test-key",
						keyMaterial,
					)
				}

				return services.SubscriptionLifecycleReconciliationResult{
					Action:                  action,
					SubscriptionID:          subscription.ID,
					Status:                  subscription.Status,
					ReconciliationAttempted: true,
				}, nil
			},
			keyMaterial: "test-key",
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	if !reconcilerCalled {
		t.Fatal("PPP reconciler was not called")
	}

	var body map[string]any
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&body,
	); err != nil {
		t.Fatal(err)
	}

	reconciliation, ok :=
		body["pppoe_reconciliation"].(map[string]any)
	if !ok {
		t.Fatalf(
			"missing PPP reconciliation response: %+v",
			body,
		)
	}

	if reconciliation["action"] != "PAYMENT_VOID" {
		t.Fatalf(
			"action = %v, want PAYMENT_VOID",
			reconciliation["action"],
		)
	}
}

func TestVoidPaymentHandlerSkipsPPPWhenNoRenewalReversal(
	t *testing.T,
) {
	subscriptionLoaderCalled := false
	reconcilerCalled := false

	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			return &models.PaymentVoidAudit{
				PaymentID: id,
				Reason:    reason,
			}, nil
		},
		paymentVoidPostCommitDependencies{
			reversalLoader: func(
				paymentID uint,
			) (*models.SubscriptionRenewalReversal, bool, error) {
				return nil, false, nil
			},
			subscriptionLoader: func(
				id uint,
			) (*models.Subscription, error) {
				subscriptionLoaderCalled = true
				return nil, nil
			},
			reconciler: func(
				subscription *models.Subscription,
				action services.SubscriptionLifecycleAction,
				keyMaterial string,
			) (services.SubscriptionLifecycleReconciliationResult, error) {
				reconcilerCalled = true
				return services.SubscriptionLifecycleReconciliationResult{}, nil
			},
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if subscriptionLoaderCalled {
		t.Fatal(
			"subscription loader called without reversal",
		)
	}

	if reconcilerCalled {
		t.Fatal(
			"PPP reconciler called without reversal",
		)
	}
}

func TestVoidPaymentHandlerKeepsVoidSuccessWhenPPPReconciliationFails(
	t *testing.T,
) {
	handler := voidPaymentHandler(
		func(
			id uint,
			reason string,
			voidedByUserID uint,
			now time.Time,
		) (*models.PaymentVoidAudit, error) {
			return &models.PaymentVoidAudit{
				PaymentID:      id,
				SubscriptionID: 9,
				Reason:         reason,
			}, nil
		},
		paymentVoidPostCommitDependencies{
			reversalLoader: func(
				paymentID uint,
			) (*models.SubscriptionRenewalReversal, bool, error) {
				return &models.SubscriptionRenewalReversal{
					PaymentID:      paymentID,
					SubscriptionID: 9,
				}, true, nil
			},
			subscriptionLoader: func(
				id uint,
			) (*models.Subscription, error) {
				return &models.Subscription{
					Model:  gorm.Model{ID: id},
					Status: "EXPIRED",
				}, nil
			},
			reconciler: func(
				subscription *models.Subscription,
				action services.SubscriptionLifecycleAction,
				keyMaterial string,
			) (services.SubscriptionLifecycleReconciliationResult, error) {
				return services.SubscriptionLifecycleReconciliationResult{
						Action:         action,
						SubscriptionID: subscription.ID,
						Status:         subscription.Status,
					},
					errors.New("router unavailable")
			},
		},
	)

	recorder := performPaymentVoidRequest(
		t,
		handler,
		"41",
		`{"reason":"duplicate recharge"}`,
		99,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body=%s",
			recorder.Code,
			http.StatusOK,
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

	if body["message"] !=
		"Payment voided successfully" {
		t.Fatalf(
			"unexpected message: %v",
			body["message"],
		)
	}

	if body["pppoe_reconciliation_error"] !=
		"router unavailable" {
		t.Fatalf(
			"reconciliation error = %v",
			body["pppoe_reconciliation_error"],
		)
	}
}
