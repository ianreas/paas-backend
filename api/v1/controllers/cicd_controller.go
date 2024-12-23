package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"paas-backend/internal/services"
)

func CreateWorkflowHandler(githubService services.GitHubService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req services.WorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Received CreateWorkflow request for %s/%s", req.RepoOwner, req.RepoName)

		// Use the GitHubService to create the workflow
		response, err := githubService.CreateWorkflow(ctx, req)
		if err != nil {
			log.Printf("Error in CreateWorkflow: %v", err)
			http.Error(w, fmt.Sprintf("Error creating workflow: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Workflow creation successful: PR #%d at %s", response.PRNumber, response.PRURL)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}
}

