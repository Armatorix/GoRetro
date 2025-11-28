package main

import (
	"context"
	"database/sql"
	"embed"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/Armatorix/GoRetro/internal/chatcompletion"
	"github.com/Armatorix/GoRetro/internal/handlers"
	"github.com/Armatorix/GoRetro/internal/models"
	"github.com/Armatorix/GoRetro/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

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

	// Initialize Redis if REDIS_URL is set (for distributed mode)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		log.Printf("Connecting to Redis at %s", redisURL)
		rdb := redis.NewClient(&redis.Options{
			Addr: redisURL,
		})

		// Test Redis connection
		ctx := context.Background()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v. Running in local-only mode.", err)
		} else {
			log.Println("Connected to Redis successfully")
			// Set up Redis pub/sub for distributed synchronization
			redisPubSub := websocket.NewRedisPubSub(rdb, hub)
			hub.SetRedisPubSub(redisPubSub)
			go redisPubSub.Start()
			log.Println("Redis pub/sub enabled for distributed synchronization")
		}
	} else {
		log.Println("REDIS_URL not set, running in local-only mode")
	}

	// Get chat completion configuration from environment (optional)
	chatEndpoint := os.Getenv("CHAT_COMPLETION_ENDPOINT")
	chatAPIKey := os.Getenv("CHAT_COMPLETION_API_KEY")
	chatModel := os.Getenv("CHAT_COMPLETION_MODEL")
	if chatModel == "" {
		chatModel = "gpt-4" // Default model
	}

	if chatEndpoint != "" && chatAPIKey != "" {
		log.Printf("Chat completion API configured - auto-merge feature enabled (model: %s)", chatModel)
		chatService := chatcompletion.NewService(chatEndpoint, chatAPIKey, chatModel)
		hub.SetChatCompletion(chatService)
	} else {
		log.Println("Chat completion API not configured - auto-merge feature disabled")
	}

	// Initialize handlers
	h := handlers.NewHandler(store, hub, chatEndpoint, chatAPIKey) // Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Parse templates
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	e.Renderer = &TemplateRenderer{templates: tmpl}

	// Serve static files from embedded filesystem
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub filesystem: %v", err)
	}
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS)))))

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
