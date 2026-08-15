package models

import "time"

// Division is the top-level Bangladesh administrative division.
type Division struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// District belongs to a Division.
type District struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DivisionID uint      `gorm:"not null;index;uniqueIndex:idx_district_division_name" json:"division_id"`
	Division   Division  `gorm:"foreignKey:DivisionID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	Name       string    `gorm:"size:100;not null;uniqueIndex:idx_district_division_name" json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Upazila represents an Upazila / Thana and belongs to a District.
type Upazila struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DistrictID uint      `gorm:"not null;index;uniqueIndex:idx_upazila_district_name" json:"district_id"`
	District   District  `gorm:"foreignKey:DistrictID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	Name       string    `gorm:"size:100;not null;uniqueIndex:idx_upazila_district_name" json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PostOffice belongs to an Upazila / Thana.
// PostalCode is stored in the location master and can be copied into a
// customer's address snapshot when a post office is selected.
type PostOffice struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UpazilaID  uint      `gorm:"not null;index;uniqueIndex:idx_post_office_upazila_name" json:"upazila_id"`
	Upazila    Upazila   `gorm:"foreignKey:UpazilaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	Name       string    `gorm:"size:150;not null;uniqueIndex:idx_post_office_upazila_name" json:"name"`
	PostalCode string    `gorm:"size:20;index" json:"postal_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
