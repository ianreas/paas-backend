package v1

import (
	"context"
	"database/sql"

	"paas-backend/api/v1/controllers"
	"paas-backend/internal/repositories"
	"paas-backend/internal/services"
	"log"

	// http package
	"net/http"

	"github.com/gorilla/mux"
)

type Dependencies struct {
	ECRService     services.ECRService
	EKSService     services.EKSService
	LogService     services.LogService
	AppsRepository repositories.ApplicationsRepository
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

// NewDependencies initializes all the required dependencies.
func NewDependencies(ctx context.Context, db *sql.DB) (*Dependencies, error) {
	dockerService := services.NewDockerService()
	ecrRepo, err := repositories.NewECRRepository(ctx)
	if err != nil {
		return nil, err
	}

	eksService, err := services.NewEKSService(ctx)
	if err != nil {
		return nil, err
	}

	logService, err := services.NewLogService(ctx)
	if err != nil {
		return nil, err
	}

	ecrService := services.NewECRService(dockerService, ecrRepo, eksService)

	appsRepo := repositories.NewApplicationsRepository(db)

	return &Dependencies{
		ECRService:     ecrService,
		EKSService:     eksService,
		LogService:     logService,
		AppsRepository: appsRepo,
	}, nil
}

func RegisterRoutes(r *mux.Router, deps *Dependencies) {
	r.Use(LoggingMiddleware)
	r.HandleFunc("/items", controllers.GetItems).Methods("GET")
	// ... (other routes)
	r.HandleFunc("/build-and-push-deploy", controllers.BuildPushDeployApiHandler(
		deps.ECRService, deps.EKSService, deps.AppsRepository)).Methods("POST")
	r.HandleFunc("/logs/{appName}", controllers.StreamLogsHandler(deps.LogService)).Methods("GET")
}