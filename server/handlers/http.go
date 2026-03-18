package handlers

import (
    "encoding/json"
    "fmt"
    "io"
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
    // Parse multipart form
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "File too large or invalid form", http.StatusBadRequest)
        return
    }

    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error retrieving file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Create Job in DB
    res, err := database.DB.Exec("INSERT INTO jobs (status, total_chunks) VALUES (?, ?)", "PROCESSING", 0)
    if err != nil {
        http.Error(w, "DB Error", http.StatusInternalServerError)
        return
    }
    jobID, _ := res.LastInsertId()

    // Save original file temporarily
    tempPath := filepath.Join("./storage", handler.Filename)
    dst, err := os.Create(tempPath)
    if err != nil {
        http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
        return
    }
    io.Copy(dst, file)
    dst.Close()

    // Split the file
    go func() {
        defer os.Remove(tempPath)
        if err := splitter.SplitAndSave(int(jobID), tempPath, handler.Filename); err != nil {
            fmt.Println("Splitting error:", err)
            database.DB.Exec("UPDATE jobs SET status = 'FAILED' WHERE id = ?", jobID)
        }
    }()

    // Return Job ID
    json.NewEncoder(w).Encode(shared.UploadResponse{JobID: int(jobID)})
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
    // In a real scenario, you would assemble the outputs here.
    // For now, we just check if completed.
    vars := mux.Vars(r)
    jobID := vars["id"]
    
    var status string
    database.DB.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&status)

    if status != "COMPLETED" {
        http.Error(w, "Job not complete yet", http.StatusForbidden)
        return
    }

    // Serve a dummy result file for now
    w.Write([]byte("Final Result Content... (Assembly logic goes here)"))
}