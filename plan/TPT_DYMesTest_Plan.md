# **Project Plan: TPT 自動化測試用 MES 伺服器 (Go \+ Native Web GUI)**

## **1\. 專案目標與概述**

建立一個模擬 MES (Manufacturing Execution System) 的伺服器應用程式，用於驗證 TPT ThinkLab 客戶端的連線與通訊流程。  
此模擬器需具備 Level 3 的邏輯深度，意味著它不僅是單純的 Echo Server，還必須維護通道狀態、驗證命令邏輯，並提供 GUI 介面供測試人員手動操作。

### **1.1 技術堆疊**

* **Backend:** Go (Golang)  
* **Frontend:** Native HTML5, CSS3, Vanilla JavaScript (無框架)  
* **Protocol:** TCP/IP (Device Communication), WebSocket (Frontend Updates)  
* **Format:** JSON (UTF-8) with 8-byte Length Header

### **1.2 核心需求**

1. **TCP Server:** 監聽 Port 50200，處理 TPT 的長連線 (Keep-Alive)。  
2. **Protocol Parser:** 解析 \[8-byte Length string\] \+ \[JSON Payload\] 的封包格式。  
3. **State Management:** 在記憶體中維護所有通道 (CH001-CH128) 的即時狀態。  
4. **Logic Control (Level 3):**  
   * 自動回覆 ACK (LINK, STATUS, REPORT)。  
   * 發送命令前檢查狀態 (例如：狀態為 Running 時不可發送 Start)。  
5. **Web GUI:** 顯示即時 Log、狀態列表，並提供手動操作按鈕。

## **2\. 專案目錄結構**

請依照以下結構建立專案：

/GoTestMES  
├── go.mod                  \# Go module definition  
├── main.go                 \# Entry point (TCP Server & HTTP Server setup)  
├── /core  
│   ├── server\_tcp.go       \# TCP Listener & Connection Handling  
│   ├── server\_http.go      \# HTTP Routes & WebSocket Hub  
│   ├── protocol.go         \# Low-level packet reading/writing (8-byte header)  
│   └── state\_manager.go    \# In-memory logic & validation (Level 3\)  
├── /models  
│   └── messages.go         \# JSON Struct definitions  
└── /static  
    ├── index.html          \# Main Dashboard  
    ├── script.js           \# Frontend Logic & WebSocket client  
    └── style.css           \# Basic UI styling

## **3\. 實作階段詳解**

### **Phase 1: 資料模型定義 (Models)**

在 models/messages.go 中定義通訊協定所需的 Struct。

* **Base Message:** 所有訊息皆包含 type, timestamp, msg\_id。  
* **Specific Payloads:**  
  * LinkMsg: 包含 work\_station\_name, channel\_count。  
  * StatusMsg: 包含 channel, state (StandBy, Running, Alarm, etc.)。  
  * StartCmd: 包含 barcode, process, data\_path。  
  * ReportMsg: 包含 record\_path。  
  * AckMsg: 通用的 ACK 回覆，包含 reply\_to, ack (OK/NG), message。

### **Phase 2: 核心通訊與協定解析 (TCP Core)**

在 core/protocol.go 與 core/server\_tcp.go 實作。

1. **Header Parsing (關鍵):**  
   * 讀取 TCP 串流的前 **8 個 Bytes** (字串格式，如 "00000194" )。  
   * 將字串轉為 int (Data Length)。  
   * 根據 Length 讀取後續的 JSON Body。  
2. **Connection Handling:**  
   * 支援多個 TPT Client 連線（雖通常只有一個，但架構需支援）。  
   * 使用 bufio.Scanner 或 io.ReadFull 確保讀取完整的封包。

### **Phase 3: 狀態管理與業務邏輯 (Level 3 Logic)**

在 core/state\_manager.go 實作核心大腦。

1. **Global State Store:**  
   * 使用 map\[string\]ChannelInfo 儲存每個通道的狀態。  
   * **必須使用 sync.RWMutex** 保護 Map，避免 TCP 寫入與 HTTP 讀取時發生 Race Condition。  
2. **Auto-Reply Logic:**  
   * 收到 LINK \-\> 建立 Session，回覆 LINK\_ACK。  
   * 收到 STATUS \-\> 更新 State Store，回覆 STATUS\_ACK。  
   * 收到 REPORT \-\> 標記通道為 Finish/StandBy，回覆 REPORT\_ACK。  
3. **Command Validation (防呆機制):**  
   * **Function:** ValidateAndStart(channelID string)  
   * **Logic:** 檢查該 Channel 目前狀態。  
     * If Running or Offline \-\> Return Error (拒絕發送)。  
     * If StandBy \-\> 允許發送 START 命令。

### **Phase 4: Web 介面與 WebSocket (Frontend & Integration)**

建立 core/server\_http.go 與 /static 檔案。

1. **WebSocket Endpoint (/ws):**  
   * 將 Go 後端接收到的 TPT 訊息 (Log) 即時推送到前端。  
   * 將 Go 後端更新的 Channel State 即時推送到前端更新表格。  
