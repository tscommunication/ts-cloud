package dto

type UpdateCustomerRequest struct {
	FullName   string `json:"full_name" binding:"required"`
	Mobile     string `json:"mobile" binding:"required"`
	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`
	AltMobile  string `json:"alt_mobile"`
	Email      string `json:"email"`
	NID        string `json:"nid"`
	Division   string `json:"division"`
	District   string `json:"district"`
	Upazila    string `json:"upazila"`
	Union      string `json:"union"`
	Village    string `json:"village"`
	Address    string `json:"address"`
	BillingDay int    `json:"billing_day" binding:"min=1,max=31"`
	PopID      *uint  `json:"pop_id"`
	AgentID    *uint  `json:"agent_id"`
}

type UpdateCustomerStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE"`
}
