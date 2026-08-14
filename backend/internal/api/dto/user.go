package dto

import "github.com/tscommunication/ts-cloud/internal/models"

type UserResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
	AgentID  *uint  `json:"agent_id"`
}

func ToUserResponse(user models.User) UserResponse {
	return UserResponse{
		ID:       user.ID,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Active:   user.Active,
		AgentID:  user.AgentID,
	}
}

func ToUserResponses(users []models.User) []UserResponse {
	result := make([]UserResponse, 0, len(users))

	for _, user := range users {
		result = append(result, ToUserResponse(user))
	}

	return result
}
