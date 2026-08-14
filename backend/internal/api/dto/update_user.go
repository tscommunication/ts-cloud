package dto

type UpdateUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Active   *bool  `json:"active"`
	AgentID  *uint  `json:"agent_id"`
}
