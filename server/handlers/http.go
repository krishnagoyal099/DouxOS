package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/krishnagoyal099/DouxOS/server/database"
	"github.com/krishnagoyal099/DouxOS/server/splitter"
	"github.com/krishnagoyal099/DouxOS/shared"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create Job in DB
	res, err := database.DB.Exec("INSERT INTO jobs (status, total_chunks) VALUES (?, ?)", "PROCESSING", 0)
	if err != nil {
		log.Println("Database Insert Error:", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	jobID, _ := res.LastInsertId()

	// Create per-job directory
	jobDir := filepath.Join("./storage/jobs", fmt.Sprintf("%d", jobID))
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		http.Error(w, "Failed to create job directory", http.StatusInternalServerError)
		return
	}

	// Save original file to storage/jobs/{id}/original.txt
	originalPath := filepath.Join(jobDir, "original.txt")
	dst, err := os.Create(originalPath)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	io.Copy(dst, file)
	dst.Close()

	// Handle script: save to storage/jobs/{id}/script.wasm
	scriptDst := filepath.Join(jobDir, "script.wasm")
	scriptFile, _, scriptErr := r.FormFile("script")
	if scriptErr == nil {
		// Script was uploaded — save it
		defer scriptFile.Close()
		sf, err := os.Create(scriptDst)
		if err != nil {
			http.Error(w, "Failed to save script", http.StatusInternalServerError)
			return
		}
		io.Copy(sf, scriptFile)
		sf.Close()
	} else {
		// No script uploaded — copy default fallback
		defaultScript := "./storage/script.wasm"
		if data, err := os.ReadFile(defaultScript); err == nil {
			os.WriteFile(scriptDst, data, 0644)
		} else {
			log.Println("Warning: no script uploaded and no default script.wasm found")
		}
	}

	// Split the file
	go func() {
		if err := splitter.SplitAndSave(int(jobID), originalPath, "original.txt"); err != nil {
			fmt.Println("Splitting error:", err)
			database.DB.Exec("UPDATE jobs SET status = 'FAILED' WHERE id = ?", jobID)
		}
	}()

	// Return Job ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shared.UploadResponse{JobID: int(jobID)})
}

func ResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// 1. Get the file from the Node
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Find Job ID and create result directory
	var jobID int
	database.DB.QueryRow("SELECT job_id FROM tasks WHERE id = ?", taskID).Scan(&jobID)

	resDir := filepath.Join("./storage/jobs", fmt.Sprintf("%d", jobID), "results")
	os.MkdirAll(resDir, 0755)

	// 3. Save the result file
	// Naming it result_{taskID}.txt ensures we can sort them later for merging
	resPath := filepath.Join(resDir, fmt.Sprintf("result_%s.txt", taskID))
	dst, _ := os.Create(resPath)
	io.Copy(dst, file)
	dst.Close()

	w.Write([]byte("OK"))
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID, _ := strconv.Atoi(vars["id"])

	var status string
	var total, completed int
	row := database.DB.QueryRow("SELECT status, total_chunks, completed_chunks FROM jobs WHERE id = ?", jobID)
	if err := row.Scan(&status, &total, &completed); err != nil {
		http.Error(w, "Job not found", 404)
		return
	}

	var progress float64
	if total > 0 {
		progress = (float64(completed) / float64(total)) * 100
	}

	json.NewEncoder(w).Encode(shared.StatusResponse{
		Status:          status,
		Progress:        progress,
		TotalChunks:     total,
		CompletedChunks: completed,
	})
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	// 1. Check Status
	var status string
	err := database.DB.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&status)
	if err != nil || status != "COMPLETED" {
		// Show a JSON error if not done
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf(`{"error": "Job not complete or not found", "status": "%s"}`, status)))
		return
	}

	// 2. Locate the file
	filePath := filepath.Join("./storage/jobs", jobID, "final_output.txt")

	// 3. Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Result file not found on server", http.StatusNotFound)
		return
	}

	// 4. Display as text in browser
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}
