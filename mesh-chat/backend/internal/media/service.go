package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var (
	ErrMediaUnavailable = errors.New("media storage unavailable")
	ErrInvalidMimeType  = errors.New("unsupported media type")
	ErrFileTooLarge     = errors.New("file exceeds size limit")
)

type UploadResponse struct {
	URL          string `json:"url"`
	ObjectPath   string `json:"object_path"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type Service struct {
	storage *Storage
	bucket  string
}

func NewService(storage *Storage, bucket string) *Service {
	return &Service{
		storage: storage,
		bucket:  bucket,
	}
}

func (s *Service) Upload(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (UploadResponse, error) {
	if s.storage == nil {
		return UploadResponse{}, ErrMediaUnavailable
	}

	file, err := fileHeader.Open()
	if err != nil {
		return UploadResponse{}, fmt.Errorf("open upload: %w", err)
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		sniff := make([]byte, 512)
		n, _ := file.Read(sniff)
		contentType = http.DetectContentType(sniff[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return UploadResponse{}, fmt.Errorf("rewind upload: %w", err)
		}
	}

	maxSize, err := s.allowedSize(contentType)
	if err != nil {
		return UploadResponse{}, err
	}
	if fileHeader.Size > maxSize {
		return UploadResponse{}, ErrFileTooLarge
	}

	body, err := io.ReadAll(file)
	if err != nil {
		return UploadResponse{}, fmt.Errorf("read upload: %w", err)
	}

	objectPath := s.buildObjectPath(userID, fileHeader.Filename, contentType)
	url, err := s.storage.UploadFile(ctx, s.bucket, objectPath, bytes.NewReader(body), int64(len(body)), contentType)
	if err != nil {
		return UploadResponse{}, err
	}

	response := UploadResponse{
		URL:        url,
		ObjectPath: objectPath,
	}

	if strings.HasPrefix(contentType, "image/") {
		thumbBody, thumbType, thumbErr := generateThumbnail(body)
		if thumbErr == nil {
			thumbPath := strings.TrimSuffix(objectPath, filepath.Ext(objectPath)) + "_thumb.jpg"
			thumbURL, uploadErr := s.storage.UploadFile(ctx, s.bucket, thumbPath, bytes.NewReader(thumbBody), int64(len(thumbBody)), thumbType)
			if uploadErr == nil {
				response.ThumbnailURL = thumbURL
			}
		}
	}

	return response, nil
}

func (s *Service) PresignedURL(objectPath string, expiry time.Duration) (string, error) {
	if s.storage == nil {
		return "", ErrMediaUnavailable
	}

	return s.storage.GeneratePresignedURL(s.bucket, objectPath, expiry)
}

func (s *Service) allowedSize(contentType string) (int64, error) {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return 10 * 1024 * 1024, nil
	case "video/mp4", "application/pdf":
		return 50 * 1024 * 1024, nil
	default:
		return 0, ErrInvalidMimeType
	}
}

func (s *Service) buildObjectPath(userID uuid.UUID, filename, contentType string) string {
	now := time.Now().UTC()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = extensionForContentType(contentType)
	}

	return fmt.Sprintf("%s/%04d/%02d/%s%s", userID.String(), now.Year(), int(now.Month()), uuid.NewString(), ext)
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

func generateThumbnail(body []byte) ([]byte, string, error) {
	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return nil, "", errors.New("invalid image dimensions")
	}

	maxSide := 320
	dstWidth := width
	dstHeight := height
	if width >= height && width > maxSide {
		dstWidth = maxSide
		dstHeight = int(float64(height) * (float64(maxSide) / float64(width)))
	} else if height > width && height > maxSide {
		dstHeight = maxSide
		dstWidth = int(float64(width) * (float64(maxSide) / float64(height)))
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), "image/jpeg", nil
}
