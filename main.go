package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Artist structure for translating from DB to JSON
type artist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Genre       string `json:"genre"`
	Image_URL   string `json:"image_url"`
	Preview_URL string `json:"preview_url"`
	Created_at  string `json:"created_at"`
}

// Initialize Artists
func InitArtists() []artist {
	artists := []artist{}

	return artists
}

// Connect backend to database
func conn_db(user string, pass string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:5432/Encore_DB", user, pass)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Optional tuning:
	// cfg.MaxConns = 10

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return pgxpool.NewWithConfig(ctx, cfg)
}

// Convert from database to JSON
func artistToJSON(dbID string, dbName string, dbGenre string, dbImageURL string, dbPreviewURL string, dbCreatedAT string) artist {
	return artist{ID: dbID, Name: dbName, Genre: dbGenre, Image_URL: dbImageURL, Preview_URL: dbPreviewURL, Created_at: dbCreatedAT}
}

// Find all artists in database
func queryArtists(ctx context.Context, db *pgxpool.Pool) ([]artist, error) {
	artists := []artist{}

	rows, err := db.Query(ctx, "SELECT id, name, genre, image_url, preview_url, created_at FROM artists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id pgtype.UUID
		var name string
		var genre string
		var image_url string
		var preview_url string
		var created_at time.Time

		if err := rows.Scan(&id, &name, &genre, &image_url, &preview_url, &created_at); err != nil {
			return nil, err
		}
		artists = append(artists, artistToJSON(id.String(), name, genre, image_url, preview_url, created_at.Format(time.RFC3339)))
	}

	return artists, rows.Err()
}

// Find artist by ID in database
func querySpecificArtist(ctx context.Context, db *pgxpool.Pool, artistID string) (artist, error) {
	var (
		id         pgtype.UUID
		name       string
		genre      string
		imageURL   string
		previewURL string
		createdAt  time.Time
	)

	err := db.QueryRow(ctx, `
		SELECT id, name, genre, image_url, preview_url, created_at
		FROM artists
		WHERE id = $1
	`, artistID).Scan(&id, &name, &genre, &imageURL, &previewURL, &createdAt)

	if err != nil {
		return artist{}, err
	}

	return artistToJSON(
		id.String(),
		name,
		genre,
		imageURL,
		previewURL,
		createdAt.Format(time.RFC3339),
	), nil
}

// Insert a new artist into the database
func insertArtist(ctx context.Context, db *pgxpool.Pool, a artist) (artist, error) {
	var (
		id        pgtype.UUID
		createdAt time.Time
	)

	err := db.QueryRow(ctx, `
		INSERT INTO artists (name, genre, image_url, preview_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, a.Name, a.Genre, a.Image_URL, a.Preview_URL).Scan(&id, &createdAt)

	if err != nil {
		return artist{}, err
	}

	a.ID = id.String()
	a.Created_at = createdAt.Format(time.RFC3339)
	return a, nil
}

// Receives new artis informations and transfer them to the db handler
func postArtist(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newArtist artist
		if err := c.BindJSON(&newArtist); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		a, err := insertArtist(c.Request.Context(), db, newArtist)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, a)
	}
}

// API Handler to find all artist in DB
func getArtists(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		artists, err := queryArtists(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.IndentedJSON(http.StatusOK, artists)
	}
}

// API Handler to find specific artist in DB
func getArtistByID(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		a, err := querySpecificArtist(c.Request.Context(), db, c.Param("id"))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.IndentedJSON(http.StatusOK, a)
	}
}

type searchArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func normalizeQuery(q string) string {
	return strings.Join(strings.Fields(q), " ")
}

func searchToJSON(dbID string, dbName string) artist {
	return artist{ID: dbID, Name: dbName}
}

func fetchSearch(ctx context.Context, db *pgxpool.Pool, q string, limit int) ([]searchArtist, error) {
	q = normalizeQuery(q)

	if len(q) < 2 {
		return []searchArtist{}, nil
	}

	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM artists
		WHERE lower(regexp_replace(name, '\s+', ' ', 'g'))
		      LIKE '%' || lower($1) || '%'
		ORDER BY name
		LIMIT $2
	`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]searchArtist, 0, limit)
	for rows.Next() {
		var id pgtype.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		results = append(results, searchArtist{
			ID:   id.String(),
			Name: name,
		})
	}
	return results, rows.Err()
}

func search(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")

		limit := 10
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
				limit = n
			}
		}

		results, err := fetchSearch(c.Request.Context(), db, q, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, results)
	}
}

func main() {
	// Load .env file
	_ = godotenv.Load(".env")

	// Grab vars in .env
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")

	// Handle connection to database
	db, err := conn_db(user, pass)
	if err != nil {
		panic(err)
	}

	// Close the database connection when the program exits.
	defer db.Close()

	// Config API routes
	router := gin.Default()
	config := cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173"
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}
	router.Use(cors.New(config))

	router.GET("/api/artists", getArtists(db))
	router.GET("/api/artists/:id", getArtistByID(db))
	router.POST("/api/artists", postArtist(db))
	router.GET("/api/search", search(db))

	router.Run("localhost:8080")
}
