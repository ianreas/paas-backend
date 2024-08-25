package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"paas-backend/internal/services"
	"path/filepath"
)

type BuildAndPushRequest struct {
	RepoFullName string `json:"repoFullName"`
	AccessToken  string `json:"accessToken"`
}

func BuildPushDeployApiHandler(ecrService services.ECRService, eksService services.EKSService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BuildAndPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ecrImageName, err := ecrService.BuildAndPushToECR(r.Context(), req.RepoFullName, req.AccessToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error building and pushing to ECR: %v", err), http.StatusInternalServerError)
			return
		}

		appName := filepath.Base(req.RepoFullName)

		err = eksService.DeployToEKS(r.Context(), ecrImageName, appName, 3000)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error deploying to EKS: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":      "Image built, pushed to ECR, and deployed to EKS successfully",
			"ecrImageName": ecrImageName,
			"appName":      appName,
		})
	}
}
