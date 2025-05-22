package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CorrelationIDHeader = "X-Correlation-ID"
const CorrelationIDKey = "CorrelationID"

// CorrelationIDMiddleware ensures every request has a correlation ID, sets it in context and response header.
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set(CorrelationIDKey, correlationID)
		c.Writer.Header().Set(CorrelationIDHeader, correlationID)
		c.Next()
	}
}
