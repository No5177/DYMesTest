# TPT MES 測試伺服器 (GoTestMES)

## 專案簡介

這是一個模擬 MES (Manufacturing Execution System) 的伺服器應用程式，用於驗證 TPT ThinkLab 客戶端的連線與通訊流程。

### 主要特性

- ✅ **TCP/IP 通訊**: 監聽 Port 50200，處理 TPT 的長連線 (Keep-Alive)
- ✅ **簡潔協定**: 支援 `[JSON]\r\n` 封包格式
- ✅ **Level 3 邏輯**: 維護通道狀態、驗證命令邏輯（防呆機制）
- ✅ **Web GUI**: 即時顯示 Log、狀態列表，並提供手動操作介面
- ✅ **WebSocket 即時更新**: 前端即時接收通訊訊息與狀態變化
- ✅ **128 通道支援**: 支援最多 128 個測試通道

## 技術堆疊

- **Backend**: Go 1.21+
- **Frontend**: Native HTML5, CSS3, Vanilla JavaScript
- **Protocol**: TCP/IP (Device Communication), WebSocket (Frontend Updates)
- **Format**: JSON (UTF-8) with \r\n terminator

## 專案結構

```
GoTestMES/
├── go.mod                  # Go module definition
├── main.go                 # Entry point
├── core/
│   ├── protocol.go         # JSON message parsing with \r\n terminator
│   ├── state_manager.go    # State management & Level 3 logic
│   ├── server_tcp.go       # TCP server & connection handling
│   └── server_http.go      # HTTP routes & WebSocket hub
├── models/
│   └── messages.go         # JSON struct definitions
└── static/
    ├── index.html          # Main dashboard
    ├── script.js           # Frontend logic & WebSocket client
    └── style.css           # UI styling
```

## 安裝與執行

### 前置需求

- Go 1.21 或更高版本
- 網頁瀏覽器（Chrome、Firefox、Edge 等）

### 安裝步驟

1. **下載依賴**

```bash
cd GoTestMES
go mod tidy
```

2. **編譯程式**

```bash
go build -o DYMesTest.exe
```

3. **執行程式**

```bash
# 使用預設設定
./DYMesTest.exe

# 或指定自訂參數
./DYMesTest.exe -tcp-port 50200 -http-port 5179 -channels 128
```

### 命令列參數

| 參數 | 說明 | 預設值 |
|------|------|--------|
| `-tcp-port` | TCP 伺服器埠號 | 50200 |
| `-http-port` | HTTP/WebSocket 伺服器埠號 | 5179 |
| `-channels` | 通道數量 | 128 |

### 啟動畫面

```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║     TPT_DYMesTest                                         ║
║     TPT Automated Testing MES Server                      ║
║                                                           ║
║     Version: 1.0.1                                        ║
║     Protocol: TCP/IP + JSON with \r\n terminator          ║
║     Purpose: TPT ThinkLab Communication Testing           ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

Starting TPT MES Test Server...
TCP Port: 50200
HTTP Port: 5179
Channel Count: 128
Web UI: http://localhost:5179
[TCP] Server listening on port 50200
[HTTP] Server starting on http://localhost:5179
✓ Server started successfully!
✓ Waiting for TPT connection on port 50200...
✓ Open web browser: http://localhost:5179
```

## 使用方式

### 1. 啟動伺服器

執行 `DYMesTest.exe` 後，伺服器會：
- 在 Port 50200 監聽 TPT 的 TCP 連線
- 在 Port 5179 提供 Web GUI 介面

### 2. 開啟 Web 介面

在瀏覽器中開啟 `http://localhost:5179`，您會看到：

- **連線狀態區**: 顯示 TCP 連線狀態、TPT 狀態、工作站名稱
- **控制面板**: 選擇通道並發送命令（START, STOP, PAUSE, RESUME）
- **通訊 Log**: 即時顯示所有收發的 JSON 訊息
- **通道監控表**: 顯示所有通道的即時狀態

### 3. TPT 連線

當 TPT ThinkLab 啟動並連線到 MES 時：
1. TPT 會發送 `LINK` 訊息
2. MES 自動回覆 `LINK_ACK`
3. TPT 發送 `STATUS_ALL` 更新所有通道狀態
4. 連線狀態會變為「已連線」

### 4. 發送命令

#### START 命令
1. 選擇通道（例如：CH001）
2. 填寫必要資訊：
   - **條碼**: 例如 `A1234578900BE`
   - **製程**: 例如 `TEST-20251201-001`
   - **資料路徑**: 例如 `C:\ThinkLab4\record`
3. 點擊「START」按鈕

#### 其他命令
- **STOP**: 停止正在運行的通道
- **PAUSE**: 暫停正在運行的通道
- **RESUME**: 恢復已暫停的通道

### 5. 狀態監控

通道狀態會以不同顏色標示：
- 🟢 **Running**: 運轉中（綠色）
- ⚪ **StandBy**: 待機（灰色）
- 🟡 **Paused**: 暫停（橘色）
- 🔴 **Alarm**: 故障（紅色）
- 🔵 **Finish**: 完工（藍色）
- ⚫ **OffLine**: 未連線（深灰色）

