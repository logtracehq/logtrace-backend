package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/adelowo/gulter"
	"github.com/adelowo/gulter/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCreds "github.com/aws/aws-sdk-go-v2/credentials"
	"gitlab.com/logtrace/logtrace/config"
)

// S3StorageService handles S3 storage initialization and configuration
type S3StorageService struct {
	cfg *config.Config
}

// NewS3StorageService creates a new S3 storage service
func NewS3StorageService(cfg *config.Config) *S3StorageService {
	return &S3StorageService{
		cfg: cfg,
	}
}

// InitializeStorage creates and configures an S3 storage client
func (s *S3StorageService) InitializeStorage(ctx context.Context) (gulter.Storage, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !s.cfg.Uploader.S3.UseTLS,
			},
		},
	}

	s3Config, err := awsConfig.LoadDefaultConfig(
		ctx,
		awsConfig.WithRegion(s.cfg.Uploader.S3.Region),
		awsConfig.WithHTTPClient(httpClient),
		awsConfig.WithCredentialsProvider(
			awsCreds.NewStaticCredentialsProvider(
				s.cfg.Uploader.S3.AccessKey,
				s.cfg.Uploader.S3.AccessSecret,
				"")),
		//nolint:staticcheck
		awsConfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			//nolint:staticcheck
			return aws.Endpoint{
				URL:               s.cfg.Uploader.S3.Endpoint,
				SigningRegion:     s.cfg.Uploader.S3.Region,
				HostnameImmutable: true,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	s3Store, err := storage.NewS3FromConfig(s3Config, storage.S3Options{
		DebugMode:        s.cfg.Uploader.S3.LogOperations,
		UsePathStyle:     true,
		Bucket:           s.cfg.Uploader.S3.Bucket,
		CloudflareDomain: s.cfg.Uploader.S3.CloudflareBucketDomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return s3Store, nil
}
