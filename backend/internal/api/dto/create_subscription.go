package dto

type CreateSubscriptionRequest struct {
	CustomerID uint `json:"customer_id" binding:"required"`
	PackageID  uint `json:"package_id" binding:"required"`

	ActivationDate string `json:"activation_date"`

	BillingDay uint `json:"billing_day"`

	RouterID uint `json:"router_id"`

	PPPoEUsername string `json:"pppoe_username"`
	PPPoEPassword string `json:"pppoe_password"`

	Remarks string `json:"remarks"`
}
