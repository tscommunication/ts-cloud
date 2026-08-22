package dto

type UpdateCustomerRequest struct {
	FullName   string `json:"full_name" binding:"required"`
	Mobile     string `json:"mobile" binding:"required,len=11,numeric,startswith=01"`
	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`
	AltMobile  string `json:"alt_mobile"`
	Email      string `json:"email"`
	NID        string `json:"nid" binding:"required,min=10,max=17,numeric"`

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
	Latitude         *float64 `json:"latitude" binding:"omitempty,gte=-90,lte=90"`
	Longitude        *float64 `json:"longitude" binding:"omitempty,gte=-180,lte=180"`

	// Legacy address fields remain accepted for compatibility.
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
