package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// responseWriter is a minimal wrapper for gin.ResponseWriter that tracks the status code and response size
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
	size   int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.body.Write(b)
	w.size += size
	return size, err
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// LoggerMiddleware creates a middleware that logs all incoming requests with detailed information
func LoggerMiddleware(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health checks to reduce noise
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Capture request body if needed (for debugging)
		var requestBody []byte
		if c.Request.Body != nil {
			body, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			requestBody = body
		}

		// Create a custom response writer to capture response status and size
		blw := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		latency := time.Since(start)


		// Build log fields
		logEvent := logger.Info()
		if c.Request.URL.Path != "/metrics" { // Skip metrics endpoint from detailed logging
			logEvent = logEvent.Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("query", c.Request.URL.RawQuery).
				Str("ip", c.ClientIP()).
				Str("user-agent", c.Request.UserAgent()).
				Int("status", blw.Status()).
				Int("size", blw.size).
				Dur("latency", latency).
				Str("latency_human", latency.String())

			// Add request ID if available
			if requestID := c.Writer.Header().Get("X-Request-ID"); requestID != "" {
				logEvent = logEvent.Str("request_id", requestID)
			}

			// Add error message if any
			if len(c.Errors) > 0 {
				logEvent = logEvent.Strs("errors", c.Errors.Errors())
			}

			// Add request body for debugging (be careful with sensitive data)
			if len(requestBody) > 0 && len(requestBody) < 1024 { // Limit size
				logEvent = logEvent.Bytes("request_body", requestBody)
			}

			logEvent.Msg("Request processed")
		} else {
			logEvent.Msg("Metrics endpoint accessed")
		}
	}
}