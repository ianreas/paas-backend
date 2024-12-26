package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	v1 "paas-backend/api/v1"

	_ "github.com/lib/pq" // Import the PostgreSQL driver

	"github.com/gorilla/mux"

	"os"

	"github.com/gorilla/handlers"
)

func main() {
	// Create a background context
	ctx := context.Background()

	// Initialize the database connection
	dataSourceName := "host=paas-backend-1.cbigmg0cgxs7.us-east-1.rds.amazonaws.com port=5432 user=postgres password=muhammedik10 dbname=paas_backend sslmode=require"

	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Initialize dependencies with the database connection
	deps, err := v1.NewDependencies(ctx, db)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

	// Set up the router
	r := mux.NewRouter()

	// Middleware (if any)
	// r.Use(middleware.YourMiddleware)

	// Register routes
	v1.RegisterRoutes(r, deps)

	allowedOrigins := []string{"*"}

	// Create the CORS middleware handler
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins(allowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	port := os.Getenv("PORT")
    if port == "" {
        port = "3005" // fallback port
    }

    log.Printf("Server listening on port %s", port)

	// Start the server
	log.Println("Server listening on port 3005")
	if err := http.ListenAndServe(":3005", corsHandler(r)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}