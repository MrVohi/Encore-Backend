package artist

import (
	"context"
	"mime/multipart"

	"groupie-tracker/internal/media"
)

type Service struct {
	repo  *Repository
	media *media.Service
}

func NewService(repo *Repository, mediaService *media.Service) *Service {
	return &Service{repo: repo, media: mediaService}
}

func (s *Service) UploadArtwork(ctx context.Context, artistID string, fileHeader *multipart.FileHeader) (Artist, error) {
	if _, err := s.repo.GetByID(ctx, artistID); err != nil {
		return Artist{}, err
	}

	asset, err := s.media.SaveAsset(ctx, fileHeader, "artwork", "image/")
	if err != nil {
		return Artist{}, err
	}

	if err := s.repo.UpdateArtworkAsset(ctx, artistID, asset.ID); err != nil {
		return Artist{}, err
	}

	return s.repo.GetByID(ctx, artistID)
}

func (s *Service) UploadPreview(ctx context.Context, artistID string, fileHeader *multipart.FileHeader) (Artist, error) {
	if _, err := s.repo.GetByID(ctx, artistID); err != nil {
		return Artist{}, err
	}

	asset, err := s.media.SaveAsset(ctx, fileHeader, "preview", "audio/")
	if err != nil {
		return Artist{}, err
	}

	if err := s.repo.UpdatePreviewAsset(ctx, artistID, asset.ID); err != nil {
		return Artist{}, err
	}

	return s.repo.GetByID(ctx, artistID)
}

func (s *Service) DeleteArtist(ctx context.Context, artistID string) error {
	a, err := s.repo.GetByID(ctx, artistID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteByID(ctx, artistID); err != nil {
		return err
	}

	if a.ArtworkAssetID != nil && *a.ArtworkAssetID != "" {
		_ = s.media.DeleteAsset(ctx, *a.ArtworkAssetID, a.ArtworkObjectKey)
	}
	if a.PreviewAssetID != nil && *a.PreviewAssetID != "" {
		_ = s.media.DeleteAsset(ctx, *a.PreviewAssetID, a.PreviewObjectKey)
	}

	return nil
}
