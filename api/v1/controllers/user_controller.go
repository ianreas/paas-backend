package controllers

import (
	"encoding/json"
	"net/http"
	"paas-backend/internal/db"
	"paas-backend/internal/models"
    
)

// handler for adding a user to the database
func AddUserHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var user models.User
    err := json.NewDecoder(r.Body).Decode(&user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    err = addUserToDB(&user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func addUserToDB(user *models.User) error {
    query := `INSERT INTO users (username, email) VALUES ($1, $2) RETURNING id`
    err := db.DB.QueryRow(query, user.Username, user.Email).Scan(&user.ID)
    if err != nil {
        return err
    }
    return nil
}