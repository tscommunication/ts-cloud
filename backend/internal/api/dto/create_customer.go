package dto

type CreateCustomerRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Mobile   string `json:"mobile" binding:"required"`

	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`

	AltMobile string `json:"alt_mobile"`
	Email     string `json:"email"`

	NID string `json:"nid"`

	Division string `json:"division"`
	District string `json:"district"`
	Upazila  string `json:"upazila"`
	Union    string `json:"union"`
	Village  string `json:"village"`
	Address  string `json:"address"`

	BillingDay int `json:"billing_day"`
}
