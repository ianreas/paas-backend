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

	"fmt"

	 "github.com/aws/aws-sdk-go-v2/config"

	"github.com/gorilla/mux"
)

type Dependencies struct {
	ECRService     services.ECRService
	EKSService     services.EKSService
	LogService     services.LogService
	AppsRepository repositories.ApplicationsRepository
	MonitoringService services.MonitoringService
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

	 // Add AWS config initialization
	 cfg, err := config.LoadDefaultConfig(ctx, 
        config.WithRegion("us-east-1"), // Replace with your AWS region
    )
    if err != nil {
        return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
    }

	monitoringService := services.NewMonitoringService(cfg)

	ecrService := services.NewECRService(dockerService, ecrRepo, eksService)

	appsRepo := repositories.NewApplicationsRepository(db)

	return &Dependencies{
		ECRService:     ecrService,
		EKSService:     eksService,
		LogService:     logService,
		AppsRepository: appsRepo,
		MonitoringService: monitoringService,  // Add this line
	}, nil
}

func RegisterRoutes(r *mux.Router, deps *Dependencies) {
	r.Use(LoggingMiddleware)
	r.HandleFunc("/items", controllers.GetItems).Methods("GET")
	r.HandleFunc("/build-and-push-deploy", controllers.BuildPushDeployApiHandler(
		deps.ECRService, deps.EKSService, deps.AppsRepository)).Methods("POST")
	r.HandleFunc("/logs/{appName}", controllers.StreamLogsHandler(deps.LogService)).Methods("GET")
	r.HandleFunc("/metrics", controllers.GetMetricsHandler(deps.MonitoringService)).Methods("POST")
	
}