## 通訊協定說明

### 封包格式

所有訊息都使用以下格式：

```
[JSON]\r\n
```

範例：
```
{"type":"LINK","timestamp":"2025-12-01T10:30:00+08:00",...}\r\n
```

### 支援的訊息類型

#### TPT → MES
- `LINK`: 連線請求
- `STATUS`: 單一通道狀態更新
- `STATUS_ALL`: 所有通道狀態回報
- `REPORT`: 測試結果回報

#### MES → TPT
- `LINK_ACK`: 連線確認
- `STATUS_ACK`: 狀態確認
- `STATUS_ALL_ACK`: 全部狀態確認
- `START`: 啟動命令
- `STOP`: 停止命令
- `PAUSE`: 暫停命令
- `RESUME`: 復歸命令
- `REPORT_ACK`: 結果確認

### 通道狀態列表

| 狀態 | 說明 |
|------|------|
| `StandBy` | 待機 |
| `Running` | 運轉中 |
| `Paused` | 暫停 |
| `StartFailed` | 啟動失敗 |
| `ChangeStepFailed` | 換段失敗 |
| `ResumeFailed` | 回復失敗 |
| `Alarm` | 故障 |
| `NoLoad` | 無載 |
| `Finish` | 完工 |
| `ReversePolarity` | 逆極性 |
| `OffLine` | 未連線 |

## Level 3 邏輯驗證

伺服器會在發送命令前檢查通道狀態：

### START 命令驗證
- ❌ 通道狀態為 `Running` → 拒絕（已在運行）
- ❌ 通道狀態為 `OffLine` → 拒絕（未連線）
- ❌ 通道狀態為 `Paused` → 拒絕（應使用 RESUME）
- ❌ 通道狀態為 `Alarm` → 拒絕（故障中）
- ✅ 通道狀態為 `StandBy` → 允許

### STOP 命令驗證
- ✅ 通道狀態為 `Running` 或 `Paused` → 允許
- ❌ 其他狀態 → 拒絕

### PAUSE 命令驗證
- ✅ 通道狀態為 `Running` → 允許
- ❌ 其他狀態 → 拒絕

### RESUME 命令驗證
- ✅ 通道狀態為 `Paused` → 允許
- ❌ 其他狀態 → 拒絕

## API 端點

### HTTP REST API

| 端點 | 方法 | 說明 |
|------|------|------|
| `/api/status` | GET | 取得連線狀態 |
| `/api/channels` | GET | 取得所有通道狀態 |
| `/api/cmd/start` | POST | 發送 START 命令 |
| `/api/cmd/stop` | POST | 發送 STOP 命令 |
| `/api/cmd/pause` | POST | 發送 PAUSE 命令 |
| `/api/cmd/resume` | POST | 發送 RESUME 命令 |

### WebSocket

- **端點**: `/ws`
- **用途**: 即時推送通訊訊息與狀態更新

## 故障排除

### TCP 連線失敗

**問題**: TPT 無法連線到 MES

**解決方案**:
1. 確認 Port 50200 未被其他程式佔用
2. 檢查防火牆設定
3. 確認 TPT 設定的 MES IP 位址正確

### Web 介面無法開啟

**問題**: 瀏覽器無法開啟 `http://localhost:5179`

**解決方案**:
1. 確認 Port 5179 未被佔用
2. 嘗試使用其他埠號：`./DYMesTest.exe -http-port 8081`
3. 檢查防火牆設定

### 命令發送失敗

**問題**: 點擊按鈕後顯示錯誤訊息

**解決方案**:
1. 確認 TPT 已連線（連線狀態顯示「已連線」）
2. 檢查通道狀態是否符合命令要求
3. 查看通訊 Log 了解詳細錯誤訊息

## 開發與測試

### 執行測試

```bash
go test ./...
```

### 開發模式

```bash
# 即時編譯並執行
go run main.go
```

### 建立 Release 版本

```bash
# Windows
go build -ldflags="-s -w" -o DYMesTest.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o GoTestMES

# macOS
GOOS=darwin GOARCH=amd64 go build -o GoTestMES
```

## 注意事項

1. **訊息結束符號**: 每個 JSON 訊息必須以 `\r\n` 結尾
2. **Keep-Alive**: TCP 連線不應主動斷開，除非發生 Read Error
3. **Timestamp 格式**: 必須符合 ISO 8601 擴充格式 (`2025-12-01T10:30:00+08:00`)
4. **錯誤處理**: 收到格式錯誤的封包時，Server 不會 Crash，會記錄錯誤並繼續運行
5. **Windows 路徑**: 程式會自動修正 JSON 中不合法的反斜線跳脫字元

## 授權

本專案僅供 TPT ThinkLab 測試使用。

## 聯絡資訊

如有問題或建議，請聯絡開發團隊。

---

**版本**: 1.0.1
**最後更新**: 2025-12-01