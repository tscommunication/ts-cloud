package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerPortalMeResponse struct {
	ID               uint   `json:"id"`
	CustomerCode     string `json:"customer_code"`
	FullName         string `json:"full_name"`
	Mobile           string `json:"mobile"`
	AltMobile        string `json:"alt_mobile"`
	Email            string `json:"email"`
	DateOfBirth      string `json:"date_of_birth,omitempty"`
	JoiningDate      string `json:"joining_date,omitempty"`
	Occupation       string `json:"occupation"`
	CompanyName      string `json:"company_name"`
	Designation      string `json:"designation"`
	PresentAddress   string `json:"present_address"`
	PermanentAddress string `json:"permanent_address"`

	Country          string `json:"country"`
	Division         string `json:"division"`
	District         string `json:"district"`
	Upazila          string `json:"upazila"`
	PostOffice       string `json:"post_office"`
	PostalCode       string `json:"postal_code"`
	RoadOrArea       string `json:"road_or_area"`
	VillageOrHolding string `json:"village_or_holding"`

	Status         string `json:"status"`
	BillingDay     int    `json:"billing_day"`
	ActivationDate string `json:"activation_date,omitempty"`
	AgentCode      string `json:"agent_code,omitempty"`
	AgentName      string `json:"agent_name,omitempty"`
	AgentMobile    string `json:"agent_mobile,omitempty"`
	POPCode        string `json:"pop_code,omitempty"`
	POPName        string `json:"pop_name,omitempty"`
}

type CustomerPortalSubscriptionResponse struct {
	ID               uint    `json:"id"`
	SubscriptionCode string  `json:"subscription_code"`
	PackageID        uint    `json:"package_id"`
	ActivationDate   string  `json:"activation_date"`
	BillingDay       int     `json:"billing_day"`
	NextBillingDate  string  `json:"next_billing_date"`
	ExpiryDate       string  `json:"expiry_date"`
	Status           string  `json:"status"`
	PPPoEUsername    string  `json:"pppoe_username"`
	PPPoEPassword    string  `json:"pppoe_password,omitempty"`
	LastPaymentDate  string  `json:"last_payment_date,omitempty"`
	LastPaidAmount   float64 `json:"last_paid_amount"`
	DueAmount        float64 `json:"due_amount"`
}

type CustomerPortalConnectionResponse struct {
	PPPoEUsername   string `json:"pppoe_username"`
	Status          string `json:"status"`
	PackageCode     string `json:"package_code"`
	PackageName     string `json:"package_name"`
	RouterCode      string `json:"router_code"`
	RouterName      string `json:"router_name"`
	ExpiryDate      string `json:"expiry_date,omitempty"`
	MACAddress      string `json:"mac_address"`
	StaticIPAddress string `json:"static_ip_address"`
	Online          bool   `json:"online"`
	IPAddress       string `json:"ip_address"`
	Uptime          string `json:"uptime"`
	DownloadBps     int64  `json:"download_bps"`
	UploadBps       int64  `json:"upload_bps"`
	LastSeenAt      string `json:"last_seen_at,omitempty"`
}

type CustomerPortalInvoiceResponse struct {
	ID             uint    `json:"id"`
	InvoiceNo      string  `json:"invoice_no"`
	SubscriptionID uint    `json:"subscription_id"`
	PackageID      uint    `json:"package_id"`
	BillMonth      int     `json:"bill_month"`
	BillYear       int     `json:"bill_year"`
	IssueDate      string  `json:"issue_date"`
	DueDate        string  `json:"due_date"`
	PackagePrice   float64 `json:"package_price"`
	Discount       float64 `json:"discount"`
	Vat            float64 `json:"vat"`
	TotalAmount    float64 `json:"total_amount"`
	PaidAmount     float64 `json:"paid_amount"`
	DueAmount      float64 `json:"due_amount"`
	Status         string  `json:"status"`
}

type CustomerPortalPaymentResponse struct {
	ID             uint    `json:"id"`
	ReceiptNo      string  `json:"receipt_no"`
	InvoiceID      uint    `json:"invoice_id"`
	SubscriptionID uint    `json:"subscription_id"`
	PaymentDate    string  `json:"payment_date"`
	Amount         float64 `json:"amount"`
	Method         string  `json:"method"`
	TransactionID  string  `json:"transaction_id"`
	Status         string  `json:"status"`
}

const customerPortalDateLayout = "02-01-2006"

func formatPortalDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(customerPortalDateLayout)
}

func formatPortalDatePtr(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}

	return value.Format(customerPortalDateLayout)
}

func ToCustomerPortalMeResponse(
	customer *models.Customer,
) CustomerPortalMeResponse {
	return CustomerPortalMeResponse{
		ID:               customer.ID,
		CustomerCode:     customer.CustomerCode,
		FullName:         customer.FullName,
		Mobile:           customer.Mobile,
		AltMobile:        customer.AltMobile,
		Email:            customer.Email,
		DateOfBirth:      formatPortalDatePtr(customer.DateOfBirth),
		JoiningDate:      formatPortalDatePtr(customer.JoiningDate),
		Occupation:       customer.Occupation,
		CompanyName:      customer.CompanyName,
		Designation:      customer.Designation,
		PresentAddress:   customer.PresentAddress,
		PermanentAddress: customer.PermanentAddress,
		Country:          customer.Country,
		Division:         customer.Division,
		District:         customer.District,
		Upazila:          customer.Upazila,
		PostOffice:       customer.PostOffice,
		PostalCode:       customer.PostalCode,
		RoadOrArea:       customer.RoadOrArea,
		VillageOrHolding: customer.VillageOrHolding,
		Status:           customer.Status,
		BillingDay:       customer.BillingDay,
		ActivationDate:   formatPortalDatePtr(customer.ActivationDate),
	}
}

func ToCustomerPortalSubscriptionResponses(
	subscriptions []models.Subscription,
) []CustomerPortalSubscriptionResponse {
	response := make(
		[]CustomerPortalSubscriptionResponse,
		0,
		len(subscriptions),
	)

	for _, subscription := range subscriptions {
		response = append(
			response,
			CustomerPortalSubscriptionResponse{
				ID:               subscription.ID,
				SubscriptionCode: subscription.SubscriptionCode,
				PackageID:        subscription.PackageID,
				ActivationDate:   formatPortalDate(subscription.ActivationDate),
				BillingDay:       subscription.BillingDay,
				NextBillingDate:  formatPortalDate(subscription.NextBillingDate),
				ExpiryDate:       formatPortalDate(subscription.ExpiryDate),
				Status:           subscription.Status,
				PPPoEUsername:    subscription.PPPoEUsername,
				LastPaymentDate:  formatPortalDatePtr(subscription.LastPaymentDate),
				LastPaidAmount:   subscription.LastPaidAmount,
				DueAmount:        subscription.DueAmount,
			},
		)
	}

	return response
}

func ToCustomerPortalInvoiceResponses(
	invoices []models.Invoice,
) []CustomerPortalInvoiceResponse {
	response := make(
		[]CustomerPortalInvoiceResponse,
		0,
		len(invoices),
	)

	for _, invoice := range invoices {
		response = append(
			response,
			CustomerPortalInvoiceResponse{
				ID:             invoice.ID,
				InvoiceNo:      invoice.InvoiceNo,
				SubscriptionID: invoice.SubscriptionID,
				PackageID:      invoice.PackageID,
				BillMonth:      invoice.BillMonth,
				BillYear:       invoice.BillYear,
				IssueDate:      formatPortalDate(invoice.IssueDate),
				DueDate:        formatPortalDate(invoice.DueDate),
				PackagePrice:   invoice.PackagePrice,
				Discount:       invoice.Discount,
				Vat:            invoice.Vat,
				TotalAmount:    invoice.TotalAmount,
				PaidAmount:     invoice.PaidAmount,
				DueAmount:      invoice.DueAmount,
				Status:         invoice.Status,
			},
		)
	}

	return response
}

func ToCustomerPortalPaymentResponses(
	payments []models.Payment,
) []CustomerPortalPaymentResponse {
	response := make(
		[]CustomerPortalPaymentResponse,
		0,
		len(payments),
	)

	for _, payment := range payments {
		response = append(
			response,
			CustomerPortalPaymentResponse{
				ID:             payment.ID,
				ReceiptNo:      payment.ReceiptNo,
				InvoiceID:      payment.InvoiceID,
				SubscriptionID: payment.SubscriptionID,
				PaymentDate:    formatPortalDate(payment.PaymentDate),
				Amount:         payment.Amount,
				Method:         payment.Method,
				TransactionID:  payment.TransactionID,
				Status:         payment.Status,
			},
		)
	}

	return response
}
