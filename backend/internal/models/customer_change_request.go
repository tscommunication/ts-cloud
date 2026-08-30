package models

import (
	"time"

	"gorm.io/gorm"
)

// CustomerChangeRequest records an agent-proposed service change. It is only
// executable after an administrator approves it.
type CustomerChangeRequest struct {
	gorm.Model
	RequestCode       string     `gorm:"size:30;not null;uniqueIndex" json:"request_code"`
	Type              string     `gorm:"size:30;not null;index" json:"type"` // BILLING_CYCLE, PACKAGE, LINE_SHIFT, CLOSE
	Status            string     `gorm:"size:20;not null;default:PENDING;index" json:"status"`
	CustomerID        uint       `gorm:"not null;index" json:"customer_id"`
	AgentID           uint       `gorm:"not null;index" json:"agent_id"`
	RequestedByUserID uint       `gorm:"not null;index" json:"requested_by_user_id"`
	Reason            string     `gorm:"type:text" json:"reason"`
	CurrentValue      string     `gorm:"type:text" json:"current_value"`
	RequestedValue    string     `gorm:"type:text" json:"requested_value"`
	ReviewedByUserID  *uint      `gorm:"index" json:"reviewed_by_user_id,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason   string     `gorm:"type:text" json:"rejection_reason"`
	ExecutedAt        *time.Time `json:"executed_at,omitempty"`
	ExecutionError    string     `gorm:"type:text" json:"execution_error"`
}
