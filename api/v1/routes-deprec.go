package v1

// import (
// 	"net/http"
// 	"paas-backend/api/v1/controllers"

// 	"github.com/gorilla/mux"

// 	"paas-backend/internal/services"

// 	"paas-backend/internal/repositories"

// 	"context"

// 	"log"
// )


// // description: a variable to hold the pointer to the struct ECRController 
// // it holds a pointer to an instance of the ECRController

// // purpose: its declared in the global scope so can be access by any function in the same package
// // this allows the controller to be intialized once and then reused across multiple https requests (which i think is dangerous)

// // behavior: as a pointer, it stores the memory address of an ECRController instance
// // when me hods are called on ecrController variable, they are executed on the actual instance it points to

// // usage: it's used to setup the http route in the RegisterRoutes function. this associates the
// // BuildAndPushToECRApiHandler method of the ecrController instance with the /build-and-push route

// // advantages: singleton like behavior, ensures only 1 instance of ECRController is created and used throughout the app

// // persistence: the controller and its state are preserved across http requests 

// // concurrency: this variable is shared across multiple requests, so we need to ensure thread safety

// // In summary, ecrController serves as a global reference to a single instance of ECRController,
// // allowing for centralized management of ECR (Elastic Container Registry) related operations across the API routes.
// var ecrController *controllers.ECRController

// // 1) in init(), the ecrService and eksService are created and then passed to ecrController
// func init() {
// 	ctx := context.Background()

// 	// Create the necessary services
// 	dockerService := services.NewDockerService()
// 	ecrRepo, err := repositories.NewECRRepository(ctx)
// 	if err != nil {
// 		log.Fatalf("Failed to create ECR repository: %v", err)
// 	}

// 	eksService, err := services.NewEKSService(ctx)
// 	if err != nil {
// 		log.Fatalf("Failed to create EKS service: %v", err)
// 	}

// 	ecrService := services.NewECRService(dockerService, ecrRepo, eksService)

// 	// Pass both ecrService and eksService to NewECRController which then creates an instance of the ECRController struct
// 	ecrController = controllers.NewECRController(ecrService, eksService)
// }

// func RegisterRoutes(r *mux.Router) {
// 	r.HandleFunc("/items", controllers.GetItems).Methods("GET")
// 	r.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
// 	r.HandleFunc("/items", controllers.CreateItem).Methods("POST")
// 	r.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
// 	r.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
// 	r.HandleFunc("/users", controllers.AddUserHandler).Methods("POST")
// 	r.HandleFunc("/ec2", controllers.CreateEC2InstanceHandler).Methods("POST")
// 	r.HandleFunc("/rds", controllers.CreateRDSInstanceHandler).Methods("POST")
// 	r.HandleFunc("/deploy", controllers.DeployHandler).Methods("POST")
// 	r.HandleFunc("/build-and-push", ecrController.BuildAndPushToECRApiHandler).Methods("POST")
// 	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte("Hello, World!"))
// 	}).Methods("GET")
// }


