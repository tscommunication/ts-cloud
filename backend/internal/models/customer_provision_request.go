package models

import (
	"time"

	"gorm.io/gorm"
)

type CustomerProvisionRequest struct {
	gorm.Model

	RequestCode string `gorm:"uniqueIndex;size:30;not null"`

	// Source / Workflow
	Source string `gorm:"size:20;not null;default:AGENT;index"`
	// AGENT, WEBSITE

	Status string `gorm:"size:20;not null;default:PENDING;index"`
	// PENDING, APPROVED, REJECTED, CANCELLED, COMPLETED

	// Request ownership / scope
	AgentID *uint `gorm:"index"`
	POPID   *uint `gorm:"index"`

	// Customer Basic Information
	FullName   string `gorm:"size:150;not null"`
	Mobile     string `gorm:"size:20;not null;index"`
	FatherName string `gorm:"size:150"`
	MotherName string `gorm:"size:150"`
	AltMobile  string `gorm:"size:20"`
	Email      string `gorm:"size:150"`
	NID        string `gorm:"size:30;index"`

	// Address
	Country          string `gorm:"size:100;default:Bangladesh"`
	Division         string `gorm:"size:100"`
	District         string `gorm:"size:100"`
	Upazila          string `gorm:"size:100"`
	PostOffice       string `gorm:"size:150"`
	PostalCode       string `gorm:"size:20"`
	RoadOrArea       string `gorm:"size:255"`
	VillageOrHolding string `gorm:"size:255"`

	// Initial Service / Subscription
	PackageID uint `gorm:"not null;index"`
	RouterID  uint `gorm:"default:0;index"`

	PPPoEUsername          string `gorm:"size:100;not null;index"`
	PPPoEPassword          string `gorm:"size:255" json:"-"`
	PPPoEPasswordEncrypted string `gorm:"column:pp_po_e_password_encrypted;type:text" json:"-"`

	BillingDay     int       `gorm:"not null;default:1"`
	ActivationDate time.Time `gorm:"not null"`

	Remarks string `gorm:"type:text"`

	// Request audit
	RequestedByUserID uint      `gorm:"not null;index"`
	RequestedAt       time.Time `gorm:"not null"`

	ReviewedByUserID *uint `gorm:"index"`
	ReviewedAt       *time.Time

	RejectionReason string `gorm:"type:text"`

	// Result after approval
	CustomerID     *uint `gorm:"index"`
	SubscriptionID *uint `gorm:"index"`
}
