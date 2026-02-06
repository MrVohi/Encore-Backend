package geo

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

type Geocoder interface {
	Geocode(ctx context.Context, city, country string) (lat, lng float64, err error)
}

type Service struct {
	repo     *Repository
	geocoder Geocoder
}

func NewService(repo *Repository, geocoder Geocoder) *Service {
	return &Service{repo: repo, geocoder: geocoder}
}

var spaceRe = regexp.MustCompile(`\s+`)

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = spaceRe.ReplaceAllString(s, " ")
	s = strings.ToLower(s)
	return s
}

// ResolveCoords: cache lookup -> geocode -> store -> return
func (s *Service) ResolveCoords(ctx context.Context, city, country string) (float64, float64, error) {
	if strings.TrimSpace(city) == "" || strings.TrimSpace(country) == "" {
		return 0, 0, errors.New("city and country are required")
	}

	cityN := normalize(city)
	countryN := normalize(country)

	lat, lng, ok, err := s.repo.Get(ctx, cityN, countryN)
	if err != nil {
		return 0, 0, err
	}
	if ok {
		return lat, lng, nil
	}

	lat, lng, err = s.geocoder.Geocode(ctx, city, country)
	if err != nil {
		return 0, 0, err
	}

	if err := s.repo.Upsert(ctx, cityN, countryN, lat, lng); err != nil {
		return 0, 0, err
	}

	return lat, lng, nil
}
