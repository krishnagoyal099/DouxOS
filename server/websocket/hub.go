package websocket

import (
    "encoding/json"
	"fmt"
    "log"
    "net/http"
    "sync"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "github.com/krishnagoyal099/DouxOS/server/database"
    "github.com/krishnagoyal099/DouxOS/shared"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // Allow all for dev
}

type Client struct {
    Conn   *websocket.Conn
    NodeID string
    mu     sync.Mutex
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }

    client := &Client{Conn: conn, NodeID: uuid.New().String()}
    defer conn.Close()

    // Read loop
    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            break
        }

        var base shared.BaseMessage
        if err := json.Unmarshal(message, &base); err != nil {
            continue
        }

        switch base.Type {
        case "register":
            // We already assigned a UUID, but we confirm here
            log.Println("Node registered:", client.NodeID)
            
        case "request_task":
            client.assignTask()

        case "task_done":
            var msg shared.MessageTaskDone
            json.Unmarshal(message, &msg)
            client.handleTaskDone(msg)
        }
    }
}

func (c *Client) assignTask() {
    // Atomically claim a pending task (prevents two nodes grabbing the same task)
    res, err := database.DB.Exec(
        `UPDATE tasks SET status = 'IN_PROGRESS', assigned_node_id = ?
         WHERE id = (SELECT id FROM tasks WHERE status = 'PENDING' LIMIT 1)`,
        c.NodeID,
    )
    if err != nil {
        return
    }
    rows, _ := res.RowsAffected()
    if rows == 0 {
        return // No tasks available
    }

    // Fetch the task we just claimed
    var taskID int
    var inputPath string
    database.DB.QueryRow(
        "SELECT id, input_file_path FROM tasks WHERE status = 'IN_PROGRESS' AND assigned_node_id = ? ORDER BY id DESC LIMIT 1",
        c.NodeID,
    ).Scan(&taskID, &inputPath)

    // Construct URL (assuming server runs on localhost:8080 for now)
    downloadURL := fmt.Sprintf("http://localhost:8080/storage/%s", inputPath)

    msg := shared.MessageTaskAssigned{
        Type:        "task_assigned",
        TaskID:      taskID,
        DownloadURL: downloadURL,
        ScriptURL:   "http://localhost:8080/storage/script.wasm", // Placeholder
        Instruction: "run_wasm",
    }

    c.mu.Lock()
    c.Conn.WriteJSON(msg)
    c.mu.Unlock()
}

func (c *Client) handleTaskDone(msg shared.MessageTaskDone) {
    // Mark task as DONE
    res, _ := database.DB.Exec("UPDATE tasks SET status = 'DONE' WHERE id = ? AND assigned_node_id = ?", msg.TaskID, c.NodeID)
    if rows, _ := res.RowsAffected(); rows > 0 {
        // Increment job progress
        var jobID int
        database.DB.QueryRow("SELECT job_id FROM tasks WHERE id = ?", msg.TaskID).Scan(&jobID)
        database.DB.Exec("UPDATE jobs SET completed_chunks = completed_chunks + 1 WHERE id = ?", jobID)
        
        // Check if job is complete
        var total, completed int
        database.DB.QueryRow("SELECT total_chunks, completed_chunks FROM jobs WHERE id = ?", jobID).Scan(&total, &completed)
        if total == completed {
            database.DB.Exec("UPDATE jobs SET status = 'COMPLETED' WHERE id = ?", jobID)
            log.Printf("Job %d COMPLETED!", jobID)
        }
    }
}