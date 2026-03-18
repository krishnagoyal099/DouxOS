package shared

// BaseMessage is used to detect the type of incoming JSON
type BaseMessage struct {
    Type string `json:"type"`
}

// MessageRegister (Node -> Server)
// Sent immediately upon connection
type MessageRegister struct {
    Type            string `json:"type"`
    NodeID          string `json:"node_id"`
    ConfidenceScore int    `json:"confidence_score"`
}

// MessageRequestTask (Node -> Server)
// Sent periodically to ask for work
type MessageRequestTask struct {
    Type string `json:"type"`
}

// MessageTaskAssigned (Server -> Node)
// Sent when work is available
type MessageTaskAssigned struct {
    Type        string `json:"type"`
    TaskID      int    `json:"task_id"`
    DownloadURL string `json:"download_url"` // URL to the chunk file
    ScriptURL   string `json:"script_url"`   // URL to the WASM binary
    Instruction string `json:"instruction"`  // e.g., "run_wasm"
}

// MessageTaskDone (Node -> Server)
// Sent after successful execution
type MessageTaskDone struct {
    Type   string `json:"type"`
    TaskID int    `json:"task_id"`
}

// API Response Structs

// UploadResponse is returned by the Server's /api/upload
type UploadResponse struct {
    JobID int `json:"job_id"`
}

// StatusResponse is returned by the Server's /api/status
type StatusResponse struct {
    Status          string  `json:"status"`
    Progress        float64 `json:"progress"`
    TotalChunks     int     `json:"total_chunks"`
    CompletedChunks int     `json:"completed_chunks"`
}