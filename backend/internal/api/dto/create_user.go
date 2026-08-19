package dto

type CreateUserRequest struct {
	Name       string `json:"name" binding:"required"`
	Username   string `json:"username" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	Role       string `json:"role"`
	AgentID    *uint  `json:"agent_id"`
	CustomerID *uint  `json:"customer_id"`
}
