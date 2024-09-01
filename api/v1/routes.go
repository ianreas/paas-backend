package v1

import (
	"context"

	"net/http"
	"paas-backend/api/v1/controllers"
	"paas-backend/internal/repositories"
	"paas-backend/internal/services"

	"github.com/gorilla/mux"
)

// meow

type Dependencies struct {
	ECRService services.ECRService
	EKSService services.EKSService
	LogService services.LogService
}

// /
func NewDependencies(ctx context.Context) (*Dependencies, error) {
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

	return &Dependencies{
		ECRService: ecrService,
		EKSService: eksService,
		LogService: logService,
	}, nil
}

func RegisterRoutes(r *mux.Router, deps *Dependencies) {
	r.HandleFunc("/items", controllers.GetItems).Methods("GET")
	r.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
	r.HandleFunc("/items", controllers.CreateItem).Methods("POST")
	r.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
	r.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
	r.HandleFunc("/users", controllers.AddUserHandler).Methods("POST")
	r.HandleFunc("/ec2", controllers.CreateEC2InstanceHandler).Methods("POST")
	r.HandleFunc("/rds", controllers.CreateRDSInstanceHandler).Methods("POST")
	r.HandleFunc("/deploy", controllers.DeployHandler).Methods("POST")
	r.HandleFunc("/build-and-push-deploy", controllers.BuildPushDeployApiHandler(deps.ECRService, deps.EKSService)).Methods("POST")
	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}).Methods("GET")
	r.HandleFunc("/logs/{appName}", controllers.StreamLogsHandler(deps.LogService)).Methods("GET")
}

// thread safety for build-and-push route
// 1) Dependencies Initialization in main.go:
// deps, err := v1.NewDependencies(ctx) => the dependencies are initialized once at the application start, not per request
// 2) route registration: routes are registered using the initialized dependencies
// 3) handler function: the BuildAndPushToECRApiHandler is used which is a closure that captures the dependencies: ECRService and EKSService
// this suggests that:
// 1) each request gets its own instance of the handler function
// 2) the services ECRService and EKSService are shared across the requests but they are designed as stateless and threadsafe:
// 3) Any state specific to a request (like the request body) is handled within the scope of each request handler.
