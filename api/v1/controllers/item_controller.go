package controllers

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
    "paas-backend/internal/models"
    "paas-backend/internal/store"
)

func GetItems(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(store.Items)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    for _, item := range store.Items {
        if item.ID == params["id"] {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(item)
            return
        }
    }
    http.Error(w, "Item not found", http.StatusNotFound)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
    var item models.Item
    json.NewDecoder(r.Body).Decode(&item)
    store.Items = append(store.Items, item)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(item)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    for i, item := range store.Items {
        if item.ID == params["id"] {
            json.NewDecoder(r.Body).Decode(&item)
            store.Items[i] = item
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(item)
            return
        }
    }
    http.Error(w, "Item not found", http.StatusNotFound)
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    for i, item := range store.Items {
        if item.ID == params["id"] {
            store.Items = append(store.Items[:i], store.Items[i+1:]...)
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(store.Items)
            return
        }
    }
    http.Error(w, "Item not found", http.StatusNotFound)
}
