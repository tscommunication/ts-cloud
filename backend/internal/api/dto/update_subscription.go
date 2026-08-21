package dto

type UpdateSubscriptionRequest struct {
	PackageID uint `json:"package_id"`
	BillingDay uint `json:"billing_day"`

	Remarks string `json:"remarks"`
}
