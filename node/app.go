package main

import (
	"context"
	"log"
	"sync"

	"github.com/krishnagoyal099/DouxOS/node/connection"
	"github.com/krishnagoyal099/DouxOS/node/executor"
)

// App struct
type App struct {
	ctx     context.Context
	client  *connection.Client
	monitor *executor.Monitor
	mu      sync.Mutex
	status  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.status = "Initializing..."

	// Initialize Monitor
	a.monitor = executor.NewMonitor()

	// Initialize Client
	serverURL := "ws://localhost:8080/ws"
	a.client = connection.NewClient(serverURL, a.monitor, a.updateStatus)

	// Start Connection
	go a.client.Connect()
}

func (a *App) shutdown(ctx context.Context) {
	if a.client != nil {
		a.client.Close()
	}
	if a.monitor != nil {
		a.monitor.Cleanup()
	}
}

// Callback to update status
func (a *App) updateStatus(state string, details string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = state + ": " + details
	log.Println("[UI] " + a.status)
}

// GetStatus is exposed to Frontend
func (a *App) GetStatus() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}
