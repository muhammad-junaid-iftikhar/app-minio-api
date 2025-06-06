package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	// Server configuration
	ServerPort string `mapstructure:"SERVER_PORT"`
	AppEnv     string `mapstructure:"APP_ENV"`

	// MinIO configuration
	MinioEndpoint   string `mapstructure:"MINIO_ENDPOINT"`
	MinioPort       string `mapstructure:"MINIO_PORT"`
	MinioAccessKey  string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey  string `mapstructure:"MINIO_SECRET_KEY"`
	MinioUseSSL     bool   `mapstructure:"MINIO_USE_SSL"`
	MinioBucketName string `mapstructure:"MINIO_BUCKET_NAME"`

	// Cloudflare R2 configuration
	R2AccountID       string `mapstructure:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `mapstructure:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `mapstructure:"R2_SECRET_ACCESS_KEY"`
	R2PublicURL       string `mapstructure:"R2_PUBLIC_URL"`
	R2Region          string `mapstructure:"R2_REGION"`

	// Presigned URL configuration
	PresignedURLExpiry int64 `mapstructure:"PRESIGNED_URL_EXPIRY"` // in seconds
	MaxFileSize       int64 `mapstructure:"MAX_FILE_SIZE"`         // in bytes
}

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() {
	// Try to load .env file but don't fail if it doesn't exist
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Warn().Err(err).Msg("Failed to load .env file")
		} else {
			log.Info().Msg("Loaded environment variables from .env file")
		}
	}
}

// LoadConfig loads the configuration from environment variables with .env fallback
func LoadConfig() (*Config, error) {
	// Load environment variables from .env file if it exists (development only)
	loadEnvFile()

	// Set up Viper to read from environment variables
	viper.AutomaticEnv()
	// Allow viper to read environment variables with _
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults()

	// Bind environment variables to viper
	bindEnvVars()

	// Create config instance and unmarshal
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug: Print the loaded configuration
	log.Info().
		Str("server_port", cfg.ServerPort).
		Str("minio_endpoint", cfg.MinioEndpoint).
		Str("minio_port", cfg.MinioPort).
		Msg("Loaded configuration")

	return &cfg, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("APP_ENV", "development")

	// MinIO defaults
	viper.SetDefault("MINIO_ENDPOINT", "localhost")
	viper.SetDefault("MINIO_PORT", "9000")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("MINIO_BUCKET_NAME", "api-uploads")

	// R2 defaults
	viper.SetDefault("R2_REGION", "auto")
	viper.SetDefault("PRESIGNED_URL_EXPIRY", 3600) // 1 hour in seconds
	viper.SetDefault("MAX_FILE_SIZE", 104857600)        // 100MB in bytes
}

func bindEnvVars() {
	// Server env vars
	_ = viper.BindEnv("SERVER_PORT")
	_ = viper.BindEnv("APP_ENV")

	// MinIO env vars
	_ = viper.BindEnv("MINIO_ENDPOINT")
	_ = viper.BindEnv("MINIO_PORT")
	_ = viper.BindEnv("MINIO_ACCESS_KEY")
	_ = viper.BindEnv("MINIO_SECRET_KEY")
	_ = viper.BindEnv("MINIO_USE_SSL")
	_ = viper.BindEnv("MINIO_BUCKET_NAME")

	// R2 env vars
	_ = viper.BindEnv("R2_ACCOUNT_ID")
	_ = viper.BindEnv("R2_ACCESS_KEY_ID")
	_ = viper.BindEnv("R2_SECRET_ACCESS_KEY")
	_ = viper.BindEnv("R2_PUBLIC_URL")
	_ = viper.BindEnv("R2_REGION")

	// Presigned URL env vars
	_ = viper.BindEnv("PRESIGNED_URL_EXPIRY")
	_ = viper.BindEnv("MAX_FILE_SIZE")
}

// InitMinioClient initializes a MinIO client
func InitMinioClient(cfg *Config) (*minio.Client, error) {
	// Initialize MinIO client
	// Simply combine the endpoint and port as provided in the config
	endpoint := cfg.MinioEndpoint + ":" + cfg.MinioPort

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check if the bucket exists, create it if it doesn't
	exists, err := client.BucketExists(context.Background(), cfg.MinioBucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = client.MakeBucket(context.Background(), cfg.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger.Info().Str("bucket", cfg.MinioBucketName).Msg("Created bucket")
	} else {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger.Info().Str("bucket", cfg.MinioBucketName).Msg("Bucket already exists")
	}

	return client, nil
}