package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type AuthMiddleware struct {
	logger       *zerolog.Logger
	authBaseURL string
}

func NewAuthMiddleware(logger *zerolog.Logger) *AuthMiddleware {
	authBaseURL := os.Getenv("AUTH_SERVICE_URL")
	return &AuthMiddleware{
		logger:       logger,
		authBaseURL: strings.TrimSuffix(authBaseURL, "/"),
	}
}

// Authenticate verifies the JWT token with app-auth-api
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log incoming request
		m.logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("ip", c.ClientIP()).
			Msg("Authentication middleware triggered")

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn().
				Str("path", c.Request.URL.Path).
				Msg("Missing Authorization header")

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   true,
				"message": "Authorization header is required",
			})
			return
		}

		// Extract the token from the header (handle both "Bearer <token>" and just "<token>" formats)
		var token string
		parts := strings.Split(authHeader, " ")
		
		// If header has 2 parts and starts with "Bearer" (case-insensitive)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = parts[1]
		} else if len(parts) == 1 {
			// If it's just the token without "Bearer" prefix
			token = parts[0]
		} else {
			errMsg := "Invalid authorization header format. Expected 'Bearer <token>' or just '<token>'"
			m.logger.Warn().
				Str("auth_header", authHeader).
				Msg(errMsg)

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   true,
				"message": errMsg,
			})
			return
		}
		if token == "" {
			errMsg := "Missing token"
			m.logger.Warn().
				Str("auth_header", authHeader).
				Msg(errMsg)

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   true,
				"message": errMsg,
			})
			return
		}

		// Verify the token with app-auth-api
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Log the auth service URL being used
		verifyURL := m.authBaseURL + "/api/v1/auth/verify"
		m.logger.Info().
			Str("auth_service_url", verifyURL).
			Msg("Preparing to verify token with auth service")

		// Create JSON body for the request
		reqBody := fmt.Sprintf(`{"token":"%s"}`, token)

		req, err := http.NewRequest("POST", verifyURL, strings.NewReader(reqBody))
		if err != nil {
			m.logger.Error().
				Err(err).
				Str("url", verifyURL).
				Msg("Failed to create verify request")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   true,
				"message": "Internal server error",
			})
			return
		}

		req.Header.Set("Content-Type", "application/json")
		
		// Log the complete request details
		headers := make(map[string]string)
		for k, v := range req.Header {
			headers[k] = strings.Join(v, ", ")
		}
		m.logger.Info().
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Interface("headers", headers).
			Str("body", reqBody).
			Msg("Sending token verification request")

		m.logger.Info().
			Str("method", req.Method).
			Str("url", verifyURL).
			Str("content_type", req.Header.Get("Content-Type")).
			Msg("Sending token verification request to auth service")

		// Send request to auth service
		resp, err := client.Do(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   true,
				"message": "Authentication service unavailable",
				"details": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			body = []byte("<failed to read response body>")
		}

		if resp.StatusCode != http.StatusOK {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   true,
				"message": "Invalid or expired token",
				"details": string(body),
			})
			return
		}

		// Token is valid, proceed to the next handler
		c.Next()
	}
}
