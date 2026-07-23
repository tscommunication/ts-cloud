package dto

type CreatePaymentRequest struct {

	InvoiceID uint `json:"invoice_id" binding:"required"`

	PaymentDate string `json:"payment_date" binding:"required"`

	Amount float64 `json:"amount" binding:"required,gt=0"`

	Method string `json:"method" binding:"required"`

	TransactionID string `json:"transaction_id"`

	Reference string `json:"reference"`

	Remarks string `json:"remarks"`
}
