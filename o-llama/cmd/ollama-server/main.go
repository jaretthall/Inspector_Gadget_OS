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
	Service   string    `json:"service"`
}

type FileReadRequest struct {
	Path string `json:"path"`
}

type FileReadResponse struct {
	Message   string    `json:"message"`
	Path      string    `json:"path,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Service:   "o-llama-enhanced",
	}
	
	json.NewEncoder(w).Encode(response)
}

func fileReadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req FileReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	response := FileReadResponse{
		Message:   "stub - file system access not implemented yet",
		Path:      req.Path,
		Timestamp: time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/fs/read", fileReadHandler)
	
	port := "11434"
	fmt.Printf("O-LLaMA Enhanced server starting on port %s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/health -> Health check")
	fmt.Println("  POST /api/fs/read -> File system read (stub)")
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}