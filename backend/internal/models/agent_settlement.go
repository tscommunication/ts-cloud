package models

import (
	"time"

	"gorm.io/gorm"
)

type AgentSettlement struct {
	gorm.Model
	SettlementNo  string    `gorm:"size:30;not null;uniqueIndex" json:"settlement_no"`
	AgentID       uint      `gorm:"not null;index" json:"agent_id"`
	Agent         Agent     `gorm:"foreignKey:AgentID" json:"-"`
	Amount        float64   `gorm:"not null" json:"amount"`
	Method        string    `gorm:"size:30;not null" json:"method"`
	TransactionID string    `gorm:"size:100;index" json:"transaction_id"`
	PaidAt        time.Time `gorm:"not null;index" json:"paid_at"`
	Status        string    `gorm:"size:20;not null;default:PAID;index" json:"status"`
	Remarks       string    `gorm:"type:text" json:"remarks"`
}
