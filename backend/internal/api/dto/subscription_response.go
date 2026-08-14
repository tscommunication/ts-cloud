package dto

import (
	"github.com/tscommunication/ts-cloud/internal/models"
)

type SubscriptionResponse struct {
	ID               uint   `json:"id"`
	SubscriptionCode string `json:"subscription_code"`

	CustomerID   uint   `json:"customer_id"`
	PackageID    uint   `json:"package_id"`
	CustomerCode string `json:"customer_code"`
	CustomerName string `json:"customer_name"`
	PackageCode  string `json:"package_code"`
	PackageName  string `json:"package_name"`

	Status string `json:"status"`

	ActivationDate  string `json:"activation_date"`
	NextBillingDate string `json:"next_billing_date"`
	ExpiryDate      string `json:"expiry_date"`

	BillingDay int `json:"billing_day"`

	RouterID uint `json:"router_id"`

	PPPoEUsername string `json:"pppoe_username"`

	Remarks string `json:"remarks"`
}

func ToSubscriptionResponse(s models.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:               s.ID,
		SubscriptionCode: s.SubscriptionCode,

		CustomerID:   s.CustomerID,
		PackageID:    s.PackageID,
		CustomerCode: s.Customer.CustomerCode,
		CustomerName: s.Customer.FullName,
		PackageCode:  s.Package.PackageCode,
		PackageName:  s.Package.Name,

		Status: s.Status,

		ActivationDate:  s.ActivationDate.Format("2006-01-02"),
		NextBillingDate: s.NextBillingDate.Format("2006-01-02"),
		ExpiryDate:      s.ExpiryDate.Format("2006-01-02"),

		BillingDay: s.BillingDay,

		RouterID: s.RouterID,

		PPPoEUsername: s.PPPoEUsername,

		Remarks: s.Remarks,
	}
}
