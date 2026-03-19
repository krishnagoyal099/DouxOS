package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/krishnagoyal099/DouxOS/server/database"
	"github.com/krishnagoyal099/DouxOS/server/handlers"
	"github.com/krishnagoyal099/DouxOS/server/websocket"
)

func main() {
	// Initialize Database
	if err := database.InitDB("douxos.db"); err != nil {
		log.Fatal("DB Init failed:", err)
	}

	r := mux.NewRouter()

	// API Routes
	r.HandleFunc("/api/upload", handlers.UploadHandler).Methods("POST")
	r.HandleFunc("/api/status/{id}", handlers.StatusHandler).Methods("GET")
	r.HandleFunc("/api/download/{id}", handlers.DownloadHandler).Methods("GET")
	r.HandleFunc("/api/result/{id}", handlers.ResultHandler).Methods("POST")

	// WebSocket Route
	r.HandleFunc("/ws", websocket.HandleWebSocket)

	// Static file server for chunks (needed for Node to download files)
	r.PathPrefix("/storage/").Handler(http.StripPrefix("/storage/", http.FileServer(http.Dir("./storage"))))

	// Serve Frontend at root
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend")))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
