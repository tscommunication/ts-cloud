package dto

type CreateCustomerRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Mobile   string `json:"mobile" binding:"required,len=11,numeric,startswith=01"`

	FatherName string `json:"father_name"`
	MotherName string `json:"mother_name"`

	AltMobile string `json:"alt_mobile"`
	Email     string `json:"email"`

	NID string `json:"nid" binding:"required,min=10,max=17,numeric"`

	Country          string `json:"country"`
	Division         string `json:"division"`
	District         string `json:"district"`
	Upazila          string `json:"upazila"`
	PostOffice       string `json:"post_office"`
	RoadOrArea       string `json:"road_or_area"`
	VillageOrHolding string `json:"village_or_holding"`

	// Legacy address fields remain accepted for compatibility.
	Union   string `json:"union"`
	Village string `json:"village"`
	Address string `json:"address"`

	BillingDay int   `json:"billing_day"`
	PopID      *uint `json:"pop_id"`
	AgentID    *uint `json:"agent_id"`
}
