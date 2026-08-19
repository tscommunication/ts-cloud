package dto

type AdjustSubscriptionDateRequest struct {
	NewExpiryDate string `json:"new_expiry_date" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
}
