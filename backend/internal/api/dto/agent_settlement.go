package dto

type CreateAgentSettlementRequest struct {
	AgentID       uint    `json:"agent_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Method        string  `json:"method" binding:"required"`
	TransactionID string  `json:"transaction_id"`
	PaidAt        string  `json:"paid_at"`
	Remarks       string  `json:"remarks"`
}
