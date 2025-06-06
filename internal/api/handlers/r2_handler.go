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

	var req GeneratePresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Str("correlation_id", correlationIDStr).Msg("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Set default expiry if not provided
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 3600 // 1 hour default
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a presigned URL with cache control headers
	presignClient := s3.NewPresignClient(h.client)
	
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
		h.logger.Error().Err(err).
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

	c.JSON(http.StatusOK, response)
}
