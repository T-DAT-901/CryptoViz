package dto

import "time"

// APIResponse est la structure de réponse standard de l'API
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// SuccessResponse crée une réponse de succès
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// SuccessMessageResponse crée une réponse de succès avec un message
func SuccessMessageResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ErrorResponse crée une réponse d'erreur
func ErrorResponse(message string) APIResponse {
	return APIResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now(),
	}
}
