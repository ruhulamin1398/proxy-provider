package common

import "github.com/gin-gonic/gin"

// APIResponse is the standard API response envelope.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success sends a successful JSON response.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Data: data})
}

// Fail sends an error JSON response.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}
