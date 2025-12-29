// WebSocket 連線
let ws = null;
let reconnectTimer = null;

// 狀態資料
let channels = [];
let connectionStatus = {
    connected: false,
    work_station_name: 'N/A',
    tpt_state: 'N/A'
};

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    initWebSocket();
    initEventListeners();
    initChannelSelect();
    loadChannels();
});

// 初始化 WebSocket
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = function() {
        console.log('WebSocket 連線成功');
        addLog('系統', 'WebSocket 連線成功', 'info');
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
    };
    
    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            handleWebSocketMessage(data);
        } catch (e) {
            console.error('解析 WebSocket 訊息失敗:', e);
        }
    };
    
    ws.onerror = function(error) {
        console.error('WebSocket 錯誤:', error);
        addLog('系統', 'WebSocket 連線錯誤', 'error');
    };
    
    ws.onclose = function() {
        console.log('WebSocket 連線關閉');
        addLog('系統', 'WebSocket 連線關閉，3秒後重連...', 'warning');
        
        // 3秒後重連
        reconnectTimer = setTimeout(function() {
            initWebSocket();
        }, 3000);
    };
}

// 處理 WebSocket 訊息
function handleWebSocketMessage(data) {
    if (data.type === 'initial_state') {
        // 初始狀態
        connectionStatus = data.status;
        channels = data.channels || [];
        updateConnectionStatus();
        updateChannelTable();
    } else if (data.direction) {
        // 通訊 Log
        const direction = data.direction;
        const msgData = data.data;
        const msgType = msgData.type || 'Unknown';
        
        addLog(direction, JSON.stringify(msgData, null, 2), 
               direction === 'TPT->MES' ? 'receive' : 'send');
        
        // 如果是狀態更新，重新載入通道列表
        if (msgType.includes('STATUS') || msgType.includes('REPORT')) {
            setTimeout(loadChannels, 500);
        }
    }
}

// 初始化事件監聽器
function initEventListeners() {
    // START 按鈕
    document.getElementById('btn-start').addEventListener('click', sendStartCommand);
    
    // STOP 按鈕
    document.getElementById('btn-stop').addEventListener('click', sendStopCommand);
    
    // PAUSE 按鈕
    document.getElementById('btn-pause').addEventListener('click', sendPauseCommand);
    
    // RESUME 按鈕
    document.getElementById('btn-resume').addEventListener('click', sendResumeCommand);
    
    // RSP_STATUS 按鈕
    document.getElementById('btn-rsp-status').addEventListener('click', sendRspStatusCommand);
    
    // 自訂命令按鈕
    document.getElementById('btn-user-command').addEventListener('click', sendUserCommand);
    
    // 清除 Log 按鈕
    document.getElementById('btn-clear-log').addEventListener('click', clearLog);
    
    // 篩選器
    document.getElementById('filter-running').addEventListener('change', updateChannelTable);
    document.getElementById('filter-standby').addEventListener('change', updateChannelTable);
    document.getElementById('filter-alarm').addEventListener('change', updateChannelTable);
    document.getElementById('filter-offline').addEventListener('change', updateChannelTable);
}

// 初始化通道選擇下拉選單
function initChannelSelect() {
    const select = document.getElementById('channel-select');
    for (let i = 1; i <= 128; i++) {
        const option = document.createElement('option');
        const channelId = `CH${String(i).padStart(3, '0')}`;
        option.value = channelId;
        option.textContent = channelId;
        select.appendChild(option);
    }
}

// 載入通道狀態
async function loadChannels() {
    try {
        const response = await fetch('/api/channels');
        if (response.ok) {
            channels = await response.json();
            updateChannelTable();
        }
    } catch (e) {
        console.error('載入通道狀態失敗:', e);
    }
    
    // 同時更新連線狀態
    loadConnectionStatus();
}

