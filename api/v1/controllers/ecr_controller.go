package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"paas-backend/internal/repositories"
	"paas-backend/internal/services"
)

type BuildAndPushRequest struct {
	RepoFullName   string `json:"repoFullName"`
	AccessToken    string `json:"accessToken"`
	UserId         string `json:"userId"`
	GithubUsername string `json:"githubUsername"`
	ContainerPort   int32  `json:"containerPort,omitempty"`
	Replicas        *int32  `json:"replicas,omitempty"`
    CPU             *string `json:"cpuAllocation,omitempty"`
    Memory          *string `json:"memoryAllocation,omitempty"`
}

// BuildPushDeployApiHandler handles the build, push, deploy, and records the application in the database.
func BuildPushDeployApiHandler(
	ecrService services.ECRService,
	eksService services.EKSService,
	appsRepo repositories.ApplicationsRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BuildAndPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Received request to build and deploy app: %s by user: %s", req.RepoFullName, req.UserId)

		// Build and push the image to ECR
		ecrImageName, err := ecrService.BuildAndPushToECR(r.Context(), req.RepoFullName, req.AccessToken)
		if err != nil {
			log.Printf("Error building and pushing to ECR for repo %s: %v", req.RepoFullName, err)
			http.Error(w, fmt.Sprintf("Error building and pushing to ECR: %v", err), http.StatusInternalServerError)
			return
		}

		appName := filepath.Base(req.RepoFullName)

		// Deploy the image to EKS
		err = eksService.DeployToEKS(r.Context(), ecrImageName, appName, req.UserId, req.ContainerPort, req.Replicas, req.CPU, req.Memory)
		if err != nil {
			log.Printf("Error deploying to EKS for app %s: %v", appName, err)
			http.Error(w, fmt.Sprintf("Error deploying to EKS: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully deployed app %s to EKS", appName)

		// Create a new application record
		app := &repositories.Application{
			GithubRepoName: req.RepoFullName,
			GithubUsername: req.GithubUsername,
			UserID:         req.UserId,
			ProjectName:    appName,
			ContainerPort:  req.ContainerPort,
			Replicas:       req.Replicas,
			CPU:            req.CPU,
			Memory:         req.Memory,
		}
		appID, err := appsRepo.CreateOrUpdateApplication(r.Context(), app)
		if err != nil {
			log.Printf("Error creating application record for app %s: %v", appName, err)
			http.Error(w, fmt.Sprintf("Error creating application record: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Application record created with ID %d for app %s", appID, appName)

		// Return the application ID in the response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":        "Image built, pushed to ECR, deployed to EKS, and recorded in the database successfully",
			"ecrImageName":   ecrImageName,
			"appName":        appName,
			"application_id": appID,
		})
	}
}

