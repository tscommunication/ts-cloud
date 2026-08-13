package dto

import "github.com/tscommunication/ts-cloud/internal/models"

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
	Division     string `json:"division"`
	District     string `json:"district"`
	Upazila      string `json:"upazila"`
	Union        string `json:"union"`
	Village      string `json:"village"`
	Address      string `json:"address"`
	Status       string `json:"status"`
	BillingDay   int    `json:"billing_day"`
}

func ToCustomerResponse(customer models.Customer) CustomerResponse {
	return CustomerResponse{
		ID:           customer.ID,
		CustomerCode: customer.CustomerCode,
		FullName:     customer.FullName,
		FatherName:   customer.FatherName,
		MotherName:   customer.MotherName,
		Mobile:       customer.Mobile,
		AltMobile:    customer.AltMobile,
		Email:        customer.Email,
		NID:          customer.NID,
		Division:     customer.Division,
		District:     customer.District,
		Upazila:      customer.Upazila,
		Union:        customer.Union,
		Village:      customer.Village,
		Address:      customer.Address,
		Status:       customer.Status,
		BillingDay:   customer.BillingDay,
	}
}
