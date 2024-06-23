package main

import (
	"context"
	"log"
	"net/http"
	"paas-backend/api/v1"
	"paas-backend/internal/db"
	"paas-backend/internal/middleware"
	"paas-backend/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Initialize database
	dataSourceName := "host=paas-backend-1.cbigmg0cgxs7.us-east-1.rds.amazonaws.com port=5432 user=postgres password=muhammedik10 dbname=paas_backend sslmode=require"
	db.InitDB(dataSourceName)

	// Initialize AWS services
	ctx := context.Background()
	if err := services.InitAWSServices(ctx); err != nil {
		log.Fatalf("Failed to initialize AWS services: %v", err)
	}

	// Middleware
	r.Use(middleware.LoggingMiddleware)

	// Register routes
	v1.RegisterRoutes(r.PathPrefix("/api/v1").Subrouter())

	// Start server
	log.Println("Server listening on port 3005")
	log.Fatal(http.ListenAndServe(":3005", r))
}