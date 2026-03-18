package splitter

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"

    "github.com/krishnagoyal099/DouxOS/server/database"
)

const LinesPerChunk = 100

func SplitAndSave(jobID int, filePath string, originalFileName string) error {
    // Create directory for this job
    jobDir := fmt.Sprintf("./storage/jobs/%d", jobID)
    chunkDir := filepath.Join(jobDir, "chunks")
    if err := os.MkdirAll(chunkDir, 0755); err != nil {
        return err
    }

    // Read original file
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    chunkIndex := 0
    lineCount := 0
    var currentChunk *os.File

    // Helper to start a new chunk
    startNewChunk := func() error {
        chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d.txt", chunkIndex))
        f, err := os.Create(chunkPath)
        if err != nil {
            return err
        }
        currentChunk = f
        
        // Insert task into DB
        _, err = database.DB.Exec(
            "INSERT INTO tasks (job_id, status, input_file_path) VALUES (?, ?, ?)",
            jobID, "PENDING", chunkPath,
        )
        return err
    }

    // Start first chunk
    if err := startNewChunk(); err != nil {
        return err
    }

    for scanner.Scan() {
        if lineCount >= LinesPerChunk {
            // Rotate to next chunk
            currentChunk.Close()
            chunkIndex++
            lineCount = 0
            if err := startNewChunk(); err != nil {
                return err
            }
        }

        line := scanner.Text()
        currentChunk.WriteString(line + "\n")
        lineCount++
    }
    currentChunk.Close()

    // Update job with total chunks
    _, err = database.DB.Exec("UPDATE jobs SET total_chunks = ? WHERE id = ?", chunkIndex+1, jobID)
    return err
}