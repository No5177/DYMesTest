package main

import (
	"GoTestMES/core"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
)

//go:embed static
var staticEmbed embed.FS

const (
	DefaultTCPPort      = 50200
	DefaultHTTPPort     = 5179
	DefaultChannelCount = 128
)

func main() {
	tcpPort := flag.Int("tcp-port", DefaultTCPPort, "TCP server port")
	httpPort := flag.Int("http-port", DefaultHTTPPort, "HTTP server port")
	channelCount := flag.Int("channels", DefaultChannelCount, "Number of channels")
	flag.Parse()

	printBanner()
	log.Printf("Starting TPT MES Test Server...")

	staticFS, err := fs.Sub(staticEmbed, "static")
	if err != nil {
		log.Fatalf("Failed to load static files: %v", err)
	}

	stateManager := core.NewStateManager(*channelCount)
	tcpServer := core.NewTCPServer(*tcpPort, stateManager)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}

	httpServer := core.NewHTTPServer(*httpPort, stateManager, tcpServer, staticFS)
	if err := httpServer.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	log.Printf("✓ Server started successfully!")
	log.Printf("✓ Web UI: http://localhost:%d", *httpPort)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("\nShutting down server...")
	tcpServer.Stop()
}

// printBanner 顯示啟動橫幅
func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║     TPT_DYMesTest                                         ║
║     TPT Automated Testing MES Server                      ║
║                                                           ║
║     Version: 1.0.3                                        ║
║     Protocol: TCP/IP + JSON with \r\n terminator          ║
║     Purpose: TPT ThinkLab Communication Testing           ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
