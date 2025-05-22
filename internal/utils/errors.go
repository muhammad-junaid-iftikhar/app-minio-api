package utils

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an error response for Swagger documentation
type ErrorResponse struct {
	Error   bool   `json:"error" example:"true"`
	Message string `json:"message" example:"Error message details"`
}

func SendError(c *gin.Context, status int, message string) {
	SendErrorWithCorrelationID(c, status, message)
}