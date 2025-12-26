package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
func conn_db(user string, pass string) (*pgx.Conn, error) {
	// Capture connection
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@localhost:5432/Encore_DB", user, pass))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Convert from database to JSON
func artistToJSON(dbID string, dbName string, dbGenre string, dbImageURL string, dbPreviewURL string, dbCreatedAT string) artist {
	return artist{ID: dbID, Name: dbName, Genre: dbGenre, Image_URL: dbImageURL, Preview_URL: dbPreviewURL, Created_at: dbCreatedAT}
}

// Find all artists in database
func queryArtists(ctx context.Context, conn *pgx.Conn) ([]artist, error) {
	artists := []artist{}

	rows, err := conn.Query(ctx, "SELECT id, name, genre, image_url, preview_url, created_at FROM artists")
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
func querySpecificArtist(ctx context.Context, conn *pgx.Conn, artistID string) (artist, error) {
	var (
		id         pgtype.UUID
		name       string
		genre      string
		imageURL   string
		previewURL string
		createdAt  time.Time
	)

	err := conn.QueryRow(ctx, `
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
func insertArtist(ctx context.Context, conn *pgx.Conn, a artist) (artist, error) {
	var (
		id        pgtype.UUID
		createdAt time.Time
	)

	err := conn.QueryRow(ctx, `
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
func postArtist(conn *pgx.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newArtist artist
		if err := c.BindJSON(&newArtist); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		a, err := insertArtist(c.Request.Context(), conn, newArtist)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, a)
	}
}

// API Handler to find all artist in DB
func getArtists(conn *pgx.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		artists, err := queryArtists(c.Request.Context(), conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.IndentedJSON(http.StatusOK, artists)
	}
}

// API Handler to find specific artist in DB
func getArtistByID(conn *pgx.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		a, err := querySpecificArtist(c.Request.Context(), conn, c.Param("id"))
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
	defer db.Close(context.Background())

	// Config API routes
	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://127.0.0.1:5500"}
	router.Use(cors.New(config))
	router.GET("/api/artists", getArtists(db))
	router.GET("/api/artists/:id", getArtistByID(db))
	router.POST("/api/artists", postArtist(db))

	router.Run("localhost:8080")
}
