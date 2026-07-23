package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type PaymentResponse struct {
	ID             uint      `json:"id"`
	ReceiptNo      string    `json:"receipt_no"`
	InvoiceID      uint      `json:"invoice_id"`
	CustomerID     uint      `json:"customer_id"`
	SubscriptionID uint      `json:"subscription_id"`

	PaymentDate time.Time `json:"payment_date"`

	Amount float64 `json:"amount"`

	Method string `json:"method"`

	TransactionID string `json:"transaction_id"`

	Status string `json:"status"`

	Reference string `json:"reference"`

	Remarks string `json:"remarks"`
}

func ToPaymentResponse(p models.Payment) PaymentResponse {
	return PaymentResponse{
		ID:             p.ID,
		ReceiptNo:      p.ReceiptNo,
		InvoiceID:      p.InvoiceID,
		CustomerID:     p.CustomerID,
		SubscriptionID: p.SubscriptionID,
		PaymentDate:    p.PaymentDate,
		Amount:         p.Amount,
		Method:         p.Method,
		TransactionID:  p.TransactionID,
		Status:         p.Status,
		Reference:      p.Reference,
		Remarks:        p.Remarks,
	}
}
