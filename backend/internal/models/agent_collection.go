package models

import (
	"time"

	"gorm.io/gorm"
)

type AgentCollection struct {
	gorm.Model
	AgentID          uint      `gorm:"not null;index" json:"agent_id"`
	Agent            Agent     `gorm:"foreignKey:AgentID" json:"-"`
	CustomerID       uint      `gorm:"not null;index" json:"customer_id"`
	Customer         Customer  `gorm:"foreignKey:CustomerID" json:"-"`
	PaymentID        uint      `gorm:"not null;uniqueIndex" json:"payment_id"`
	Payment          Payment   `gorm:"foreignKey:PaymentID" json:"-"`
	Amount           float64   `gorm:"not null" json:"amount"`
	CommissionRate   float64   `gorm:"not null" json:"commission_rate"`
	CommissionAmount float64   `gorm:"not null" json:"commission_amount"`
	Status           string    `gorm:"size:20;not null;default:ACTIVE;index" json:"status"`
	CollectedAt      time.Time `gorm:"not null;index" json:"collected_at"`
}
