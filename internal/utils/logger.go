package utils

import (
	"os"
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

		// httpRequest object (nested)
		httpRequest := map[string]interface{}{
			"requestMethod": c.Request.Method,
			"requestUrl": path,
			"status": c.Writer.Status(),
			"latency": latency.String(),
		}

		// Resource labels (dynamic from environment variables)
		resource := map[string]interface{}{
			"labels": map[string]interface{}{
				"project_id": os.Getenv("PROJECT_ID"),
				"app": os.Getenv("APP_NAME"),
				"source": os.Getenv("APP_SOURCE"),
			},
		}

		// Get correlation ID from context
		correlationID, _ := c.Get(CorrelationIDKey)
		correlationIDStr, _ := correlationID.(string)

		logger.Info().
			Str("severity", severity).
			Str("correlation_id", correlationIDStr).
			Time("timestamp", end).
			Interface("resource", resource).
			Interface("httpRequest", httpRequest).
			Str("service", os.Getenv("APP_NAME")).
			Msg(msg)
	}
}