// 載入連線狀態
async function loadConnectionStatus() {
    try {
        const response = await fetch('/api/status');
        if (response.ok) {
            connectionStatus = await response.json();
            updateConnectionStatus();
        }
    } catch (e) {
        console.error('載入連線狀態失敗:', e);
    }
}

// 更新連線狀態顯示
function updateConnectionStatus() {
    const tcpStatus = document.getElementById('tcp-status');
    const tptStatus = document.getElementById('tpt-status');
    const workstationName = document.getElementById('workstation-name');
    
    // TCP 連線狀態（純粹的 socket 連接）
    if (connectionStatus.tcp_connected) {
        tcpStatus.textContent = '已連線';
        tcpStatus.className = 'status-badge online';
    } else {
        tcpStatus.textContent = '離線';
        tcpStatus.className = 'status-badge offline';
    }
    
    // TPT 狀態（收到 LINK 後才顯示）
    if (connectionStatus.tpt_connected) {
        tptStatus.textContent = connectionStatus.tpt_state || 'Online';
        tptStatus.className = 'status-badge online';
    } else {
        tptStatus.textContent = '未連線';
        tptStatus.className = 'status-badge offline';
    }
    
    workstationName.textContent = connectionStatus.work_station_name || 'N/A';
}

// 更新通道表格
function updateChannelTable() {
    const tbody = document.getElementById('channel-tbody');
    tbody.innerHTML = '';
    
    // 取得篩選器狀態
    const filters = {
        running: document.getElementById('filter-running').checked,
        standby: document.getElementById('filter-standby').checked,
        alarm: document.getElementById('filter-alarm').checked,
        offline: document.getElementById('filter-offline').checked
    };
    
    channels.forEach(channel => {
        // 篩選邏輯
        const state = channel.State.toLowerCase();
        if (state.includes('running') && !filters.running) return;
        if (state.includes('standby') && !filters.standby) return;
        if (state.includes('alarm') && !filters.alarm) return;
        if (state.includes('offline') && !filters.offline) return;
        
        const row = document.createElement('tr');
        
        // 通道 ID
        const tdChannel = document.createElement('td');
        tdChannel.textContent = channel.ChannelID;
        row.appendChild(tdChannel);
        
        // 狀態
        const tdState = document.createElement('td');
        const stateBadge = document.createElement('span');
        stateBadge.className = `state-badge state-${getStateClass(channel.State)}`;
        stateBadge.textContent = channel.State;
        tdState.appendChild(stateBadge);
        row.appendChild(tdState);
        
        // 條碼
        const tdBarcode = document.createElement('td');
        tdBarcode.textContent = channel.Barcode || '-';
        row.appendChild(tdBarcode);
        
        // 製程
        const tdProcess = document.createElement('td');
        tdProcess.textContent = channel.Process || '-';
        row.appendChild(tdProcess);
        
        // 訊息
        const tdMessage = document.createElement('td');
        tdMessage.textContent = channel.Message || '-';
        tdMessage.className = channel.Message ? 'message-cell' : '';
        row.appendChild(tdMessage);
        
        tbody.appendChild(row);
    });
}

// 取得狀態對應的 CSS class
function getStateClass(state) {
    const s = state.toLowerCase();
    if (s.includes('running')) return 'running';
    if (s.includes('standby')) return 'standby';
    if (s.includes('paused')) return 'paused';
    if (s.includes('alarm')) return 'alarm';
    if (s.includes('finish')) return 'finish';
    if (s.includes('offline')) return 'offline';
    return 'default';
}

// 發送 START 命令
async function sendStartCommand() {
    const channel = document.getElementById('channel-select').value;
    const barcode = document.getElementById('barcode-input').value.trim();
    const process = document.getElementById('process-input').value.trim();
    const dataPath = document.getElementById('datapath-input').value.trim();
    
    if (!channel) {
        alert('請選擇通道');
        return;
    }
    
    if (!barcode || !process || !dataPath) {
        alert('請填寫所有必要欄位（條碼、製程、資料路徑）');
        return;
    }
    
    try {
        const response = await fetch('/api/cmd/start', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ channel, barcode, process, data_path: dataPath })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', `START 命令已發送至 ${channel}`, 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送 START 命令失敗:', e);
        addLog('錯誤', '發送 START 命令失敗: ' + e.message, 'error');
    }
}

