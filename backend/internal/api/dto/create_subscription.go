package dto

type CreateSubscriptionRequest struct {
	CustomerID uint `json:"customer_id" binding:"required"`
	PackageID  uint `json:"package_id" binding:"required"`

	ActivationDate string `json:"activation_date"`

	BillingDay uint `json:"billing_day"`

	Remarks string `json:"remarks"`
}