2. **REST API (Control):**  
   * POST /api/cmd/start: 接收前端請求 \-\> 呼叫 Phase 3 的 Validation \-\> 透過 TCP 發送給 TPT。  
   * POST /api/cmd/stop: 發送停止命令。  
3. **Frontend UI (index.html):**  
   * **Connection Status:** 顯示 MES 與 TPT 的連線狀況。  
   * **Log Console:** 一個 \<textarea readonly\> 顯示即時收發的 JSON。  
   * **Control Panel:**  
     * 下拉選單選擇 Channel (CH001-CH128)。  
     * 按鈕：START, STOP, PAUSE, RESUME。  
   * **Monitor Table:** 顯示所有通道的狀態與顏色標記 (Running=Green, Alarm=Red, StandBy=Grey)。

## **4\. 特殊規則與注意事項 (Constraints)**

1. **8-byte Header:** 發送 JSON 給 TPT 時，**務必**計算 JSON byte 長度，並在前面加上 8 碼 ASCII (不足補 0)，例如 fmt.Sprintf("%08d%s", len(jsonStr), jsonStr)。  
2. **Keep-Alive:** TCP 連線不應主動斷開，除非發生 Read Error。  
3. **Timestamp:** 格式必須符合 ISO 8601 擴充格式 (2025-10-17T15:30:00+08:00)。  
4. **Error Handling:** 若收到格式錯誤的封包，Server 不應 Crash，應記錄錯誤並忽略或回傳錯誤 ACK。

## **5\. 執行步驟 (Step-by-Step for Cursor)**

1. **Step 1:** 建立 models 與 JSON structs。  
2. **Step 2:** 實作 core/protocol.go 中的封包封裝與拆解函數。  
3. **Step 3:** 建立 state\_manager 並實作 LINK, STATUS 的處理與自動回覆。  
4. **Step 4:** 啟動 TCP Server 監聽 50200，驗證與 TPT 的基礎連線 (Ping-Pong)。  
5. **Step 5:** 建立 HTTP Server 與 WebSocket，實作前端 HTML/JS 頁面。  
6. **Step 6:** 串接前端按鈕至後端 API，實作 START/STOP 的狀態檢查邏輯。

## **6\. 命令格式 (Command Formats)**

請依照下列 JSON 格式實作資料結構。注意所有 JSON 封包前皆需加上 8 碼長度檔頭。

1. **TPT → MES**  
   * 命令: LINK (連線請求)  
   * JSON:  
     {  
       "type": "LINK",  
       "timestamp": "2025-10-17T15:30:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7B8",  
       "work\_station\_name": "TPT-001",  
       "state": "Online-Auto",  
       "channel\_count": "50",  
       "software\_version": "v1.2.3"  
     }

2. **MES → TPT**  
   * 命令: LINK\_ACK (連線確認)  
   * JSON:  
     {  
       "type": "LINK\_ACK",  
       "timestamp": "2025-10-17T15:30:01+08:00",  
       "msg\_id": "B8A7F6E5D4C3B2A1",  
       "work\_station\_name": "TPT-001",  
       "reply\_to": "A1B2C3D4E5F6A7B8",  
       "ack": "OK",  
       "message": ""  
     }

3. **TPT → MES**  
   * 命令: STATUS\_ALL (所有通道狀態回報)  
   * JSON:  
     {  
       "type": "STATUS\_ALL",  
       "timestamp": "2025-10-17T15:35:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7B9",  
       "work\_station\_name": "TPT-001",  
       "connection state": "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",  
       "channels": \[  
         {"ch": "001", "state": "RUNNING"},  
         {"ch": "002", "state": "STOP"},  
         {"ch": "003", "state": "ALARM"},  
         {"ch": "004", "state": "OFFLINE"}  
       \]  
     }

4. **MES → TPT**  
   * 命令: STATUS\_ALL\_ACK (狀態回報確認)  
   * JSON:  
     {  
       "type": "STATUS\_ALL\_ACK",  
       "timestamp": "2025-10-17T15:35:00+08:00",  
       "msg\_id": "B8A7F6E5D4C3B2A2",  
       "work\_station\_name": "TPT-001",  
       "reply\_to": "A1B2C3D4E5F6A7B9",  
       "ack": "OK",  
       "message": ""  
     }

5. **TPT → MES**  
   * 命令: STATUS (單一通道狀態更新)  
   * JSON:  
     {  
       "type": "STATUS",  
       "timestamp": "2025-10-17T15:35:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7B9",  
       "work\_station\_name": "TPT-001",  
       "channel": "CH005",  
       "state": "RUNNING"  
     }

   * 異常狀態範例 (ALARM):  
     {  
       "type": "STATUS",  
       "timestamp": "2025-10-17T15:35:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7BA",  
       "work\_station\_name": "TPT-001",  
       "channel": "CH006",  
       "state": "ALARM",  
       "message": "OVP"  
     }

