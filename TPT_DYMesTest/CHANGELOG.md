# 更新日誌

## 2025-12-29 v1.0.3 - 新增 RSP_STATUS 和自訂命令功能

### 新增功能

#### 1. RSP_STATUS 命令
- 新增 RSP_STATUS 按鈕（紫色，圖示：📊）
- 可向 TPT 發送狀態請求命令
- 命令格式：
  ```json
  {
    "type": "RSP_STATUS",
    "timestamp": "2025-12-29T15:35:00+08:00",
    "msg_id": "20251229153500",
    "work_station_name": "TPT-001"
  }
  ```

#### 2. 自訂命令 (User Command)
- 新增自訂命令輸入區塊
- 使用者可輸入任意命令類型
- 系統自動產生 timestamp 和 msg_id
- 命令格式：
  ```json
  {
    "type": "使用者輸入的類型",
    "timestamp": "2025-12-29T15:35:00+08:00",
    "msg_id": "20251229153500"
  }
  ```

### 修改檔案

#### 前端
1. **static/index.html**
   - 在「其他命令」區塊新增 RSP_STATUS 按鈕
   - 新增「自訂命令」區塊，包含輸入框和發送按鈕

2. **static/script.js**
   - 新增 `sendRspStatusCommand()` 函數
   - 新增 `sendUserCommand()` 函數
   - 更新 `initEventListeners()` 註冊新按鈕事件

3. **static/style.css**
   - 新增 `.btn-info` 樣式（紫色，用於 RSP_STATUS）
   - 新增 `.btn-custom` 樣式（青綠色，用於自訂命令）

#### 後端
4. **core/server_http.go**
   - 新增 `/api/cmd/rsp_status` API 端點
   - 新增 `/api/cmd/user_command` API 端點
   - 新增 `handleRspStatusCommand()` 處理函數
   - 新增 `handleUserCommand()` 處理函數
   - 新增 `UserCommandRequest` 結構

5. **core/state_manager.go**
   - 新增 `SendRspStatus()` 方法
   - 新增 `SendUserCommand()` 方法
   - 兩個方法都包含連線狀態檢查
   - 自動廣播命令到 WebSocket 前端

### 功能特點

✅ **連線檢查**: 發送命令前自動檢查 TPT 連線狀態  
✅ **輸入驗證**: 自訂命令的 type 欄位必填  
✅ **自動時間戳記**: 使用 ISO 8601 格式（+08:00 時區）  
✅ **自動訊息 ID**: 系統自動產生唯一的訊息 ID  
✅ **即時 Log**: 所有命令都會顯示在通訊 Log 中  
✅ **錯誤處理**: 完整的錯誤訊息提示  

### 使用方式

#### RSP_STATUS 命令
1. 確認 TPT 已連線
2. 點擊「📊 RSP_STATUS」按鈕
3. 系統自動發送命令並顯示在 Log 中

#### 自訂命令
1. 在輸入框中輸入命令類型（例如：`aaa`）
2. 點擊「📤 發送自訂命令」按鈕
3. 系統發送命令並顯示在 Log 中

### API 端點

- **RSP_STATUS**: `POST /api/cmd/rsp_status`
  - Request: 無需 body
  - Response: `{"status": "ok"}` 或錯誤訊息

- **自訂命令**: `POST /api/cmd/user_command`
  - Request: `{"type": "命令類型"}`
  - Response: `{"status": "ok"}` 或錯誤訊息

### 測試建議

1. 測試 RSP_STATUS 命令發送
2. 測試自訂命令發送（例如：`aaa`、`TEST_COMMAND`）
3. 測試空白輸入驗證
4. 測試未連線時的錯誤處理
5. 檢查通訊 Log 是否正確顯示命令

### 相關文件

- `新功能說明.md` - 詳細功能說明
- `快速測試指南.md` - 測試步驟和預期結果

---

## 2025-12-01 v2 - 簡化訊息格式

### 重大變更

**取消 8-byte Length Header，改用純粹的 JSON + \r\n 格式**

#### 舊格式
```
[8-byte Length][JSON]\r\n
例如: 00000194{"type":"LINK",...}\r\n
```

#### 新格式
```
[JSON]\r\n
例如: {"type":"LINK",...}\r\n
```

### 修改內容

1. **protocol.go - ReadMessage()**
   - 使用 `bufio.Reader.ReadString('\n')` 讀取一行
   - 自動處理 `\r\n` 或 `\n` 結束符號
   - 移除 8-byte header 解析邏輯

2. **protocol.go - WriteMessage()**
   - 直接寫入 JSON
   - 加上 `\r\n` 結束符號
   - 移除 8-byte header 生成邏輯

3. **server_tcp.go - handleConnection()**
   - 建立 `bufio.Reader` 用於讀取訊息
   - 支援行導向的訊息讀取

### 優點

- ✅ 更簡單的協定
- ✅ 更容易除錯（可直接用文字工具查看）
- ✅ 相容標準的行導向協定
- ✅ 減少封包解析複雜度

