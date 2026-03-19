package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/krishnagoyal099/DouxOS/shared"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Monitor struct {
	currentJobID int
	runtime      wazero.Runtime
	module       wazero.CompiledModule
	cacheDir     string
}

func NewMonitor() *Monitor {
	dir, _ := os.MkdirTemp("", "douxos_cache")
	return &Monitor{
		currentJobID: -1,
		cacheDir:     dir,
	}
}

func (m *Monitor) ProcessTask(task shared.MessageTaskAssigned) bool {
	// 1. CHECK CONTEXT SWITCH (New Job?)
	if task.JobID != m.currentJobID {
		log.Printf("New Job Detected: %d. Resetting Sandbox...", task.JobID)
		m.resetEnvironment()
		m.currentJobID = task.JobID

		// Download and Compile Script
		if !m.setupScript(task.ScriptURL) {
			log.Println("Failed to setup script")
			return false
		}
	}

	// 2. PREPARE INPUT
	inputPath := filepath.Join(m.cacheDir, "input.txt")
	outputPath := filepath.Join(m.cacheDir, "output.txt")
	os.Remove(inputPath)
	os.Remove(outputPath)

	// Download the chunk
	if err := downloadFile(task.DownloadURL, inputPath); err != nil {
		log.Println("Failed to download chunk:", err)
		return false
	}

	// 3. EXECUTE WASM
	ctx := context.Background()
	fsConfig := wazero.NewFSConfig().WithDirMount(m.cacheDir, "/mnt")

	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithFSConfig(fsConfig).
		WithArgs("wasm", "/mnt/input.txt", "/mnt/output.txt") // Pass paths

	instance, err := m.runtime.InstantiateModule(ctx, m.module, config)
	if err != nil {
		log.Println("Execution Failed:", err)
		return false
	}
	instance.Close(ctx)

	log.Println("Execution Complete.")

	// 4. UPLOAD RESULT
	// NOTE: This block must be BEFORE the final 'return true'
	if task.ResultUploadURL == "" {
		log.Println("ERROR: No ResultUploadURL provided. Cannot save result.")
		return false
	}

	log.Println("Uploading result to:", task.ResultUploadURL)
	err = uploadFile(task.ResultUploadURL, outputPath)
	if err != nil {
		log.Println("Failed to upload result:", err)
		return false
	}

	log.Println("Result uploaded successfully.")
	return true
}

func (m *Monitor) resetEnvironment() {
	if m.runtime != nil {
		m.runtime.Close(context.Background())
	}
	ctx := context.Background()
	m.runtime = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, m.runtime)
}

func (m *Monitor) setupScript(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	binary, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	m.module, err = m.runtime.CompileModule(context.Background(), binary)
	if err != nil {
		log.Println("Failed to compile WASM:", err)
		return false
	}
	return true
}

func (m *Monitor) Cleanup() {
	if m.runtime != nil {
		m.runtime.Close(context.Background())
	}
	os.RemoveAll(m.cacheDir)
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// uploadFile sends the result file back to the server
func uploadFile(url, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Create a multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "result.txt")
	if err != nil {
		return err
	}
	fw.Write(body)
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	return nil
}
