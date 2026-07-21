package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type CustomerResponse struct {
	ID           uint   `json:"id"`
	CustomerCode string `json:"customer_code"`
	FullName     string `json:"full_name"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	BillingDay   int    `json:"billing_day"`
}

func ToCustomerResponse(customer models.Customer) CustomerResponse {
	return CustomerResponse{
		ID:           customer.ID,
		CustomerCode: customer.CustomerCode,
		FullName:     customer.FullName,
		Mobile:       customer.Mobile,
		Email:        customer.Email,
		Status:       customer.Status,
		BillingDay:   customer.BillingDay,
	}
}