---

## 2025-12-01 v1 - 修正 TPT 通訊問題

### 問題描述

1. **TCP 連線成功但 Web 顯示離線**
   - TPT 可以成功連線到 MES (Port 50200)
   - 但 Web 介面仍顯示「離線」狀態

2. **JSON 解析錯誤**
   - 錯誤訊息: `invalid character '0' in string escape code`
   - 原因: TPT 發送的 JSON 中包含 Windows 路徑，使用單一反斜線 `\`
   - 例如: `"data_path": "D:\00 Processing\..."`
   - JSON 標準中 `\0` 不是合法的跳脫序列

### 解決方案

#### 1. 修正 JSON 跳脫字元處理 (`core/protocol.go`)

新增 `fixInvalidEscapeSequences()` 函數：
- 自動檢測並修正不合法的跳脫字元
- 在不合法的反斜線前再加一個反斜線
- 例如: `D:\00` → `D:\\00`

**修改內容:**
```go
// 在 ReadMessage() 中加入
jsonData = fixInvalidEscapeSequences(jsonData)

// 新增函數
func fixInvalidEscapeSequences(data []byte) []byte {
    // 處理 Windows 路徑中的單一反斜線問題
    // 合法的跳脫字元: \", \\, \/, \b, \f, \n, \r, \t, \u
    // 其他的 \x 都會被修正為 \\x
}
```

#### 2. 加入 TCP 結束符號 (`core/protocol.go`)

根據 TPT 的協定要求，在每個訊息結尾加上 `\r\n`：

**發送訊息 (WriteMessage):**
```go
// 寫入 JSON payload
writer.Write(jsonData)

// 寫入結束符號 \r\n
writer.Write([]byte("\r\n"))
```

**接收訊息 (ReadMessage):**
```go
// 讀取 JSON payload
io.ReadFull(reader, jsonData)

// 讀取並丟棄結束符號 \r\n
terminator := make([]byte, 2)
io.ReadFull(reader, terminator)
```

### 訊息格式

#### 完整格式
```
[JSON]\r\n
```

#### 範例
```
{"type":"LINK","timestamp":"2025-12-01T10:30:00+08:00",...}\r\n
```

### 測試方法

1. **重新編譯程式**
```bash
cd GoTestMES
go build -o GoTestMES.exe
```

2. **啟動伺服器**
```bash
GoTestMES.exe
```
或
```bash
run.bat
```

3. **測試連線**
   - 啟動 TPT ThinkLab
   - 設定 MES IP 和 Port (50200)
   - 觀察 Web 介面 (http://localhost:8080)
   - 確認連線狀態變為「已連線」

4. **測試訊息**
   - TPT 發送 LINK → MES 回覆 LINK_ACK
   - TPT 發送 STATUS_ALL → MES 回覆 STATUS_ALL_ACK
   - 所有訊息都應該正常解析，不再出現跳脫字元錯誤

### 預期結果

✅ TCP 連線成功  
✅ JSON 正確解析（包含 Windows 路徑）  
✅ Web 介面顯示「已連線」  
✅ 通道狀態正常更新  
✅ 所有命令正常收發  

### 技術細節

#### JSON 跳脫字元規則

**合法的跳脫序列:**
- `\"` - 雙引號
- `\\` - 反斜線
- `\/` - 正斜線
- `\b` - 退格
- `\f` - 換頁
- `\n` - 換行
- `\r` - 回車
- `\t` - Tab
- `\uXXXX` - Unicode 字元

**不合法的範例:**
- `\0` - 不是合法的跳脫序列
- `\S` - 不是合法的跳脫序列
- `\M` - 不是合法的跳脫序列

#### Windows 路徑處理

**錯誤格式:**
```json
{"path": "D:\00 Processing\STM2511006\MES"}
```

**修正後:**
```json
{"path": "D:\\00 Processing\\STM2511006\\MES"}
```

或使用正斜線（JSON 中也合法）:
```json
{"path": "D:/00 Processing/STM2511006/MES"}
```

### 相關檔案

- `core/protocol.go` - 協定處理與 JSON 修正
- `core/state_manager.go` - 狀態管理
- `core/server_tcp.go` - TCP 伺服器

### 注意事項

1. **訊息結束符號**
   - 每個 JSON 訊息必須以 `\r\n` 結尾
   - 程式使用 `bufio.Reader` 讀取完整的一行

2. **相容性**
   - 支援 `\r\n` 或單純 `\n` 作為結束符號
   - 自動去除結尾的換行符號

3. **效能**
   - JSON 修正只在必要時進行
   - 對效能影響極小（< 1ms）

### 下一步

如果還有問題，請檢查：
1. TPT 的 Log 檔案
2. MES 的 Console 輸出
3. Web 介面的「通訊 Log」區域

---

**版本**: 1.0.1  
**更新日期**: 2025-12-01  
**修正問題**: JSON 跳脫字元、TCP 結束符號

