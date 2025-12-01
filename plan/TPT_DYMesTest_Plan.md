Project Plan: TPT 自動化測試用 MES 伺服器 (Go + Native Web GUI)
1. 專案目標與概述
建立一個模擬 MES (Manufacturing Execution System) 的伺服器應用程式，用於驗證 TPT ThinkLab 客戶端的連線與通訊流程。
此模擬器需具備 Level 3 的邏輯深度，意味著它不僅是單純的 Echo Server，還必須維護通道狀態、驗證命令邏輯，並提供 GUI 介面供測試人員手動操作。
1.1 技術堆疊
* Backend: Go (Golang)
* Frontend: Native HTML5, CSS3, Vanilla JavaScript (無框架)
* Protocol: TCP/IP (Device Communication), WebSocket (Frontend Updates)
* Format: JSON (UTF-8) with 8-byte Length Header
1.2 核心需求
1. TCP Server: 監聽 Port 50200，處理 TPT 的長連線 (Keep-Alive)。
2. Protocol Parser: 解析 [8-byte Length string] + [JSON Payload] 的封包格式。
3. State Management: 在記憶體中維護所有通道 (CH001-CH128) 的即時狀態。
4. Logic Control (Level 3):
   * 自動回覆 ACK (LINK, STATUS, REPORT)。
   * 發送命令前檢查狀態 (例如：狀態為 Running 時不可發送 Start)。
5. Web GUI: 顯示即時 Log、狀態列表，並提供手動操作按鈕。
2. 專案目錄結構
請依照以下結構建立專案：
/GoTestMES
├── go.mod                  # Go module definition
├── main.go                 # Entry point (TCP Server & HTTP Server setup)
├── /core
│   ├── server_tcp.go       # TCP Listener & Connection Handling
│   ├── server_http.go      # HTTP Routes & WebSocket Hub
│   ├── protocol.go         # Low-level packet reading/writing (8-byte header)
│   └── state_manager.go    # In-memory logic & validation (Level 3)
├── /models
│   └── messages.go         # JSON Struct definitions
└── /static
   ├── index.html          # Main Dashboard
   ├── script.js           # Frontend Logic & WebSocket client
   └── style.css           # Basic UI styling

3. 實作階段詳解
Phase 1: 資料模型定義 (Models)
在 models/messages.go 中定義通訊協定所需的 Struct。
* Base Message: 所有訊息皆包含 type, timestamp, msg_id。
* Specific Payloads:
   * LinkMsg: 包含 work_station_name, channel_count。
   * StatusMsg: 包含 channel, state (StandBy, Running, Alarm, etc.)。
   * StartCmd: 包含 barcode, process, data_path。
   * ReportMsg: 包含 record_path。
   * AckMsg: 通用的 ACK 回覆，包含 reply_to, ack (OK/NG), message。
Phase 2: 核心通訊與協定解析 (TCP Core)
在 core/protocol.go 與 core/server_tcp.go 實作。
1. Header Parsing (關鍵):
   * 讀取 TCP 串流的前 8 個 Bytes (字串格式，如 "00000194" )。
   * 將字串轉為 int (Data Length)。
   * 根據 Length 讀取後續的 JSON Body。
2. Connection Handling:
   * 支援多個 TPT Client 連線（雖通常只有一個，但架構需支援）。
   * 使用 bufio.Scanner 或 io.ReadFull 確保讀取完整的封包。
Phase 3: 狀態管理與業務邏輯 (Level 3 Logic)
在 core/state_manager.go 實作核心大腦。
1. Global State Store:
   * 使用 map[string]ChannelInfo 儲存每個通道的狀態。
   * 必須使用 sync.RWMutex 保護 Map，避免 TCP 寫入與 HTTP 讀取時發生 Race Condition。
2. Auto-Reply Logic:
   * 收到 LINK -> 建立 Session，回覆 LINK_ACK。
   * 收到 STATUS -> 更新 State Store，回覆 STATUS_ACK。
   * 收到 REPORT -> 標記通道為 Finish/StandBy，回覆 REPORT_ACK。
3. Command Validation (防呆機制):
   * Function: ValidateAndStart(channelID string)
   * Logic: 檢查該 Channel 目前狀態。
      * If Running or Offline -> Return Error (拒絕發送)。
      * If StandBy -> 允許發送 START 命令。
Phase 4: Web 介面與 WebSocket (Frontend & Integration)
建立 core/server_http.go 與 /static 檔案。
1. WebSocket Endpoint (/ws):
   * 將 Go 後端接收到的 TPT 訊息 (Log) 即時推送到前端。
   * 將 Go 後端更新的 Channel State 即時推送到前端更新表格。
2. REST API (Control):
   * POST /api/cmd/start: 接收前端請求 -> 呼叫 Phase 3 的 Validation -> 透過 TCP 發送給 TPT。
   * POST /api/cmd/stop: 發送停止命令。
3. Frontend UI (index.html):
   * Connection Status: 顯示 MES 與 TPT 的連線狀況。
   * Log Console: 一個 <textarea readonly> 顯示即時收發的 JSON。
   * Control Panel:
      * 下拉選單選擇 Channel (CH001-CH128)。
      * 按鈕：START, STOP, PAUSE, RESUME。
   * Monitor Table: 顯示所有通道的狀態與顏色標記 (Running=Green, Alarm=Red, StandBy=Grey)。
4. 特殊規則與注意事項 (Constraints)
1. 8-byte Header: 發送 JSON 給 TPT 時，務必計算 JSON byte 長度，並在前面加上 8 碼 ASCII (不足補 0)，例如 fmt.Sprintf("%08d%s", len(jsonStr), jsonStr)。
2. Keep-Alive: TCP 連線不應主動斷開，除非發生 Read Error。
3. Timestamp: 格式必須符合 ISO 8601 擴充格式 (2025-10-17T15:30:00+08:00)。
4. Error Handling: 若收到格式錯誤的封包，Server 不應 Crash，應記錄錯誤並忽略或回傳錯誤 ACK。
5. 執行步驟 (Step-by-Step for Cursor)
1. Step 1: 建立 models 與 JSON structs。
2. Step 2: 實作 core/protocol.go 中的封包封裝與拆解函數。
3. Step 3: 建立 state_manager 並實作 LINK, STATUS 的處理與自動回覆。
4. Step 4: 啟動 TCP Server 監聽 50200，驗證與 TPT 的基礎連線 (Ping-Pong)。
5. Step 5: 建立 HTTP Server 與 WebSocket，實作前端 HTML/JS 頁面。
6. Step 6: 串接前端按鈕至後端 API，實作 START/STOP 的狀態檢查邏輯。