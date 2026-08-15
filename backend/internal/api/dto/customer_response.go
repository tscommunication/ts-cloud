package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type CustomerResponse struct {
	ID               uint   `json:"id"`
	CustomerCode     string `json:"customer_code"`
	FullName         string `json:"full_name"`
	FatherName       string `json:"father_name"`
	MotherName       string `json:"mother_name"`
	Mobile           string `json:"mobile"`
	AltMobile        string `json:"alt_mobile"`
	Email            string `json:"email"`
	NID              string `json:"nid"`
	Country          string `json:"country"`
	Division         string `json:"division"`
	District         string `json:"district"`
	Upazila          string `json:"upazila"`
	PostOffice       string `json:"post_office"`
	RoadOrArea       string `json:"road_or_area"`
	VillageOrHolding string `json:"village_or_holding"`
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
		Country:          customer.Country,
		Division:         customer.Division,
		District:         customer.District,
		Upazila:          customer.Upazila,
		PostOffice:       customer.PostOffice,
		RoadOrArea:       customer.RoadOrArea,
		VillageOrHolding: customer.VillageOrHolding,
		Union:            customer.Union,
		Village:          customer.Village,
		Address:          customer.Address,
		Status:           customer.Status,
		BillingDay:       customer.BillingDay,
		PopID:            customer.PopID,
		AgentID:          customer.AgentID,
	}
}
