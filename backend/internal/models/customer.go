package models

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID uint `gorm:"primaryKey" json:"id"`

	CustomerCode string `gorm:"size:30;uniqueIndex;not null" json:"customer_code"`

	FullName string `gorm:"size:150;not null" json:"full_name"`

	FatherName string `gorm:"size:150" json:"father_name"`
	MotherName string `gorm:"size:150" json:"mother_name"`

	Mobile    string `gorm:"size:20;uniqueIndex:idx_customers_mobile_unique;not null" json:"mobile"`
	AltMobile string `gorm:"size:20" json:"alt_mobile"`

	Email string `gorm:"size:150" json:"email"`

	NID         string     `gorm:"size:30" json:"nid"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	JoiningDate *time.Time `json:"joining_date,omitempty"`

	Occupation  string `gorm:"size:150" json:"occupation"`
	CompanyName string `gorm:"size:200" json:"company_name"`
	Designation string `gorm:"size:150" json:"designation"`

	NIDBirthDate *time.Time `gorm:"column:nid_birth_date" json:"nid_birth_date,omitempty"`
	NIDIssueDate *time.Time `gorm:"column:nid_issue_date" json:"nid_issue_date,omitempty"`
	NIDAddress   string     `gorm:"column:nid_address;type:text" json:"nid_address"`

	PresentAddress   string `gorm:"type:text" json:"present_address"`
	PermanentAddress string `gorm:"type:text" json:"permanent_address"`
	TIN              string `gorm:"size:50" json:"tin"`
	CustomerNote     string `gorm:"type:text" json:"customer_note"`

	Country          string `gorm:"size:100" json:"country"`
	Division         string `gorm:"size:100" json:"division"`
	District         string `gorm:"size:100" json:"district"`
	Upazila          string `gorm:"size:100" json:"upazila"`
	PostOffice       string `gorm:"size:150" json:"post_office"`
	PostalCode       string `gorm:"size:20" json:"postal_code"`
	RoadOrArea       string   `gorm:"size:200" json:"road_or_area"`
	VillageOrHolding string   `gorm:"size:200" json:"village_or_holding"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`

	// Legacy address fields are retained for backward compatibility and imports.
	Union   string `gorm:"size:100" json:"union"`
	Village string `gorm:"size:150" json:"village"`
	Address string `gorm:"type:text" json:"address"`

	PopID     *uint  `gorm:"index" json:"pop_id,omitempty"`
	POP       *POP   `gorm:"foreignKey:PopID" json:"-"`
	AgentID   *uint  `gorm:"index" json:"agent_id,omitempty"`
	Agent     *Agent `gorm:"foreignKey:AgentID" json:"-"`
	PackageID *uint  `json:"package_id,omitempty"`

	Status string `gorm:"size:30;default:ACTIVE" json:"status"`

	BillingDay int `gorm:"default:1" json:"billing_day"`

	ActivationDate *time.Time `json:"activation_date,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
