package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type DashboardData struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "0.1.0-phase0",
	}
	
	json.NewEncoder(w).Encode(response)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	data := DashboardData{
		Title:     "Inspector Gadget OS - Web Management",
		Message:   "Phase 0: Basic HTTP server running",
		Timestamp: time.Now(),
	}
	
	json.NewEncoder(w).Encode(data)
}

func main() {
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/healthz", healthHandler)
	
	port := "8080"
	fmt.Printf("Inspector Gadget OS Web Management starting on port %s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  GET / -> Dashboard")
	fmt.Println("  GET /healthz -> Health check")
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}