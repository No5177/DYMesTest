package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"io/fs"
	"github.com/gorilla/websocket"
)

// HTTPServer HTTP 與 WebSocket 伺服器
type HTTPServer struct {
	port         int
	stateManager *StateManager
	tcpServer    *TCPServer
	staticFS     fs.FS 
	upgrader     websocket.Upgrader
	wsClients    map[*websocket.Conn]bool
	wsClientsMu  sync.RWMutex
	broadcast    chan interface{}
}

// NewHTTPServer 建立新的 HTTP 伺服器
func NewHTTPServer(port int, stateManager *StateManager, tcpServer *TCPServer, staticFS fs.FS) *HTTPServer {
	server := &HTTPServer{
		port:         port,
		stateManager: stateManager,
		tcpServer:    tcpServer,
		staticFS:     staticFS, // 儲存傳入的檔案系統
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		wsClients: make(map[*websocket.Conn]bool),
		broadcast: make(chan interface{}, 100),
	}

	stateManager.SetBroadcastFunc(server.BroadcastToWebSocket)
	go server.handleBroadcast()

	return server
}

// Start 啟動 HTTP 伺服器
func (s *HTTPServer) Start() error {
	// 3. 修改：直接使用傳入的 staticFS
	// 這裡不需要 fs.Sub，因為 main.go 已經處理好了
	http.Handle("/", http.FileServer(http.FS(s.staticFS)))

	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/api/status", s.handleGetStatus)
	http.HandleFunc("/api/channels", s.handleGetChannels)
	http.HandleFunc("/api/cmd/start", s.handleStartCommand)
	http.HandleFunc("/api/cmd/stop", s.handleStopCommand)
	http.HandleFunc("/api/cmd/pause", s.handlePauseCommand)
	http.HandleFunc("/api/cmd/resume", s.handleResumeCommand)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[HTTP] Server starting on http://localhost%s", addr)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			// 使用 Printf 避免 Port 佔用時直接閃退
			log.Printf("[HTTP] ❌ Server Start Error (Check if port %d is used): %v", s.port, err)
		}
	}()

	return nil
}

// handleWebSocket 處理 WebSocket 連線
func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}

	log.Printf("[WS] New WebSocket connection from %s", conn.RemoteAddr())

	s.wsClientsMu.Lock()
	s.wsClients[conn] = true
	s.wsClientsMu.Unlock()

	// 發送當前狀態給新連線的客戶端
	s.sendCurrentState(conn)

	// 處理客戶端訊息（如果需要）
	go func() {
		defer func() {
			s.wsClientsMu.Lock()
			delete(s.wsClients, conn)
			s.wsClientsMu.Unlock()
			conn.Close()
			log.Printf("[WS] Connection closed: %s", conn.RemoteAddr())
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// sendCurrentState 發送當前狀態給 WebSocket 客戶端
func (s *HTTPServer) sendCurrentState(conn *websocket.Conn) {
	status := s.stateManager.GetConnectionStatus()
	channels := s.stateManager.GetAllChannels()

	data := map[string]interface{}{
		"type":     "initial_state",
		"status":   status,
		"channels": channels,
	}

	if err := conn.WriteJSON(data); err != nil {
		log.Printf("[WS] Failed to send initial state: %v", err)
	}
}

// BroadcastToWebSocket 廣播訊息到所有 WebSocket 客戶端
func (s *HTTPServer) BroadcastToWebSocket(data interface{}) {
	select {
	case s.broadcast <- data:
	default:
		log.Printf("[WS] Broadcast channel full, dropping message")
	}
}

// handleBroadcast 處理廣播訊息
func (s *HTTPServer) handleBroadcast() {
	for data := range s.broadcast {
		s.wsClientsMu.RLock()
		for conn := range s.wsClients {
			if err := conn.WriteJSON(data); err != nil {
				log.Printf("[WS] Broadcast error: %v", err)
			}
		}
		s.wsClientsMu.RUnlock()
	}
}

// handleGetStatus 取得連線狀態
func (s *HTTPServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.stateManager.GetConnectionStatus()
	
	// TCP 連線狀態（純粹的 socket 連接）
	tcpClientCount := s.tcpServer.GetClientCount()
	status["tcp_connected"] = tcpClientCount > 0
	status["tcp_clients"] = tcpClientCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleGetChannels 取得所有通道狀態
func (s *HTTPServer) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels := s.stateManager.GetAllChannels()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// CommandRequest 命令請求結構
type CommandRequest struct {
	Channel  string `json:"channel"`
	Barcode  string `json:"barcode,omitempty"`
	Process  string `json:"process,omitempty"`
	DataPath string `json:"data_path,omitempty"`
}

// handleStartCommand 處理 START 命令
func (s *HTTPServer) handleStartCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 驗證必要欄位
	if req.Channel == "" || req.Barcode == "" || req.Process == "" || req.DataPath == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// 執行 START 命令
	err := s.stateManager.ValidateAndSendStart(req.Channel, req.Barcode, req.Process, req.DataPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleStopCommand 處理 STOP 命令
func (s *HTTPServer) handleStopCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		http.Error(w, "Missing channel field", http.StatusBadRequest)
		return
	}

	err := s.stateManager.ValidateAndSendStop(req.Channel)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handlePauseCommand 處理 PAUSE 命令
func (s *HTTPServer) handlePauseCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		http.Error(w, "Missing channel field", http.StatusBadRequest)
		return
	}

	err := s.stateManager.ValidateAndSendPause(req.Channel)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleResumeCommand 處理 RESUME 命令
func (s *HTTPServer) handleResumeCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		http.Error(w, "Missing channel field", http.StatusBadRequest)
		return
	}

	err := s.stateManager.ValidateAndSendResume(req.Channel)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

