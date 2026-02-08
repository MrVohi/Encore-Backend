package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"groupie-tracker/internal/storage"
	"groupie-tracker/pkg/utils"
)

var ErrInvalidMimeType = errors.New("invalid mime type")

type Service struct {
	repo  *Repository
	store storage.Driver
}

func NewService(repo *Repository, store storage.Driver) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) SaveAsset(ctx context.Context, fileHeader *multipart.FileHeader, kind string, allowedPrefix string) (Asset, error) {
	mimeType, err := sniffMimeType(fileHeader)
	if err != nil {
		return Asset{}, err
	}
	if !strings.HasPrefix(mimeType, allowedPrefix) {
		return Asset{}, fmt.Errorf("%w: %s", ErrInvalidMimeType, mimeType)
	}

	token, err := utils.GenerateRandomToken(16)
	if err != nil {
		return Asset{}, err
	}

	ext := filepath.Ext(fileHeader.Filename)
	objectKey := strings.TrimLeft(kind, "/") + "/" + token + ext

	src, err := fileHeader.Open()
	if err != nil {
		return Asset{}, err
	}
	defer src.Close()

	cr := &countingReader{r: src}
	if err := s.store.Put(ctx, objectKey, cr, mimeType); err != nil {
		return Asset{}, err
	}
	size := cr.n
	if size == 0 && fileHeader.Size > 0 {
		size = fileHeader.Size
	}

	created, err := s.repo.Create(ctx, Asset{
		Kind:      kind,
		Storage:   s.store.Name(),
		ObjectKey: objectKey,
		MimeType:  mimeType,
		SizeBytes: size,
	})
	if err != nil {
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
		if err := s.store.Delete(ctx, *objectKey); err != nil {
			return err
		}
	}

	if err := s.repo.DeleteByID(ctx, assetID); err != nil {
		return err
	}

	return nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
