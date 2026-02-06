package geo

import "time"

type GeoCache struct {
	City      string     `json:"city"`
	Country   string     `json:"country"`
	Lat       float64    `json:"lat"`
	Lng       float64    `json:"lng"`
	CreatedAt *time.Time `json:"created_at"`
}
