package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"payment-service/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"
)

const (
	defaultUploadTimeout = 60 * time.Second
	maxUploadSize        = 5 * 1024 * 1024 // 5 MB
)

var allowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

type MinioStorageStruct struct {
	cfg    *config.Config
	client *minio.Client
}

type MinioStorageInterface interface {
	UploadFile(ctx context.Context, path string, fileBuffer *bytes.Buffer) (string, error)
	ProcessAndUploadImage(ctx context.Context, fileHeader *multipart.FileHeader) (string, error)
}

func NewMinioStorage(cfg *config.Config, client *minio.Client) MinioStorageInterface {
	return &MinioStorageStruct{
		cfg:    cfg,
		client: client,
	}
}

func (m *MinioStorageStruct) UploadFile(ctx context.Context, path string, fileBuffer *bytes.Buffer) (string, error) {
	ctx, cancel := context.WithTimeout(
		ctx,
		defaultUploadTimeout,
	)
	defer cancel()

	contentType := getContentType(path)

	_, err := m.client.PutObject(
		ctx,
		m.cfg.Storage.Bucket,
		path,
		bytes.NewReader(fileBuffer.Bytes()),
		int64(fileBuffer.Len()),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("bucket", m.cfg.Storage.Bucket).
			Str("path", path).
			Str("source", "internal.adapter.storage.UploadFile").
			Msg("failed upload file to minio")

		return "", err
	}

	publicURL := strings.TrimSuffix(
		m.cfg.Storage.PublicURL,
		"/",
	)

	url := fmt.Sprintf(
		"%s/%s/%s",
		publicURL,
		m.cfg.Storage.Bucket,
		path,
	)

	return url, nil
}

func (m *MinioStorageStruct) ProcessAndUploadImage(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", fmt.Errorf("image file is required")
	}

	if err := validateImage(fileHeader); err != nil {
		return "", err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	fileBuffer := new(bytes.Buffer)

	_, err = io.Copy(fileBuffer, src)
	if err != nil {
		return "", err
	}

	newFileName := fmt.Sprintf(
		"%s_%d%s",
		uuid.New().String(),
		time.Now().Unix(),
		getExtension(fileHeader.Filename),
	)

	uploadPath := fmt.Sprintf(
		"public/uploads/%s",
		newFileName,
	)

	url, err := m.UploadFile(ctx, uploadPath, fileBuffer)
	if err != nil {
		return "", err
	}

	return url, nil
}

func validateImage(file *multipart.FileHeader) error {
	if file == nil {
		return fmt.Errorf("image file is required")
	}

	if file.Size > maxUploadSize {
		return fmt.Errorf("image size exceeds limit 5 MB")
	}

	ext := strings.ToLower(
		filepath.Ext(file.Filename),
	)

	if !allowedImageExtensions[ext] {
		return fmt.Errorf(
			"invalid image extension: %s",
			ext,
		)
	}

	return nil
}

func getExtension(fileName string) string {
	return strings.ToLower(
		filepath.Ext(fileName),
	)
}

func getContentType(fileName string) string {
	switch getExtension(fileName) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
