package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/muhammad-junaid-iftikhar/app-minio-api/docs" // Import generated docs
	"github.com/muhammad-junaid-iftikhar/app-minio-api/config"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/internal/api/routes"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/internal/utils"
)

// @title           Minio Go API
// @version         1.0
// @description     A RESTful API for Minio Go application
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.yourdomain.com/support
// @contact.email  support@yourdomain.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth
func main() {
	// Set Gin mode based on APP_ENV (must be done before any Gin initialization)
	if os.Getenv("APP_ENV") != "dev" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize MinIO client
	minioClient, err := config.InitMinioClient(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize MinIO client")
	}

	// Set up Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	// Ensure every request has a correlation ID
	router.Use(utils.CorrelationIDMiddleware())
	router.Use(utils.LoggerMiddleware(&logger))

	// CORS middleware
	router.Use(func(c *gin.Context) {
		// List of allowed origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"https://drive-two.junistudio.org",
		}

		origin := c.Request.Header.Get("Origin")
		allowed := false

		// Check if the origin is in the allowed list
		for _, o := range allowedOrigins {
			if o == origin {
				allowed = true
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If origin is not in the allowed list, use the first one as default
		if !allowed && len(allowedOrigins) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Requested-With, X-Request-Id, X-Correlation-Id")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, X-Request-Id, X-Correlation-Id")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// For actual requests
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, X-Request-Id, X-Correlation-Id")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		c.Next()
	})

	// Initialize routes
	routes.SetupRoutes(router, minioClient, &logger, cfg)

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Start server
	go func() {
		logger.Info().Msgf("Starting server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	logger.Info().Msg("Server exited")
}