package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/muhammad-junaid-iftikhar/app-minio-api/config"
	"github.com/rs/zerolog"
)

type R2Handler struct {
	client *s3.Client
	logger *zerolog.Logger
	config *config.Config
}

func NewR2Handler(cfg *config.Config, logger *zerolog.Logger) (*R2Handler, error) {
	// Create a custom HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a new credential provider
	creds := credentials.NewStaticCredentialsProvider(
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		"",
	)

	// Create a custom endpoint resolver
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID),
			SigningRegion:     cfg.R2Region,
			HostnameImmutable: true,
		}, nil
	})

	// Create a new AWS config with our custom settings
	awsCfg := aws.Config{
		Region: cfg.R2Region,
		Credentials: creds,
		HTTPClient: httpClient,
		EndpointResolverWithOptions: customResolver,
	}

	// Create an S3 client with path-style addressing
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &R2Handler{
		client: client,
		logger: logger,
		config: cfg,
	}, nil
}

// ListFilesRequest represents the request body for listing files in a bucket
type ListFilesRequest struct {
	BucketName string `json:"bucket_name" binding:"required"`
}

// FileInfo represents a file in the bucket
type FileInfo struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
}

// GeneratePresignedURLRequest represents the request body for generating a presigned URL
type GeneratePresignedURLRequest struct {
	BucketName string `json:"bucket_name" binding:"required"`
	ObjectKey  string `json:"object_key" binding:"required"`
	ExpiresIn  int32  `json:"expires_in"` // in seconds
}

// PresignedURLResponse represents the response with presigned URL
type PresignedURLResponse struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	ExpiresAt int64  `json:"expires_at"`
	// Additional headers that should be included in the upload request
	Headers   map[string]string `json:"headers,omitempty"`
}

// ListFilesResponse represents the response with list of files
type ListFilesResponse struct {
	Files []FileInfo `json:"files"`
}

// ListFiles lists all files in the specified R2 bucket
// @Summary List files in R2 bucket
// @Description List all files in the specified Cloudflare R2 bucket
// @Tags cloudflare
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ListFilesRequest true "Bucket name"
// @Success 200 {object} ListFilesResponse
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /cloudflare/r2/files [post]
func (h *R2Handler) ListFiles(c *gin.Context) {
	correlationID, _ := c.Get("CorrelationID")
	correlationIDStr, _ := correlationID.(string)

	var req ListFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Str("correlation_id", correlationIDStr).Msg("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// List objects in the bucket
	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Str("bucket", req.BucketName).
		Msg("Listing objects in bucket")

	result, err := h.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(req.BucketName),
	})

	if err != nil {
		h.logger.Error().
			Err(err).
			Str("correlation_id", correlationIDStr).
			Str("bucket", req.BucketName).
			Msg("Failed to list objects")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Failed to list objects: " + err.Error(),
		})
		return
	}

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Str("bucket", req.BucketName).
		Int("object_count", len(result.Contents)).
		Msg("Successfully listed objects")

	// Convert to our response format
	files := make([]FileInfo, 0, len(result.Contents))
	if len(result.Contents) == 0 {
		h.logger.Info().
			Str("correlation_id", correlationIDStr).
			Str("bucket", req.BucketName).
			Msg("No objects found in bucket")
	}
	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)
		// Get file metadata
		headObj, err := h.client.HeadObject(context.Background(), &s3.HeadObjectInput{
			Bucket: aws.String(req.BucketName),
			Key:    obj.Key,
		})

		contentType := ""
		if err != nil {
			h.logger.Warn().
				Err(err).
				Str("correlation_id", correlationIDStr).
				Str("bucket", req.BucketName).
				Str("key", key).
				Msg("Failed to get object metadata, using empty content type")
		} else if headObj.ContentType != nil {
			contentType = *headObj.ContentType
			h.logger.Debug().
				Str("correlation_id", correlationIDStr).
				Str("bucket", req.BucketName).
				Str("key", key).
				Str("content_type", contentType).
				Msg("Retrieved object metadata")
		}

		// Safely dereference the Size pointer
		size := int64(0)
		if obj.Size != nil {
			size = *obj.Size
		}

		files = append(files, FileInfo{
			Key:          aws.ToString(obj.Key),
			LastModified: aws.ToTime(obj.LastModified),
			Size:         size,
			ContentType:  contentType,
		})
	}

	response := ListFilesResponse{
		Files: files,
	}

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Str("bucket", req.BucketName).
		Int("file_count", len(files)).
		Interface("files", files).
		Msg("Sending response")

	// Set CORS headers
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Send the response using Gin's JSON method
	c.JSON(http.StatusOK, response)
}

