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
