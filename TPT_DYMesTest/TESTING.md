# 測試指南

## 快速開始測試

### 1. 啟動 MES 伺服器

**方法 A: 使用啟動腳本（推薦）**
```bash
run.bat
```

**方法 B: 直接執行**
```bash
GoTestMES.exe
```

**方法 C: 自訂參數**
```bash
GoTestMES.exe -tcp-port 50200 -http-port 8080 -channels 128
```

### 2. 開啟 Web 介面

在瀏覽器中開啟：
```
http://localhost:8080
```

您應該會看到：
- ✅ 美觀的 Web 介面
- ✅ 連線狀態顯示為「離線」（等待 TPT 連線）
- ✅ 128 個通道，狀態都是「OffLine」

## 測試場景

### 場景 1: 模擬 TPT 連線（使用 Telnet）

如果您沒有實際的 TPT 客戶端，可以使用 Telnet 模擬：

1. **安裝 Telnet（Windows）**
```powershell
# 以管理員身份執行
dism /online /Enable-Feature /FeatureName:TelnetClient
```

2. **連線到 MES**
```bash
telnet localhost 50200
```

3. **發送 LINK 訊息**

手動輸入（注意：需要精確計算長度）：
```
00000194{"type":"LINK","timestamp":"2025-12-01T10:30:00+08:00","msg_id":"A1B2C3D4E5F6A7B8","work_station_name":"TPT-001","state":"Online-Auto","channel_count":"50","software_version":"v1.2.3"}
```

**注意**: 實際使用時，建議使用專門的 TCP 測試工具（如下方介紹）。

### 場景 2: 使用 TCP 測試工具

推薦使用以下工具之一：

#### A. Hercules SETUP Utility
- 下載: https://www.hw-group.com/software/hercules-setup-utility
- 功能: 強大的 TCP/UDP 測試工具
- 使用方式:
  1. 開啟 Hercules
  2. 選擇 "TCP Client"
  3. 輸入 Host: `localhost`, Port: `50200`
  4. 點擊 "Connect"
  5. 在 "Send" 區域輸入完整訊息（含 8-byte header）
  6. 點擊 "Send"

#### B. SocketTest
- 下載: http://sockettest.sourceforge.net/
- 輕量級 TCP/UDP 測試工具

#### C. 使用 Python 腳本

建立 `test_client.py`:

```python
import socket
import json
import time

def send_message(sock, msg_dict):
    """發送訊息到 MES"""
    json_str = json.dumps(msg_dict)
    json_bytes = json_str.encode('utf-8')
    length = len(json_bytes)
    header = f"{length:08d}".encode('ascii')
    
    print(f"發送: {header.decode()}{json_str}")
    sock.sendall(header + json_bytes)
    
    # 接收回覆
    header = sock.recv(8)
    if header:
        length = int(header.decode('ascii'))
        response = sock.recv(length)
        print(f"收到: {response.decode('utf-8')}")
        return json.loads(response)
    return None

# 連線到 MES
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(('localhost', 50200))

try:
    # 1. 發送 LINK
    link_msg = {
        "type": "LINK",
        "timestamp": "2025-12-01T10:30:00+08:00",
        "msg_id": "A1B2C3D4E5F6A7B8",
        "work_station_name": "TPT-001",
        "state": "Online-Auto",
        "channel_count": "128",
        "software_version": "v1.0.0"
    }
    send_message(sock, link_msg)
    time.sleep(1)
    
    # 2. 發送 STATUS_ALL
    status_all_msg = {
        "type": "STATUS_ALL",
        "timestamp": "2025-12-01T10:30:05+08:00",
        "msg_id": "A1B2C3D4E5F6A7B9",
        "work_station_name": "TPT-001",
        "connection_state": "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
        "channels": [
            {"ch": "001", "state": "StandBy"},
            {"ch": "002", "state": "StandBy"},
            {"ch": "003", "state": "Running"},
        ]
    }
    send_message(sock, status_all_msg)
    time.sleep(1)
    
    # 3. 等待 MES 發送命令
    print("\n等待 MES 命令...")
    print("請在 Web 介面操作...")
    
    while True:
        header = sock.recv(8)
        if not header:
            break
        length = int(header.decode('ascii'))
        data = sock.recv(length)
        msg = json.loads(data)
        print(f"\n收到命令: {msg['type']}")
        print(json.dumps(msg, indent=2, ensure_ascii=False))
        
        # 自動回覆 ACK
        ack_msg = {
            "type": msg['type'] + "_ACK",
            "timestamp": "2025-12-01T10:30:10+08:00",
            "msg_id": "B8A7F6E5D4C3B2A1",
            "work_station_name": "TPT-001",
            "reply_to": msg['msg_id'],
            "channel": msg.get('channel', ''),
            "ack": "OK",
            "message": ""
        }
        send_message(sock, ack_msg)

except KeyboardInterrupt:
    print("\n中斷連線")
finally:
    sock.close()
```

執行：
```bash
python test_client.py
```

### 場景 3: Web 介面功能測試

#### 測試 START 命令

1. 確認 TPT 已連線（連線狀態顯示「已連線」）
2. 在控制面板選擇通道：`CH001`
3. 填寫資訊：
   - 條碼: `TEST123456789`
   - 製程: `TEST-20251201-001`
   - 資料路徑: `C:\ThinkLab4\record`
4. 點擊「START」按鈕
5. 觀察：
   - ✅ Log 顯示發送的 START 命令
   - ✅ TPT 客戶端收到命令
   - ✅ 通道狀態更新

#### 測試狀態驗證（Level 3 邏輯）

