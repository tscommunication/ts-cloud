package dto

type CreatePaymentRequest struct {
	InvoiceID uint `json:"invoice_id" binding:"required"`

	PaymentDate string `json:"payment_date"`

	Amount float64 `json:"amount" binding:"required"`

	Method string `json:"method"`

	TransactionID string `json:"transaction_id"`

	Reference string `json:"reference"`

	Remarks string `json:"remarks"`
}
