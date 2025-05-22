package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/config"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/internal/utils"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog"
)

// MinioHandler handles operations related to MinIO
type MinioHandler struct {
	minioClient *minio.Client
	logger      *zerolog.Logger
	config      *config.Config
}

// NewMinioHandler creates a new MinioHandler
func NewMinioHandler(minioClient *minio.Client, logger *zerolog.Logger, cfg *config.Config) *MinioHandler {
	return &MinioHandler{
		minioClient: minioClient,
		logger:      logger,
		config:      cfg,
	}
}

// UploadFile handles file upload to MinIO
// @Summary Upload a file to MinIO
// @Description Upload a file to MinIO storage
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {object} map[string]string
// @Router /files [post]
func (h *MinioHandler) UploadFile(c *gin.Context) {
	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get file from form")
		utils.SendError(c, http.StatusBadRequest, "Failed to get file")
		return
	}
	defer file.Close()

	// Generate object name (using original filename)
	objectName := header.Filename
	contentType := header.Header.Get("Content-Type")

	// If content type is not provided, try to determine it from the file extension
	if contentType == "" {
		contentType = "application/octet-stream"
		ext := filepath.Ext(objectName)
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".pdf":
			contentType = "application/pdf"
		case ".txt":
			contentType = "text/plain"
		case ".mp4":
			contentType = "video/mp4"
		}
	}

	// Upload the file to MinIO
	info, err := h.minioClient.PutObject(
		context.Background(),
		h.config.MinioBucketName,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to upload file to MinIO")
		utils.SendError(c, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	h.logger.Info().
		Str("bucket", info.Bucket).
		Str("object", info.Key).
		Int64("size", info.Size).
		Msg("File uploaded successfully")

	utils.SendJSONWithCorrelationID(c, http.StatusOK, map[string]interface{}{
		"message":    "File uploaded successfully",
		"filename":   objectName,
		"size":       info.Size,
		"bucketName": info.Bucket,
	})
}

// ListFiles lists all files in the bucket
// @Summary List all files
// @Description List all files in the MinIO bucket
// @Tags files
// @Produce json
// @Success 200 {array} object
// @Router /files [get]
func (h *MinioHandler) ListFiles(c *gin.Context) {
	ctx := context.Background()
	objectCh := h.minioClient.ListObjects(ctx, h.config.MinioBucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	var objects []map[string]interface{}
	for object := range objectCh {
		if object.Err != nil {
			h.logger.Error().Err(object.Err).Msg("Error listing objects")
			utils.SendError(c, http.StatusInternalServerError, "Failed to list files")
			return
		}

		objects = append(objects, map[string]interface{}{
			"name":         object.Key,
			"size":         object.Size,
			"lastModified": object.LastModified,
			"contentType":  object.ContentType,
		})
	}

	utils.SendJSONWithCorrelationID(c, http.StatusOK, objects)
}

// GetFile gets a file from MinIO
// @Summary Get a file
// @Description Get a file from MinIO by its name
// @Tags files
// @Produce octet-stream
// @Param filename path string true "File name"
// @Success 200 {file} binary
// @Router /files/{filename} [get]
func (h *MinioHandler) GetFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename is required"})
		return
	}

	// Get the object from MinIO
	object, err := h.minioClient.GetObject(
		context.Background(),
		h.config.MinioBucketName,
		filename,
		minio.GetObjectOptions{},
	)
	if err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to get file from MinIO")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get file")
		return
	}
	defer object.Close()

	// Get object info to determine content type
	stat, err := object.Stat()
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			utils.SendError(c, http.StatusNotFound, "File not found")
			return
		}
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to get file stats")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get file info")
		return
	}

	// Set the content disposition header to force download with original filename
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))

	// Stream the file to the response
	if _, err := io.Copy(c.Writer, object); err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to stream file")
		// Cannot send JSON response here as we've already started writing the response
		return
	}
}

// DeleteFile deletes a file from MinIO
// @Summary Delete a file
// @Description Delete a file from MinIO by its name
// @Tags files
// @Produce json
// @Param filename path string true "File name"
// @Success 200 {object} map[string]string
// @Router /files/{filename} [delete]
func (h *MinioHandler) DeleteFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename is required"})
		return
	}

	// Delete the object from MinIO
	err := h.minioClient.RemoveObject(
		context.Background(),
		h.config.MinioBucketName,
		filename,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		h.logger.Error().Err(err).Str("filename", filename).Msg("Failed to delete file from MinIO")
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete file")
		return
	}

	h.logger.Info().Str("filename", filename).Msg("File deleted successfully")
	utils.SendJSONWithCorrelationID(c, http.StatusOK, map[string]interface{}{
		"message": "File deleted successfully",
		"filename": filename,
	})
}

// ListBuckets lists all buckets
// @Summary List all buckets
// @Description List all buckets in MinIO
// @Tags buckets
// @Produce json
// @Success 200 {array} object
// @Router /buckets [get]
func (h *MinioHandler) ListBuckets(c *gin.Context) {
	buckets, err := h.minioClient.ListBuckets(context.Background())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list buckets")
		utils.SendError(c, http.StatusInternalServerError, "Failed to list buckets")
		return
	}

	var result []map[string]interface{}
	for _, bucket := range buckets {
		result = append(result, map[string]interface{}{
			"name":         bucket.Name,
			"creationDate": bucket.CreationDate.Format(time.RFC3339),
		})
	}

	utils.SendJSONWithCorrelationID(c, http.StatusOK, result)
}