**測試 1: 重複 START**
1. 對已經 Running 的通道發送 START
2. 預期結果: ❌ 顯示錯誤「channel is already running」

**測試 2: 對 OffLine 通道 START**
1. 選擇狀態為 OffLine 的通道
2. 點擊 START
3. 預期結果: ❌ 顯示錯誤「channel is offline」

**測試 3: PAUSE 非 Running 通道**
1. 選擇狀態為 StandBy 的通道
2. 點擊 PAUSE
3. 預期結果: ❌ 顯示錯誤「channel is not running」

**測試 4: 正常流程**
1. StandBy → START → Running ✅
2. Running → PAUSE → Paused ✅
3. Paused → RESUME → Running ✅
4. Running → STOP → StandBy ✅

### 場景 4: WebSocket 即時更新測試

1. 開啟兩個瀏覽器視窗，都連到 `http://localhost:8080`
2. 在其中一個視窗發送命令
3. 觀察另一個視窗是否即時更新：
   - ✅ Log 即時顯示
   - ✅ 通道狀態即時更新

### 場景 5: 壓力測試

#### 測試多通道同時操作

使用 Python 腳本模擬多個通道同時變更狀態：

```python
import socket
import json
import time
import random

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(('localhost', 50200))

def send_message(msg_dict):
    json_str = json.dumps(msg_dict)
    json_bytes = json_str.encode('utf-8')
    length = len(json_bytes)
    header = f"{length:08d}".encode('ascii')
    sock.sendall(header + json_bytes)
    
    # 接收回覆
    header = sock.recv(8)
    if header:
        length = int(header.decode('ascii'))
        sock.recv(length)

# LINK
send_message({
    "type": "LINK",
    "timestamp": "2025-12-01T10:30:00+08:00",
    "msg_id": "A1B2C3D4E5F6A7B8",
    "work_station_name": "TPT-001",
    "state": "Online-Auto",
    "channel_count": "128",
    "software_version": "v1.0.0"
})

# 模擬 128 個通道快速變更狀態
states = ["StandBy", "Running", "Paused", "Alarm", "Finish"]
for i in range(1, 129):
    status_msg = {
        "type": "STATUS",
        "timestamp": "2025-12-01T10:30:00+08:00",
        "msg_id": f"MSG{i:08d}",
        "work_station_name": "TPT-001",
        "channel": f"CH{i:03d}",
        "state": random.choice(states)
    }
    send_message(status_msg)
    time.sleep(0.01)  # 10ms 間隔

print("壓力測試完成")
sock.close()
```

觀察：
- ✅ Web 介面是否流暢更新
- ✅ 伺服器是否穩定運行
- ✅ 記憶體使用是否正常

## 驗證清單

### 基本功能
- [ ] TCP 伺服器正常啟動（Port 50200）
- [ ] HTTP 伺服器正常啟動（Port 8080）
- [ ] Web 介面可正常開啟
- [ ] WebSocket 連線成功

### 通訊協定
- [ ] 正確解析 8-byte header
- [ ] 正確解析 JSON payload
- [ ] 正確發送 8-byte header + JSON
- [ ] 支援所有訊息類型（LINK, STATUS, START, etc.）

### 狀態管理
- [ ] 正確維護 128 個通道狀態
- [ ] 狀態更新即時反映到 Web 介面
- [ ] 多個 WebSocket 客戶端同步更新

### Level 3 邏輯
- [ ] START 命令狀態驗證正確
- [ ] STOP 命令狀態驗證正確
- [ ] PAUSE 命令狀態驗證正確
- [ ] RESUME 命令狀態驗證正確
- [ ] 錯誤訊息清楚明確

### Web 介面
- [ ] 連線狀態正確顯示
- [ ] 通道列表正確顯示
- [ ] Log 即時更新
- [ ] 篩選功能正常
- [ ] 按鈕操作正常
- [ ] 表單驗證正常

### 錯誤處理
- [ ] 格式錯誤的封包不會導致 Crash
- [ ] 網路斷線後可自動重連（WebSocket）
- [ ] 錯誤訊息記錄到 Log

## 效能指標

### 預期效能
- **TCP 連線處理**: < 10ms
- **訊息解析**: < 1ms
- **狀態更新**: < 5ms
- **WebSocket 推送**: < 10ms
- **記憶體使用**: < 100MB（128 通道）
- **CPU 使用**: < 5%（閒置時）

### 壓力測試目標
- **同時連線數**: 支援至少 5 個 TPT 客戶端
- **訊息處理速率**: > 1000 msg/sec
- **通道狀態更新**: 128 個通道同時更新 < 100ms

## 常見問題

### Q1: 為什麼 Telnet 測試失敗？
A: Telnet 會自動加入換行符號，導致 JSON 格式錯誤。建議使用專門的 TCP 測試工具或 Python 腳本。

### Q2: Web 介面顯示「WebSocket 連線關閉」？
A: 這是正常的，系統會自動在 3 秒後重連。如果持續失敗，請檢查伺服器是否正常運行。

### Q3: 發送命令後沒有反應？
A: 檢查：
1. TPT 是否已連線
2. 通道狀態是否符合命令要求
3. 查看 Log 了解詳細錯誤

### Q4: 如何查看詳細的通訊內容？
A: 所有收發的訊息都會顯示在 Web 介面的「通訊 Log」區域。

## 下一步

測試完成後，您可以：
1. 連接實際的 TPT ThinkLab 客戶端進行整合測試
2. 根據實際需求調整通道數量
3. 擴展更多功能（例如：資料庫記錄、報表生成等）

---

**祝測試順利！** 🎉

