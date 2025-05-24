package routes

import (
	"github.com/muhammad-junaid-iftikhar/app-minio-api/config"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/internal/api/handlers"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
	"net/http"
)

// addCorsHeaders adds CORS headers to the response
func addCorsHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// optionsHandler handles OPTIONS requests
func optionsHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func SetupRoutes(router *gin.Engine, minioClient *minio.Client, logger *zerolog.Logger, cfg *config.Config) {
	// Add CORS middleware to all routes
	router.Use(addCorsHeaders())

	// Initialize MinIO handler
	minioHandler := handlers.NewMinioHandler(minioClient, logger, cfg)

	// Handle OPTIONS method for all routes
	router.OPTIONS("/*any", optionsHandler)

	// API version group
	v1 := router.Group("/api/v1")
	{
		// File operations
		files := v1.Group("/files")
		{
			// Handle OPTIONS for /api/v1/files
			files.OPTIONS("", optionsHandler)
			// Upload file
			// @Summary Upload a file to MinIO
			// @Description Upload a file to MinIO storage
			// @Tags files
			// @Accept multipart/form-data
			// @Produce json
			// @Param file formData file true "File to upload"
			// @Success 200 {object} map[string]string
			// @Router /api/v1/files [post]
			files.POST("", minioHandler.UploadFile)

			// List files
			// @Summary List all files
			// @Description List all files in the MinIO bucket
			// @Tags files
			// @Produce json
			// @Success 200 {array} object
			// @Router /api/v1/files [get]
			// @Router /api/v1/files [options]
			files.GET("", minioHandler.ListFiles)
			files.OPTIONS("", optionsHandler) // Explicit OPTIONS for the base path

			// Get file
			// @Summary Get a file
			// @Description Get a file from MinIO by its name
			// @Tags files
			// @Produce octet-stream
			// @Param filename path string true "File name"
			// @Success 200 {file} binary
			// @Router /api/v1/files/{filename} [get]
			// @Router /api/v1/files/{filename} [options]
			files.GET("/:filename", minioHandler.GetFile)
			files.OPTIONS("/:filename", optionsHandler)

			// Delete file
			// @Summary Delete a file
			// @Description Delete a file from MinIO by its name
			// @Tags files
			// @Produce json
			// @Param filename path string true "File name"
			// @Success 200 {object} map[string]string
			// @Router /api/v1/files/{filename} [delete]
			// @Router /api/v1/files/{filename} [options]
			files.DELETE("/:filename", minioHandler.DeleteFile)
			files.OPTIONS("/:filename", optionsHandler)
		}

		// Bucket operations
		buckets := v1.Group("/buckets")
		{
			// List buckets
			// @Summary List all buckets
			// @Description List all buckets in MinIO
			// @Tags buckets
			// @Produce json
			// @Success 200 {array} object
			// @Router /api/v1/buckets [get]
			buckets.GET("", minioHandler.ListBuckets)
		}
	}

	// Health check
	// @Summary Health check endpoint
	// @Description Check if the API is up and running
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /health [get]
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}