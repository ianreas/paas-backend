// controllers/monitoring_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"paas-backend/internal/services"
	"time"
)

type MetricsRequest struct {
	ClusterName string `json:"clusterName"`
	Namespace   string `json:"namespace"`
	AppName     string `json:"appName"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

func GetMetricsHandler(monitoringService services.MonitoringService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MetricsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		fmt.Printf("Received metrics request: %+v\n", req)

		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid start time format: %v", err), http.StatusBadRequest)
			return
		}

		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid end time format: %v", err), http.StatusBadRequest)
			return
		}

		metrics, err := monitoringService.GetMetrics(r.Context(), req.ClusterName, req.Namespace, req.AppName, startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching metrics: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
		}
	}
}