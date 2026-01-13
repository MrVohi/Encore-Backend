package search

type SearchResult struct {
	Kind     string `json:"kind"` // "artist" | "album" | "track"
	ID       string `json:"id"`
	Label    string `json:"label"`     // display string
	ParentID string `json:"parent_id"` // album->artist_id, track->album_id, artists->"" (optional)
}
