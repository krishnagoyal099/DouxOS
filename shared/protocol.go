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
type MessageTaskAssigned struct {
	Type            string `json:"type"`
	TaskID          int    `json:"task_id"`
	JobID           int    `json:"job_id"`
	DownloadURL     string `json:"download_url"`
	ScriptURL       string `json:"script_url"`
	Instruction     string `json:"instruction"`
	ResultUploadURL string `json:"result_upload_url"` // <--- NEW: URL to send result
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
