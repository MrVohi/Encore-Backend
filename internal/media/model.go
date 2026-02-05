package media

import "time"

type Asset struct {
	ID        string
	Kind      string
	Storage   string
	ObjectKey string
	MimeType  string
	SizeBytes int64
	CreatedAt time.Time
}
