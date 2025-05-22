package config

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort      string `mapstructure:"SERVER_PORT"`
	MinioEndpoint   string `mapstructure:"MINIO_ENDPOINT"`
	MinioPort       string `mapstructure:"MINIO_PORT"`
	MinioAccessKey  string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey  string `mapstructure:"MINIO_SECRET_KEY"`
	MinioUseSSL     bool   `mapstructure:"MINIO_USE_SSL"`
	MinioBucketName string `mapstructure:"MINIO_BUCKET_NAME"`
}

func LoadConfig() (*Config, error) {
	// Set up Viper to read from environment variables
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // Automatically read environment variables

	// Explicitly bind each environment variable to its corresponding struct field
	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("MINIO_ENDPOINT")
	viper.BindEnv("MINIO_PORT")
	viper.BindEnv("MINIO_ACCESS_KEY")
	viper.BindEnv("MINIO_SECRET_KEY")
	viper.BindEnv("MINIO_USE_SSL")
	viper.BindEnv("MINIO_BUCKET_NAME")

	// Load .env file if it exists, but don't fail if it doesn't
	if err := viper.ReadInConfig(); err != nil {
		// If the error is due to the config file not being found, we'll just
		// use environment variables, so we can ignore this error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return an error if it's not a ConfigFileNotFoundError
			// This means there was a problem with the file itself, not that it was missing
			if _, ok := err.(*os.PathError); !ok {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
		}
		// Log that we're proceeding with environment variables
		fmt.Println("No .env file found or could not be read, using environment variables")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug: Print the loaded configuration
	fmt.Printf("Loaded configuration: SERVER_PORT=%s, MINIO_ENDPOINT=%s, MINIO_PORT=%s\n", 
		cfg.ServerPort, cfg.MinioEndpoint, cfg.MinioPort)

	return &cfg, nil
}

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
		fmt.Printf("Created bucket: %s\n", cfg.MinioBucketName)
	} else {
		fmt.Printf("Bucket already exists: %s\n", cfg.MinioBucketName)
	}

	return client, nil
}