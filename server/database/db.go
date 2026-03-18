package database

import (
    "database/sql"
    "fmt"
    "os"

    _ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dataSourceName string) error {
    var err error
    // Ensure the file directory exists
    dir := "./storage"
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        os.Mkdir(dir, 0755)
    }

    DB, err = sql.Open("sqlite", dataSourceName)
    if err != nil {
        return err
    }

    // Create Schema
    schema := `
    CREATE TABLE IF NOT EXISTS jobs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        status TEXT,
        total_chunks INTEGER,
        completed_chunks INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        job_id INTEGER,
        status TEXT,
        assigned_node_id TEXT,
        input_file_path TEXT,
        output_file_path TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(job_id) REFERENCES jobs(id)
    );
    `

    _, err = DB.Exec(schema)
    if err != nil {
        return fmt.Errorf("failed to create schema: %w", err)
    }

    return nil
}