package v1

import (
    "github.com/gorilla/mux"
    "paas-backend/api/v1/controllers"
)

func RegisterRoutes(r *mux.Router) {
    r.HandleFunc("/items", controllers.GetItems).Methods("GET")
    r.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
    r.HandleFunc("/items", controllers.CreateItem).Methods("POST")
    r.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
    r.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
}