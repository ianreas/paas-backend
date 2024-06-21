package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux"
    "paas-backend/api/v1"
    "paas-backend/internal/middleware"
)

func main() {
    r := mux.NewRouter()

    // Middleware
    r.Use(middleware.LoggingMiddleware)

    // Register routes
    v1.RegisterRoutes(r.PathPrefix("/api/v1").Subrouter())

    // Start server
    log.Println("Server listening on port 3005")
    log.Fatal(http.ListenAndServe(":3005", r))
}
