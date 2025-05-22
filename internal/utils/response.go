package utils

import (
	"github.com/gin-gonic/gin"
)

type StandardResponse struct {
	CorrelationID string      `json:"correlation_id"`
	Data          interface{} `json:"data,omitempty"`
	Error         interface{} `json:"error,omitempty"`
}

func SendJSONWithCorrelationID(c *gin.Context, status int, data interface{}) {
	correlationID, _ := c.Get(CorrelationIDKey)
	c.JSON(status, StandardResponse{
		CorrelationID: correlationID.(string),
		Data:          data,
	})
}

func SendErrorWithCorrelationID(c *gin.Context, status int, errMsg string) {
	correlationID, _ := c.Get(CorrelationIDKey)
	c.JSON(status, StandardResponse{
		CorrelationID: correlationID.(string),
		Error:         errMsg,
	})
}
