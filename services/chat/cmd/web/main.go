package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
	"html/template"
	
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/gorilla/websocket"
	
	"github.com/Tauhid-UAP/global-chat/services/chat/core/handlers"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/store"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/redisclient"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/awsclient"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/websockethandlers"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/chat"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/sfuclient"
	"github.com/Tauhid-UAP/global-chat/services/chat/core/config"
)

var Version = "development"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found: %v\n", err)
	}
	
	DATABASE_URL := os.Getenv("DATABASE_URL")
	log.Printf("DATABASE_URL: %s", DATABASE_URL)

	db, err := sql.Open("postgres", DATABASE_URL)
	if err != nil {
		log.Fatal(err)
	}
	store.DB = db

	redisclient.Init()
	if err := redisclient.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}

	awsclient.Init()

	mux := http.NewServeMux()

	cfg := config.Load()
	if cfg.Debug {
		// Static files
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	}
	
	staticAssetBaseURL := template.URL(cfg.StaticAssetBaseURL)

	// Public routes
	mux.HandleFunc("/register", handlers.RegisterHandler(staticAssetBaseURL))
	mux.HandleFunc("/login", handlers.LoginHandler(staticAssetBaseURL))
	
	// Protected routes
	protected := http.NewServeMux()
	log.Printf("STATIC BASE: %s", cfg.StaticAssetBaseURL)
	protected.HandleFunc("/", handlers.Profile)
	protected.HandleFunc("/logout", handlers.Logout)
	
	protectedHandler := middleware.AuthMiddleware(middleware.CSRFMiddleware(protected))
	
	// Routes for both authenticated and anonymous users
	optionalAuthMux := http.NewServeMux()
	optionalAuthMux.HandleFunc("/chat", handlers.ChatPageHandler(staticAssetBaseURL))

	hub := chat.CreateHub()
	sfuClient := sfuclient.NewSFUClient("127.0.0.1:50051")
	websocketUpgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {return true},
	}
	optionalAuthMux.HandleFunc("/ws/chat", websockethandlers.ChatHandler(websocketUpgrader, hub))

	optionalAuthHandler := middleware.OptionalAuthMiddleware(middleware.CSRFMiddleware(optionalAuthMux))

	mux.Handle("/chat", optionalAuthHandler)
	mux.Handle("/ws/chat", optionalAuthHandler)

	mux.Handle("/", protectedHandler)
	
	addr := ":8000"
	server := &http.Server{
		Addr: addr,
		Handler: loggingMiddleware(mux),
		ReadTimeout: 10*time.Second,
		WriteTimeout: 10*time.Second,
		IdleTimeout: 60*time.Second,
	}

	log.Println("Server running on ", addr)
	log.Fatal(server.ListenAndServe())
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
