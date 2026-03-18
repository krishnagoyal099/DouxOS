package connection

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
	"github.com/krishnagoyal099/DouxOS/node/executor"
	"github.com/krishnagoyal099/DouxOS/shared"
)

type Client struct {
	conn     *websocket.Conn
	url      string
	monitor  *executor.Monitor
	uiCb     func(string, string)
	state    string
	stopChan chan struct{}
}

func NewClient(serverURL string, monitor *executor.Monitor, uiCb func(string, string)) *Client {
	return &Client{
		url:      serverURL,
		monitor:  monitor,
		uiCb:     uiCb,
		state:    "IDLE",
		stopChan: make(chan struct{}),
	}
}

func (c *Client) Connect() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if c.conn != nil {
				continue
			}

			c.uiCb("CONNECTING", "Connecting to Grid...")
			conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
			if err != nil {
				c.uiCb("OFFLINE", "Server Unreachable")
				continue
			}

			c.conn = conn
			c.uiCb("ONLINE", "Connected to Grid")
			go c.readLoop()
		}
	}
}

func (c *Client) readLoop() {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.uiCb("DISCONNECTED", "Connection Lost")
	}()

	// 1. Register
	reg := shared.MessageRegister{
		Type:            "register",
		NodeID:          randomString(8),
		ConfidenceScore: 90,
	}
	c.conn.WriteJSON(reg)

	// 2. Start Request Loop
	go c.requestLoop()

	// 3. Read Messages
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var base shared.BaseMessage
		json.Unmarshal(message, &base)

		if base.Type == "task_assigned" {
			var task shared.MessageTaskAssigned
			json.Unmarshal(message, &task)
			c.handleTask(task)
		}
	}
}

func (c *Client) requestLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if c.conn == nil {
				return
			}
			// Only request if we are IDLE
			if c.state == "IDLE" {
				msg := shared.MessageRequestTask{Type: "request_task"}
				c.conn.WriteJSON(msg)
			}
		}
	}
}

func (c *Client) handleTask(task shared.MessageTaskAssigned) {
	c.state = "WORKING"
	c.uiCb("WORKING", "Processing Task #"+string(rune(task.TaskID)))

	// Execute via Monitor (Sandbox)
	success := c.monitor.ProcessTask(task)

	if success {
		c.uiCb("SUCCESS", "Task Completed")
		doneMsg := shared.MessageTaskDone{
			Type:   "task_done",
			TaskID: task.TaskID,
		}
		if c.conn != nil {
			c.conn.WriteJSON(doneMsg)
		}
		// Brief pause before going back to IDLE
		time.Sleep(1 * time.Second)
		c.state = "IDLE"
	} else {
		c.uiCb("ERROR", "Task Failed")
		c.state = "IDLE"
	}
}

func (c *Client) Close() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) GetState() string {
	return c.state
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
