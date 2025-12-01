package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPServer TCP 伺服器
type TCPServer struct {
	port         int
	listener     net.Listener
	stateManager *StateManager
	clients      map[net.Conn]bool
	clientsMu    sync.RWMutex
	stopChan     chan struct{}
}

// NewTCPServer 建立新的 TCP 伺服器
func NewTCPServer(port int, stateManager *StateManager) *TCPServer {
	return &TCPServer{
		port:         port,
		stateManager: stateManager,
		clients:      make(map[net.Conn]bool),
		stopChan:     make(chan struct{}),
	}
}

// Start 啟動 TCP 伺服器
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	s.listener = listener
	log.Printf("[TCP] Server listening on port %d", s.port)

	// 設定 StateManager 的發送函數
	s.stateManager.SetSendToTPTFunc(s.SendToAllClients)

	go s.acceptConnections()
	return nil
}

// acceptConnections 接受新連線
func (s *TCPServer) acceptConnections() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				log.Printf("[TCP] Accept error: %v", err)
				continue
			}
		}

		log.Printf("[TCP] New connection from %s", conn.RemoteAddr())

		s.clientsMu.Lock()
		s.clients[conn] = true
		s.clientsMu.Unlock()

		go s.handleConnection(conn)
	}
}

// handleConnection 處理單一連線
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		log.Printf("[TCP] Connection closed: %s", conn.RemoteAddr())
	}()

	for {
		// 讀取訊息
		jsonData, err := ReadMessage(conn)
		if err != nil {
			log.Printf("[TCP] Read error from %s: %v", conn.RemoteAddr(), err)
			return
		}

		// 記錄收到的訊息
		log.Printf("[TCP] Received from %s: %s", conn.RemoteAddr(), string(jsonData))

		// 處理訊息
		response, err := s.stateManager.HandleMessage(jsonData)
		if err != nil {
			log.Printf("[TCP] Handle message error: %v", err)
			// 即使處理失敗，也不中斷連線
			continue
		}

		// 發送回覆
		if response != nil {
			if err := WriteMessage(conn, response); err != nil {
				log.Printf("[TCP] Write error to %s: %v", conn.RemoteAddr(), err)
				return
			}

			// 記錄發送的訊息
			respJSON, _ := json.Marshal(response)
			log.Printf("[TCP] Sent to %s: %s", conn.RemoteAddr(), string(respJSON))

			// 廣播到前端
			if s.stateManager.broadcastFunc != nil {
				s.stateManager.broadcast(map[string]interface{}{
					"direction": "MES->TPT",
					"data":      response,
				})
			}
		}
	}
}

// SendToAllClients 發送訊息給所有連線的 TPT 客戶端
func (s *TCPServer) SendToAllClients(data interface{}) error {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	if len(s.clients) == 0 {
		return fmt.Errorf("no TPT clients connected")
	}

	var lastErr error
	for conn := range s.clients {
		if err := WriteMessage(conn, data); err != nil {
			log.Printf("[TCP] Failed to send to %s: %v", conn.RemoteAddr(), err)
			lastErr = err
		} else {
			// 記錄發送的訊息
			jsonData, _ := json.Marshal(data)
			log.Printf("[TCP] Sent to %s: %s", conn.RemoteAddr(), string(jsonData))
		}
	}

	return lastErr
}

// Stop 停止 TCP 伺服器
func (s *TCPServer) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}

	// 關閉所有客戶端連線
	s.clientsMu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[net.Conn]bool)
	s.clientsMu.Unlock()

	log.Printf("[TCP] Server stopped")
}

// GetClientCount 取得連線的客戶端數量
func (s *TCPServer) GetClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