// 發送 STOP 命令
async function sendStopCommand() {
    const channel = document.getElementById('channel-select').value;
    
    if (!channel) {
        alert('請選擇通道');
        return;
    }
    
    try {
        const response = await fetch('/api/cmd/stop', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ channel })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', `STOP 命令已發送至 ${channel}`, 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送 STOP 命令失敗:', e);
        addLog('錯誤', '發送 STOP 命令失敗: ' + e.message, 'error');
    }
}

// 發送 PAUSE 命令
async function sendPauseCommand() {
    const channel = document.getElementById('channel-select').value;
    
    if (!channel) {
        alert('請選擇通道');
        return;
    }
    
    try {
        const response = await fetch('/api/cmd/pause', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ channel })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', `PAUSE 命令已發送至 ${channel}`, 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送 PAUSE 命令失敗:', e);
        addLog('錯誤', '發送 PAUSE 命令失敗: ' + e.message, 'error');
    }
}

// 發送 RESUME 命令
async function sendResumeCommand() {
    const channel = document.getElementById('channel-select').value;
    
    if (!channel) {
        alert('請選擇通道');
        return;
    }
    
    try {
        const response = await fetch('/api/cmd/resume', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ channel })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', `RESUME 命令已發送至 ${channel}`, 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送 RESUME 命令失敗:', e);
        addLog('錯誤', '發送 RESUME 命令失敗: ' + e.message, 'error');
    }
}

// 發送 RSP_STATUS 命令
async function sendRspStatusCommand() {
    try {
        const response = await fetch('/api/cmd/rsp_status', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', 'RSP_STATUS 命令已發送', 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送 RSP_STATUS 命令失敗:', e);
        addLog('錯誤', '發送 RSP_STATUS 命令失敗: ' + e.message, 'error');
    }
}

// 發送自訂命令
async function sendUserCommand() {
    const commandType = document.getElementById('user-command-input').value.trim();
    
    if (!commandType) {
        alert('請輸入命令類型');
        return;
    }
    
    try {
        const response = await fetch('/api/cmd/user_command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type: commandType })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            addLog('命令', `自訂命令已發送 (type: ${commandType})`, 'success');
        } else {
            addLog('錯誤', result.error || '發送失敗', 'error');
            alert('錯誤: ' + (result.error || '發送失敗'));
        }
    } catch (e) {
        console.error('發送自訂命令失敗:', e);
        addLog('錯誤', '發送自訂命令失敗: ' + e.message, 'error');
    }
}

// 新增 Log
function addLog(source, message, type = 'info') {
    const logConsole = document.getElementById('log-console');
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry log-${type}`;
    
    const timestamp = new Date().toLocaleTimeString('zh-TW', { hour12: false });
    
    logEntry.innerHTML = `
        <span class="log-time">[${timestamp}]</span>
        <span class="log-source">[${source}]</span>
        <span class="log-message">${escapeHtml(message)}</span>
    `;
    
    logConsole.appendChild(logEntry);
    logConsole.scrollTop = logConsole.scrollHeight;
    
    // 限制 Log 數量（最多 500 條）
    while (logConsole.children.length > 500) {
        logConsole.removeChild(logConsole.firstChild);
    }
}

// 清除 Log
function clearLog() {
    document.getElementById('log-console').innerHTML = '';
    addLog('系統', 'Log 已清除', 'info');
}

// HTML 跳脫
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 定期更新通道狀態（每 5 秒）
setInterval(loadChannels, 5000);

