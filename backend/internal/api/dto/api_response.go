package dto

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
	}
}

func SuccessMessage(message string) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
	}
}

func Error(message string) APIResponse {
	return APIResponse{
		Success: false,
		Message: message,
	}
}
