package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type CreateCustomerProvisionRequest struct {
	FullName   string `json:"full_name" binding:"required"`
	Mobile     string `json:"mobile" binding:"required"`
	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`
	AltMobile  string `json:"alt_mobile"`
	Email      string `json:"email"`
	NID        string `json:"nid" binding:"required"`

	Country          string   `json:"country"`
	Division         string   `json:"division"`
	District         string   `json:"district"`
	Upazila          string   `json:"upazila"`
	PostOffice       string   `json:"post_office"`
	PostalCode       string   `json:"postal_code"`
	RoadOrArea       string   `json:"road_or_area"`
	VillageOrHolding string   `json:"village_or_holding"`
	Latitude         *float64 `json:"latitude" binding:"omitempty,gte=-90,lte=90"`
	Longitude        *float64 `json:"longitude" binding:"omitempty,gte=-180,lte=180"`

	PackageID uint `json:"package_id" binding:"required"`
	RouterID  uint `json:"router_id"`

	PPPoEUsername string `json:"pppoe_username" binding:"required"`
	PPPoEPassword string `json:"pppoe_password"`

	BillingDay     int    `json:"billing_day" binding:"required,min=1,max=31"`
	ActivationDate string `json:"activation_date"`

	Remarks string `json:"remarks"`
}

type RejectCustomerProvisionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type CustomerProvisionRequestResponse struct {
	ID          uint   `json:"id"`
	RequestCode string `json:"request_code"`
	Source      string `json:"source"`
	Status      string `json:"status"`

	AgentID *uint `json:"agent_id,omitempty"`
	POPID   *uint `json:"pop_id,omitempty"`

	FullName   string `json:"full_name"`
	Mobile     string `json:"mobile"`
	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`
	AltMobile  string `json:"alt_mobile"`
	Email      string `json:"email"`
	NID        string `json:"nid"`

	Country          string   `json:"country"`
	Division         string   `json:"division"`
	District         string   `json:"district"`
	Upazila          string   `json:"upazila"`
	PostOffice       string   `json:"post_office"`
	PostalCode       string   `json:"postal_code"`
	RoadOrArea       string   `json:"road_or_area"`
	VillageOrHolding string   `json:"village_or_holding"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`

	PackageID uint `json:"package_id"`
	RouterID  uint `json:"router_id"`

	PPPoEUsername string `json:"pppoe_username"`

	BillingDay     int       `json:"billing_day"`
	ActivationDate time.Time `json:"activation_date"`

	Remarks string `json:"remarks"`

	RequestedByUserID uint      `json:"requested_by_user_id"`
	RequestedAt       time.Time `json:"requested_at"`

	ReviewedByUserID *uint      `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`

	RejectionReason string `json:"rejection_reason"`

	CustomerID     *uint `json:"customer_id,omitempty"`
	SubscriptionID *uint `json:"subscription_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

func ToCustomerProvisionRequestResponse(
	row models.CustomerProvisionRequest,
) CustomerProvisionRequestResponse {
	return CustomerProvisionRequestResponse{
		ID:                row.ID,
		RequestCode:       row.RequestCode,
		Source:            row.Source,
		Status:            row.Status,
		AgentID:           row.AgentID,
		POPID:             row.POPID,
		FullName:          row.FullName,
		Mobile:            row.Mobile,
		FatherName:        row.FatherName,
		MotherName:        row.MotherName,
		AltMobile:         row.AltMobile,
		Email:             row.Email,
		NID:               row.NID,
		Country:           row.Country,
		Division:          row.Division,
		District:          row.District,
		Upazila:           row.Upazila,
		PostOffice:        row.PostOffice,
		PostalCode:        row.PostalCode,
		RoadOrArea:        row.RoadOrArea,
		VillageOrHolding:  row.VillageOrHolding,
		Latitude:          row.Latitude,
		Longitude:         row.Longitude,
		PackageID:         row.PackageID,
		RouterID:          row.RouterID,
		PPPoEUsername:     row.PPPoEUsername,
		BillingDay:        row.BillingDay,
		ActivationDate:    row.ActivationDate,
		Remarks:           row.Remarks,
		RequestedByUserID: row.RequestedByUserID,
		RequestedAt:       row.RequestedAt,
		ReviewedByUserID:  row.ReviewedByUserID,
		ReviewedAt:        row.ReviewedAt,
		RejectionReason:   row.RejectionReason,
		CustomerID:        row.CustomerID,
		SubscriptionID:    row.SubscriptionID,
		CreatedAt:         row.CreatedAt,
	}
}
