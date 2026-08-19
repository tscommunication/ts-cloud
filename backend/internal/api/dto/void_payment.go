package dto

type VoidPaymentRequest struct {
	Reason string `json:"reason" binding:"required"`
}
