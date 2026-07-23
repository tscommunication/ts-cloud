package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type InvoiceResponse struct {
	ID             uint      `json:"id"`
	InvoiceNo      string    `json:"invoice_no"`
	SubscriptionID uint      `json:"subscription_id"`
	CustomerID     uint      `json:"customer_id"`
	PackageID      uint      `json:"package_id"`

	BillMonth int `json:"bill_month"`
	BillYear  int `json:"bill_year"`

	IssueDate time.Time `json:"issue_date"`
	DueDate   time.Time `json:"due_date"`

	PackagePrice float64 `json:"package_price"`
	Discount     float64 `json:"discount"`
	Vat          float64 `json:"vat"`
	TotalAmount  float64 `json:"total_amount"`
	PaidAmount   float64 `json:"paid_amount"`
	DueAmount    float64 `json:"due_amount"`

	Status  string `json:"status"`
	Remarks string `json:"remarks"`
}

func ToInvoiceResponse(i models.Invoice) InvoiceResponse {
	return InvoiceResponse{
		ID:             i.ID,
		InvoiceNo:      i.InvoiceNo,
		SubscriptionID: i.SubscriptionID,
		CustomerID:     i.CustomerID,
		PackageID:      i.PackageID,

		BillMonth: i.BillMonth,
		BillYear:  i.BillYear,

		IssueDate: i.IssueDate,
		DueDate:   i.DueDate,

		PackagePrice: i.PackagePrice,
		Discount:     i.Discount,
		Vat:          i.Vat,
		TotalAmount:  i.TotalAmount,
		PaidAmount:   i.PaidAmount,
		DueAmount:    i.DueAmount,

		Status:  i.Status,
		Remarks: i.Remarks,
	}
}
