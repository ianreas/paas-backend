package v1

import (
	"paas-backend/api/v1/controllers"
    "net/http"
	"github.com/gorilla/mux"

    "paas-backend/internal/services"

	"paas-backend/internal/repositories"

	"context"
	
	"log"
)

var ecrController *controllers.ECRController

func init() {
    ctx := context.Background()

    // Create the necessary services
    dockerService := services.NewDockerService()
    ecrRepo, err := repositories.NewECRRepository(ctx)
    if err != nil {
        log.Fatalf("Failed to create ECR repository: %v", err)
    }
    
    eksService, err := services.NewEKSService(ctx)
    if err != nil {
        log.Fatalf("Failed to create EKS service: %v", err)
    }
    
    ecrService := services.NewECRService(dockerService, ecrRepo, eksService)
    
    // Pass both ecrService and eksService to NewECRController
    ecrController = controllers.NewECRController(ecrService, eksService)
}


func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/items", controllers.GetItems).Methods("GET")
	r.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
	r.HandleFunc("/items", controllers.CreateItem).Methods("POST")
	r.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
	r.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
	r.HandleFunc("/users", controllers.AddUserHandler).Methods("POST")
	r.HandleFunc("/ec2", controllers.CreateEC2InstanceHandler).Methods("POST")
	r.HandleFunc("/rds", controllers.CreateRDSInstanceHandler).Methods("POST")
	r.HandleFunc("/deploy", controllers.DeployHandler).Methods("POST")
	r.HandleFunc("/build-and-push", ecrController.BuildAndPushToECR).Methods("POST")
    r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    }).Methods("GET")
}
