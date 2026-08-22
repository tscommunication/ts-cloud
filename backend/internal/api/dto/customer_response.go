package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerResponse struct {
	ID           uint   `json:"id"`
	CustomerCode string `json:"customer_code"`
	FullName     string `json:"full_name"`
	FatherName   string `json:"father_name"`
	MotherName   string `json:"mother_name"`
	Mobile       string `json:"mobile"`
	AltMobile    string `json:"alt_mobile"`
	Email        string `json:"email"`
	NID          string `json:"nid"`

	DateOfBirth string `json:"date_of_birth"`
	JoiningDate string `json:"joining_date"`

	Occupation  string `json:"occupation"`
	CompanyName string `json:"company_name"`
	Designation string `json:"designation"`

	NIDBirthDate string `json:"nid_birth_date"`
	NIDIssueDate string `json:"nid_issue_date"`
	NIDAddress   string `json:"nid_address"`

	PresentAddress   string `json:"present_address"`
	PermanentAddress string `json:"permanent_address"`
	TIN              string `json:"tin"`
	CustomerNote     string `json:"customer_note"`
	Country          string `json:"country"`
	Division         string `json:"division"`
	District         string `json:"district"`
	Upazila          string `json:"upazila"`
	PostOffice       string `json:"post_office"`
	PostalCode       string `json:"postal_code"`
	RoadOrArea       string   `json:"road_or_area"`
	VillageOrHolding string   `json:"village_or_holding"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	Union            string `json:"union"`
	Village          string `json:"village"`
	Address          string `json:"address"`
	Status           string `json:"status"`
	BillingDay       int    `json:"billing_day"`
	PopID            *uint  `json:"pop_id"`
	AgentID          *uint  `json:"agent_id"`
}

func ToCustomerResponse(customer models.Customer) CustomerResponse {
	return CustomerResponse{
		ID:               customer.ID,
		CustomerCode:     customer.CustomerCode,
		FullName:         customer.FullName,
		FatherName:       customer.FatherName,
		MotherName:       customer.MotherName,
		Mobile:           customer.Mobile,
		AltMobile:        customer.AltMobile,
		Email:            customer.Email,
		NID:              customer.NID,
		DateOfBirth:      formatOptionalCustomerDate(customer.DateOfBirth),
		JoiningDate:      formatOptionalCustomerDate(customer.JoiningDate),
		Occupation:       customer.Occupation,
		CompanyName:      customer.CompanyName,
		Designation:      customer.Designation,
		NIDBirthDate:     formatOptionalCustomerDate(customer.NIDBirthDate),
		NIDIssueDate:     formatOptionalCustomerDate(customer.NIDIssueDate),
		NIDAddress:       customer.NIDAddress,
		PresentAddress:   customer.PresentAddress,
		PermanentAddress: customer.PermanentAddress,
		TIN:              customer.TIN,
		CustomerNote:     customer.CustomerNote,
		Country:          customer.Country,
		Division:         customer.Division,
		District:         customer.District,
		Upazila:          customer.Upazila,
		PostOffice:       customer.PostOffice,
		PostalCode:       customer.PostalCode,
		RoadOrArea:       customer.RoadOrArea,
		VillageOrHolding: customer.VillageOrHolding,
		Latitude:         customer.Latitude,
		Longitude:        customer.Longitude,
		Union:            customer.Union,
		Village:          customer.Village,
		Address:          customer.Address,
		Status:           customer.Status,
		BillingDay:       customer.BillingDay,
		PopID:            customer.PopID,
		AgentID:          customer.AgentID,
	}
}

func formatOptionalCustomerDate(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.Format("02-01-2006")
}
