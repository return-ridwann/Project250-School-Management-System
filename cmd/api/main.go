package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"sms/internal/database"
	"sms/internal/handlers"
	"sms/internal/repository"
	"sms/internal/services"
)

func main() {
	// Initialize SQLite db
	db, err := database.InitDB("school.db", "schema.sql")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run the seeder on startup
	if err := repository.SeedDefaultAdmin(db); err != nil {
		log.Printf("Seeder note: %v", err)
	}

	// Initialize Dependency Injection Chain
	authRepo := repository.NewAuthrepo(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev_fallback_secret_key_change_in_prod"
		log.Println("Warning: JWT_SECRET not set, using default development key")
	}
	authService := services.NewAuthService(authRepo, jwtSecret)
	authHandler := handlers.NewAuthHandler(authService)

	// Setup Router
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status":   "error",
				"database": "disconnected",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	})

	// Register Auth Routes
	authHandler.RegisterRoutes(mux)

	// Start Server
	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}