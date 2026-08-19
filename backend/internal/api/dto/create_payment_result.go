package dto

import "github.com/tscommunication/ts-cloud/internal/models"

// PaymentRenewalResultResponse is the stable HTTP representation of a
// payment-triggered subscription renewal result.
type PaymentRenewalResultResponse struct {
	Renewed bool   `json:"renewed"`
	Reason  string `json:"reason,omitempty"`

	Renewal *models.SubscriptionRenewal `json:"renewal,omitempty"`
}

// PPPoEReconciliationResponse is the stable HTTP representation of the
// optional post-commit PPPoE reconciliation result.
type PPPoEReconciliationResponse struct {
	Action string `json:"action,omitempty"`

	SubscriptionID uint `json:"subscription_id,omitempty"`

	ReconciliationAttempted bool `json:"reconciliation_attempted"`

	Reconciliation any `json:"reconciliation,omitempty"`

	ReconciliationError string `json:"reconciliation_error,omitempty"`
}

// CreatePaymentResultResponse documents the HTTP 201 response returned after
// payment creation.
//
// Payment creation and renewal are committed atomically. MikroTik/PPPoE
// reconciliation happens after commit, so reconciliation failure does not
// invalidate the successfully created payment.
type CreatePaymentResultResponse struct {
	Payment PaymentResponse `json:"payment"`

	Renewal PaymentRenewalResultResponse `json:"renewal"`

	PPPoEReconciliation *PPPoEReconciliationResponse `json:"pppoe_reconciliation,omitempty"`

	PPPoEReconciliationError string `json:"pppoe_reconciliation_error,omitempty"`
}
