package main

import (
	"GoTestMES/core"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	// 預設設定
	DefaultTCPPort     = 50200 // TCP 伺服器埠號
	DefaultHTTPPort    = 8080  // HTTP/WebSocket 伺服器埠號
	DefaultChannelCount = 128   // 預設通道數量
)

func main() {
	// 命令列參數
	tcpPort := flag.Int("tcp-port", DefaultTCPPort, "TCP server port")
	httpPort := flag.Int("http-port", DefaultHTTPPort, "HTTP server port")
	channelCount := flag.Int("channels", DefaultChannelCount, "Number of channels")
	flag.Parse()

	// 顯示啟動資訊
	printBanner()
	log.Printf("Starting TPT MES Test Server...")
	log.Printf("TCP Port: %d", *tcpPort)
	log.Printf("HTTP Port: %d", *httpPort)
	log.Printf("Channel Count: %d", *channelCount)
	log.Printf("Web UI: http://localhost:%d", *httpPort)

	// 建立狀態管理器
	stateManager := core.NewStateManager(*channelCount)
	log.Printf("[StateManager] Initialized with %d channels", *channelCount)

	// 建立並啟動 TCP 伺服器
	tcpServer := core.NewTCPServer(*tcpPort, stateManager)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}

	// 建立並啟動 HTTP 伺服器
	httpServer := core.NewHTTPServer(*httpPort, stateManager, tcpServer)
	if err := httpServer.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	log.Printf("✓ Server started successfully!")
	log.Printf("✓ Waiting for TPT connection on port %d...", *tcpPort)
	log.Printf("✓ Open web browser: http://localhost:%d", *httpPort)

	// 等待中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Printf("\nShutting down server...")

	// 清理資源
	tcpServer.Stop()
	log.Printf("Server stopped. Goodbye!")
}

// printBanner 顯示啟動橫幅
func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║     TPT MES 測試伺服器 (GoTestMES)                       ║
║     TPT Automated Testing MES Server                      ║
║                                                           ║
║     Version: 1.0.0                                        ║
║     Protocol: TCP/IP + JSON (8-byte header)               ║
║     Purpose: TPT ThinkLab Communication Testing           ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

