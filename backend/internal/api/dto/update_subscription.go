package dto

type UpdateSubscriptionRequest struct {
	BillingDay uint `json:"billing_day"`

	RouterID uint `json:"router_id"`

	PPPoEUsername string `json:"pppoe_username"`
	PPPoEPassword string `json:"pppoe_password"`

	Remarks string `json:"remarks"`
}