6. **MES → TPT**  
   * 命令: STATUS\_ACK (單一通道狀態確認)  
   * JSON:  
     {  
       "type": "STATUS\_ACK",  
       "timestamp": "2025-10-17T15:35:00+08:00",  
       "msg\_id": "B8A7F6E5D4C3B2A2",  
       "work\_station\_name": "TPT-001",  
       "reply\_to": "A1B2C3D4E5F6A7B9",  
       "channel": "CH005",  
       "ack": "OK",  
       "message": ""  
     }

7. **MES → TPT**  
   * 命令: START (啟動命令)  
   * JSON:  
     {  
       "type": "START",  
       "timestamp": "2025-10-17T15:36:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7BB",  
       "work\_station\_name": "TPT-001",  
       "channel": "CH003",  
       "barcode": "A1234578900BE",  
       "process": "TEST-20251017-001",  
       "data\_path": "C:\\\\ThinkLab4\\\\record"  
     }

8. **TPT → MES**  
   * 命令: START\_ACK (啟動確認)  
   * JSON:  
     {  
       "type": "START\_ACK",  
       "timestamp": "2025-10-17T15:36:00+08:00",  
       "msg\_id": "B8A7F6E5D4C3B2A4",  
       "work\_station\_name": "TPT-001",  
       "reply\_to": "A1B2C3D4E5F6A7BB",  
       "channel": "CH003",  
       "ack": "OK",  
       "message": ""  
     }

9. **MES → TPT**  
   * 命令: STOP (停止命令)  
   * JSON:  
     {  
       "type": "STOP",  
       "timestamp": "2025-10-17T15:36:00+08:00",  
       "msg\_id": "A1B2C3D4E5F6A7BC",  
       "work\_station\_name": "TPT-001",  
       "channel": "CH003"  
     }

10. **TPT → MES**  
    * 命令: STOP\_ACK (停止確認)  
    * JSON (成功):  
      {  
        "type": "STOP\_ACK",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "B8A7F6E5D4C3B2A5",  
        "work\_station\_name": "TPT-001",  
        "reply\_to": "A1B2C3D4E5F6A7BC",  
        "channel": "CH003",  
        "ack": "OK",  
        "message": ""  
      }

    * JSON (失敗 \- 例如通道未運行):  
      {  
        "type": "STOP\_ACK",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "B8A7F6E5D4C3B2A6",  
        "work\_station\_name": "TPT-001",  
        "reply\_to": "A1B2C3D4E5F6A7BD",  
        "channel": "CH005",  
        "ack": "NG",  
        "message": "Channel is not running."  
      }

11. **MES → TPT**  
    * 命令: PAUSE (暫停命令)  
    * JSON:  
      {  
        "type": "PAUSE",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "A1B2C3D4E5F6A7BE",  
        "work\_station\_name": "TPT-001",  
        "channel": "CH003"  
      }

12. **TPT → MES**  
    * 命令: PAUSE\_ACK (暫停確認)  
    * JSON:  
      {  
        "type": "PAUSE\_ACK",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "B8A7F6E5D4C3B2A7",  
        "work\_station\_name": "TPT-001",  
        "reply\_to": "A1B2C3D4E5F6A7BE",  
        "channel": "CH003",  
        "ack": "OK",  
        "message": ""  
      }

13. **MES → TPT**  
    * 命令: RESUME (復歸命令)  
    * JSON:  
      {  
        "type": "RESUME",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "A1B2C3D4E5F6A7BF",  
        "work\_station\_name": "TPT-001",  
        "channel": "CH003"  
      }

14. **TPT → MES**  
    * 命令: RESUME\_ACK (復歸確認)  
    * JSON:  
      {  
        "type": "RESUME\_ACK",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "B8A7F6E5D4C3B2A8",  
        "work\_station\_name": "TPT-001",  
        "reply\_to": "A1B2C3D4E5F6A7BF",  
        "channel": "CH003",  
        "ack": "OK",  
        "message": ""  
      }

15. **TPT → MES**  
    * 命令: REPORT (結果回報 \- 完工或停止時)  
    * JSON:  
      {  
        "type": "REPORT",  
        "timestamp": "2025-10-17T15:36:00+08:00",  
        "msg\_id": "A1B2C3D4E5F6A7C0",  
        "work\_station\_name": "TPT-001",  
        "channel": "ch003",  
        "record\_path": "C:\\\\ThinkLab\\\\record\\\\WSThinkPower\_20240516\\\\WSThinkPower CH001 20240516101240.fud"  
      }

16. **MES → TPT**  
    * 命令: REPORT\_ACK (結果回報確認)  
    * JSON:  
      {  
        "type": "REPORT\_ACK",  
        "timestamp": "2025-10-17T15:35:00+08:00",  
        "msg\_id": "B8A7F6E5D4C3B2A9",  
        "work\_station\_name": "TPT-001",  
        "reply\_to": "A1B2C3D4E5F6A7C0",  
        "channel": "CH003",  
        "ack": "OK",  
        "message": ""  
      }  
