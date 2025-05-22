package routes

import (
	"github.com/muhammad-junaid-iftikhar/app-minio-api/config"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/internal/api/handlers"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
)

func SetupRoutes(router *gin.Engine, minioClient *minio.Client, logger *zerolog.Logger, cfg *config.Config) {
	// Initialize MinIO handler
	minioHandler := handlers.NewMinioHandler(minioClient, logger, cfg)

	// API version group
	v1 := router.Group("/api/v1")
	{
		// File operations
		files := v1.Group("/files")
		{
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
			files.GET("", minioHandler.ListFiles)

			// Get file
			// @Summary Get a file
			// @Description Get a file from MinIO by its name
			// @Tags files
			// @Produce octet-stream
			// @Param filename path string true "File name"
			// @Success 200 {file} binary
			// @Router /api/v1/files/{filename} [get]
			files.GET("/:filename", minioHandler.GetFile)

			// Delete file
			// @Summary Delete a file
			// @Description Delete a file from MinIO by its name
			// @Tags files
			// @Produce json
			// @Param filename path string true "File name"
			// @Success 200 {object} map[string]string
			// @Router /api/v1/files/{filename} [delete]
			files.DELETE("/:filename", minioHandler.DeleteFile)
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