// GeneratePresignedURL generates a presigned URL for direct upload to R2
// @Summary Generate presigned URL for direct upload to R2
// @Description Generate a presigned URL that can be used to upload a file directly to R2
// @Tags cloudflare
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body GeneratePresignedURLRequest true "Presigned URL request"
// @Success 200 {object} PresignedURLResponse
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /cloudflare/r2/upload/presigned-url [post]
func (h *R2Handler) GeneratePresignedURL(c *gin.Context) {
	correlationID, _ := c.Get("CorrelationID")
	correlationIDStr, _ := correlationID.(string)

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Msg("Received request to generate presigned URL")

	var req GeneratePresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().
			Err(err).
			Str("correlation_id", correlationIDStr).
			Msg("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Str("bucket", req.BucketName).
		Str("object_key", req.ObjectKey).
		Int32("expires_in", req.ExpiresIn).
		Msg("Processing presigned URL request")

	// Set default expiry if not provided
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 3600 // 1 hour default
		h.logger.Debug().
			Str("correlation_id", correlationIDStr).
			Msg("Using default expiration time of 1 hour")
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a presigned URL with cache control headers
	presignClient := s3.NewPresignClient(h.client)

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Msg("Generating presigned URL...")
	
	// Generate the presigned URL
	presignResult, err := presignClient.PresignPutObject(ctx,
		&s3.PutObjectInput{
			Bucket:       aws.String(req.BucketName),
			Key:          aws.String(req.ObjectKey),
			CacheControl: aws.String("no-store, no-cache, must-revalidate, max-age=0"),
			ContentType:  aws.String("application/octet-stream"),
		},
		s3.WithPresignExpires(time.Duration(req.ExpiresIn)*time.Second),
	)
	
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("correlation_id", correlationIDStr).
			Str("bucket", req.BucketName).
			Str("object_key", req.ObjectKey).
			Msg("Failed to generate presigned URL")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   true,
			"message": "Failed to generate presigned URL: " + err.Error(),
		})
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second).Unix()

	// Add cache control headers that should be included in the upload request
	headers := map[string]string{
		"Cache-Control": "no-store, no-cache, must-revalidate, max-age=0",
		"Pragma":       "no-cache",
		"Expires":      "0",
	}

	response := PresignedURLResponse{
		URL:       presignResult.URL,
		Method:    "PUT",
		ExpiresAt: expiresAt,
		Headers:   headers,
	}

	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Str("bucket", req.BucketName).
		Str("object_key", req.ObjectKey).
		Str("url", response.URL).
		Int64("expires_at", response.ExpiresAt).
		Msg("Successfully generated presigned URL")

	// Log the response that will be sent
	h.logger.Info().
		Str("correlation_id", correlationIDStr).
		Str("url", response.URL).
		Str("method", response.Method).
		Int64("expires_at", response.ExpiresAt).
		Interface("headers", response.Headers).
		Msg("Generated presigned URL")

	// Log the response for debugging
	h.logger.Debug().
		Str("correlation_id", correlationIDStr).
		Interface("response", response).
		Msg("Sending response to client")

	// Set CORS headers
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Send the response using Gin's JSON method
	c.JSON(http.StatusOK, response)
}
