package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"groupie-tracker/pkg/utils"
)

var ErrInvalidMimeType = errors.New("invalid mime type")

type Service struct {
	repo      *Repository
	uploadDir string
}

func NewService(repo *Repository, uploadDir string) *Service {
	return &Service{repo: repo, uploadDir: uploadDir}
}

func (s *Service) SaveAsset(ctx context.Context, fileHeader *multipart.FileHeader, kind string, allowedPrefix string) (Asset, error) {
	mimeType, err := sniffMimeType(fileHeader)
	if err != nil {
		return Asset{}, err
	}
	if !strings.HasPrefix(mimeType, allowedPrefix) {
		return Asset{}, fmt.Errorf("%w: %s", ErrInvalidMimeType, mimeType)
	}

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return Asset{}, err
	}

	token, err := utils.GenerateRandomToken(16)
	if err != nil {
		return Asset{}, err
	}

	ext := filepath.Ext(fileHeader.Filename)
	objectKey := token + ext
	path := filepath.Join(s.uploadDir, objectKey)

	src, err := fileHeader.Open()
	if err != nil {
		return Asset{}, err
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return Asset{}, err
	}
	defer dst.Close()

	size, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(path)
		return Asset{}, err
	}

	created, err := s.repo.Create(ctx, Asset{
		Kind:      kind,
		Storage:   "local",
		ObjectKey: objectKey,
		MimeType:  mimeType,
		SizeBytes: size,
	})
	if err != nil {
		_ = os.Remove(path)
		return Asset{}, err
	}

	return created, nil
}

func sniffMimeType(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	return http.DetectContentType(buf[:n]), nil
}

func (s *Service) DeleteAsset(ctx context.Context, assetID string, objectKey *string) error {
	if objectKey != nil && *objectKey != "" {
		path := filepath.Join(s.uploadDir, *objectKey)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if err := s.repo.DeleteByID(ctx, assetID); err != nil {
		return err
	}

	return nil
}
