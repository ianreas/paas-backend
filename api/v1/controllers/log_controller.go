package controllers

import (
    "encoding/json"
    "net/http"
    "paas-backend/internal/services"
    "time"

    "github.com/gorilla/mux"
)

func StreamLogsHandler(logService services.LogService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        appName := mux.Vars(r)["appName"]
        startTime := time.Now().Add(-24 * time.Hour) // Get logs from the last 24 hours

        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        w.Header().Set("Access-Control-Allow-Origin", "*")

        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
            return
        }

        logChan, errChan := logService.StreamLogs(r.Context(), appName, startTime)

        for {
            select {
            case log, ok := <-logChan:
                if !ok {
                    return
                }
                event := struct {
                    Event string `json:"event"`
                    Data  string `json:"data"`
                }{
                    Event: "log",
                    Data:  log,
                }
                if err := json.NewEncoder(w).Encode(event); err != nil {
                    return
                }
                flusher.Flush()
            case err, ok := <-errChan:
                if !ok {
                    return
                }
                event := struct {
                    Event string `json:"event"`
                    Data  string `json:"data"`
                }{
                    Event: "error",
                    Data:  err.Error(),
                }
                if err := json.NewEncoder(w).Encode(event); err != nil {
                    return
                }
                flusher.Flush()
                return
            case <-r.Context().Done():
                return
            }
        }
    }
}