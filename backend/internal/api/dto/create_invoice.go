package dto

type CreateInvoiceRequest struct {
	SubscriptionID uint `json:"subscription_id" binding:"required"`

	BillMonth int `json:"bill_month" binding:"required"`
	BillYear  int `json:"bill_year" binding:"required"`

	IssueDate string `json:"issue_date"`
	DueDate   string `json:"due_date"`

	PackagePrice float64 `json:"package_price"`
	Discount     float64 `json:"discount"`
	Vat          float64 `json:"vat"`

	Remarks string `json:"remarks"`
}
