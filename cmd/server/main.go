package main

import (
	"log"
	"net/http"

	"github.com/HaykAghajanyan/chat-backend/internal/config"
	"github.com/HaykAghajanyan/chat-backend/internal/database"
	"github.com/HaykAghajanyan/chat-backend/internal/handlers"
	"github.com/HaykAghajanyan/chat-backend/internal/middleware"
	"github.com/HaykAghajanyan/chat-backend/internal/repository"
	"github.com/HaykAghajanyan/chat-backend/internal/service"
	"github.com/HaykAghajanyan/chat-backend/internal/websocket"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	// Initialize database
	db, err := database.NewConnection(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run() // Start hub in background

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userRepo)
	messageHandler := handlers.NewMessageHandler(messageRepo, userRepo)
	wsHandler := handlers.NewWebSocketHandler(hub, messageRepo, authService)

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Public routes
	r.Get("/health", handlers.Health)
	r.Get("/db-health", handlers.DatabaseHealthCheck(db))

	// Auth routes
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// User routes
		r.Get("/api/users/me", userHandler.GetMe)
		r.Get("/api/users/search", userHandler.SearchUsers)

		// Message routes
		r.Get("/api/messages/conversations", messageHandler.GetConversationList)
		r.Get("/api/messages/conversation/{userID}", messageHandler.GetConversation)
		r.Get("/api/messages/unread-count", messageHandler.GetUnreadCount)
		r.Put("/api/messages/{messageID}/read", messageHandler.MarkAsRead)

		// WebSocket route
		r.Get("/ws", wsHandler.HandleWebSocket)
	})

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
