package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func assertJSONDoesNotContainKeys(
	t *testing.T,
	value any,
	keys ...string,
) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	body := string(data)

	for _, key := range keys {
		needle := `"` + key + `"`
		if strings.Contains(body, needle) {
			t.Fatalf(
				"sensitive/internal key %q leaked in response: %s",
				key,
				body,
			)
		}
	}
}

func TestCustomerPortalMeResponseDoesNotLeakSensitiveFields(
	t *testing.T,
) {
	customer := &models.Customer{
		ID:           1,
		CustomerCode: "CUS-SEC-001",
		FullName:     "Portal Customer",
		Mobile:       "01783000101",
		NID:          "1234567890123",
		TIN:          "TIN-SECRET",
		NIDAddress:   "Sensitive NID Address",
		CustomerNote: "Internal customer note",
		AgentID:      uintPtr(11),
		PopID:        uintPtr(12),
		Status:       "ACTIVE",
	}

	response := ToCustomerPortalMeResponse(customer)

	assertJSONDoesNotContainKeys(
		t,
		response,
		"nid",
		"nid_birth_date",
		"nid_issue_date",
		"nid_address",
		"tin",
		"customer_note",
		"agent_id",
		"pop_id",
	)
}

func TestCustomerPortalSubscriptionResponseDoesNotLeakInternalFields(
	t *testing.T,
) {
	subscription := models.Subscription{
		CustomerID:             10,
		PackageID:              20,
		RouterID:               30,
		PPPoEUsername:          "portal-user",
		PPPoEPassword:          "plaintext-secret",
		PPPoEPasswordEncrypted: "encrypted-secret",
		Remarks:                "Internal subscription note",
		Status:                 "ACTIVE",
	}

	response := ToCustomerPortalSubscriptionResponses(
		[]models.Subscription{subscription},
	)

	assertJSONDoesNotContainKeys(
		t,
		response,
		"customer_id",
		"router_id",
		"pppoe_password",
		"pp_po_e_password_encrypted",
		"remarks",
		"customer",
		"package",
	)
}

func TestCustomerPortalInvoiceResponseDoesNotLeakRelations(
	t *testing.T,
) {
	invoice := models.Invoice{
		CustomerID:     10,
		SubscriptionID: 20,
		PackageID:      30,
		InvoiceNo:      "INV-SEC-001",
		Status:         "UNPAID",
		Remarks:        "Internal invoice note",
	}

	response := ToCustomerPortalInvoiceResponses(
		[]models.Invoice{invoice},
	)

	assertJSONDoesNotContainKeys(
		t,
		response,
		"customer_id",
		"customer",
		"subscription",
		"package",
		"remarks",
	)
}

func TestCustomerPortalPaymentResponseDoesNotLeakCollectorRelations(
	t *testing.T,
) {
	payment := models.Payment{
		CustomerID:         10,
		SubscriptionID:     20,
		InvoiceID:          30,
		ReceiptNo:          "RCP-SEC-001",
		CollectedByUserID:  uintPtr(40),
		CollectedByAgentID: uintPtr(50),
		Reference:          "Internal reference",
		Remarks:            "Internal payment note",
		Status:             "SUCCESS",
	}

	response := ToCustomerPortalPaymentResponses(
		[]models.Payment{payment},
	)

	assertJSONDoesNotContainKeys(
		t,
		response,
		"customer_id",
		"customer",
		"invoice",
		"subscription",
		"collected_by_user_id",
		"collected_by_user",
		"collected_by_agent_id",
		"collected_by_agent",
		"reference",
		"remarks",
	)
}

func uintPtr(value uint) *uint {
	return &value
}

func TestCustomerPortalDateFormatIsDDMMYYYY(
	t *testing.T,
) {
	value := time.Date(
		2026,
		time.August,
		20,
		15,
		30,
		0,
		0,
		time.UTC,
	)

	customer := &models.Customer{
		DateOfBirth:    &value,
		JoiningDate:    &value,
		ActivationDate: &value,
	}

	me := ToCustomerPortalMeResponse(customer)

	if me.DateOfBirth != "20-08-2026" {
		t.Fatalf(
			"expected DD-MM-YYYY date, got %q",
			me.DateOfBirth,
		)
	}

	subscriptions := ToCustomerPortalSubscriptionResponses(
		[]models.Subscription{
			{
				ActivationDate:  value,
				NextBillingDate: value,
				ExpiryDate:      value,
				LastPaymentDate: &value,
			},
		},
	)

	if subscriptions[0].ExpiryDate != "20-08-2026" {
		t.Fatalf(
			"expected DD-MM-YYYY expiry date, got %q",
			subscriptions[0].ExpiryDate,
		)
	}

	invoices := ToCustomerPortalInvoiceResponses(
		[]models.Invoice{
			{
				IssueDate: value,
				DueDate:   value,
			},
		},
	)

	if invoices[0].DueDate != "20-08-2026" {
		t.Fatalf(
			"expected DD-MM-YYYY due date, got %q",
			invoices[0].DueDate,
		)
	}

	payments := ToCustomerPortalPaymentResponses(
		[]models.Payment{
			{
				PaymentDate: value,
			},
		},
	)

	if payments[0].PaymentDate != "20-08-2026" {
		t.Fatalf(
			"expected DD-MM-YYYY payment date, got %q",
			payments[0].PaymentDate,
		)
	}
}
