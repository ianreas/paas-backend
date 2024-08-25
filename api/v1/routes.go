package v1

import (
	"context"

	"net/http"
	"paas-backend/api/v1/controllers"
	"paas-backend/internal/repositories"
	"paas-backend/internal/services"

	"github.com/gorilla/mux"
)

type Dependencies struct {
	ECRService services.ECRService
	EKSService services.EKSService
}
///
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

	ecrService := services.NewECRService(dockerService, ecrRepo, eksService)

	return &Dependencies{
		ECRService: ecrService,
		EKSService: eksService,
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
	r.HandleFunc("/build-and-push", controllers.BuildAndPushToECRApiHandler(deps.ECRService, deps.EKSService)).Methods("POST")
	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}).Methods("GET")
}
