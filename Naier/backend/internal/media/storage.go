package media

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client   *minio.Client
	endpoint string
	secure   bool
}

func NewStorage(endpoint, accessKey, secretKey string, secure bool) (*Storage, error) {
	normalizedEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	if endpoint != "" {
		secure = isSecureEndpoint(endpoint)
	}

	client, err := minio.New(normalizedEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Storage{
		client:   client,
		endpoint: normalizedEndpoint,
		secure:   secure,
	}, nil
}

func (s *Storage) UploadFile(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload object: %w", err)
	}

	scheme := "http"
	if s.secure {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, bucket, url.PathEscape(objectName)), nil
}

func (s *Storage) DeleteFile(ctx context.Context, bucket, objectName string) error {
	if err := s.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

func (s *Storage) GeneratePresignedURL(bucket, objectName string, expiry time.Duration) (string, error) {
	presigned, err := s.client.PresignedGetObject(context.Background(), bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("generate presigned url: %w", err)
	}

	return presigned.String(), nil
}

func isSecureEndpoint(endpoint string) bool {
	return strings.HasPrefix(strings.ToLower(endpoint), "https://")
}
