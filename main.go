package main

import (
	"database/sql"
	"embed"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Armatorix/GoRetro/internal/handlers"
	"github.com/Armatorix/GoRetro/internal/models"
	"github.com/Armatorix/GoRetro/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

//go:embed templates/*.html
var templateFS embed.FS

// TemplateRenderer is a custom html/template renderer for Echo
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://goretro:goretro@localhost:5432/goretro?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ping database to verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Initialize store and hub
	store := models.NewRoomStore(db)

	// Initialize database schema
	if err := store.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	log.Println("Database schema initialized")

	hub := websocket.NewHub(store)
	go hub.Run()

	// Initialize handlers
	h := handlers.NewHandler(store, hub)

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Parse templates
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	e.Renderer = &TemplateRenderer{templates: tmpl}

	// Routes
	e.GET("/", h.Index)
	// Room routes
	e.POST("/rooms", h.CreateRoom)
	e.GET("/rooms", h.ListRooms)
	e.GET("/rooms/:id", h.GetRoom)
	e.DELETE("/rooms/:id", h.DeleteRoom)

	// API routes
	e.GET("/api/rooms/:id", h.GetRoomAPI)

	// WebSocket
	e.GET("/ws/:id", h.WebSocket)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
