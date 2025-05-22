package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LoggerMiddleware creates a gin middleware for logging requests using zerolog
func LoggerMiddleware(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log request details
		end := time.Now()
		latency := end.Sub(start)

		msg := "Request"
		if len(c.Errors) > 0 {
			msg = c.Errors.String()
		}

		// GCP severity (uppercase)
		severity := "INFO"
		if c.Writer.Status() >= 500 {
			severity = "ERROR"
		} else if c.Writer.Status() >= 400 {
			severity = "WARNING"
		}

		// GCP httpRequest object
		httpRequest := map[string]interface{}{
			"requestMethod": c.Request.Method,
			"requestUrl": path,
			"status": c.Writer.Status(),
			"latency": latency.String(),
			"remoteIp": c.ClientIP(),
		}

		// Get correlation ID from context
		correlationID, _ := c.Get(CorrelationIDKey)
		correlationIDStr, _ := correlationID.(string)

		logger.Info().
			Str("severity", severity).
			Str("correlation_id", correlationIDStr).
			Time("timestamp", end).
			Interface("httpRequest", httpRequest).
			Msg(msg)
	}
}
