package geo

import (
	"context"
	"fmt"
	"strings"
)

type DummyGeocoder struct{}

func (DummyGeocoder) Geocode(ctx context.Context, city, country string) (float64, float64, error) {
	key := strings.ToLower(strings.TrimSpace(city)) + "|" + strings.ToLower(strings.TrimSpace(country))

	coords := map[string][2]float64{
		"paris|fr":     {48.8566, 2.3522},
		"lyon|fr":      {45.7640, 4.8357},
		"marseille|fr": {43.2965, 5.3698},
		"berlin|de":    {52.5200, 13.4050},
		"london|gb":    {51.5074, -0.1278},
		"bruxelles|be": {50.8503, 4.3517},
		"amsterdam|nl": {52.3676, 4.9041},
		"barcelona|es": {41.3851, 2.1734},
	}

	if v, ok := coords[key]; ok {
		return v[0], v[1], nil
	}
	return 0, 0, fmt.Errorf("dummy geocoder: unknown location %q", key)
}
