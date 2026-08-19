package models

import (
	"time"

	"gorm.io/gorm"
)

type CustomerReference struct {
	ID uint `gorm:"primaryKey" json:"id"`

	CustomerID uint     `gorm:"not null;index" json:"customer_id"`
	Customer   Customer `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	Name     string `gorm:"size:150" json:"name"`
	Mobile   string `gorm:"size:20" json:"mobile"`
	Address  string `gorm:"type:text" json:"address"`
	Relation string `gorm:"size:100" json:"relation"`
	Note     string `gorm:"type:text" json:"note"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
