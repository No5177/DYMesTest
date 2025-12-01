package core

import (
	"GoTestMES/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// ChannelState 通道狀態資訊
type ChannelState struct {
	ChannelID string // 通道 ID (例如: "CH001")
	State     string // 當前狀態
	Barcode   string // 條碼（如果正在執行）
	Process   string // 製程名稱
	DataPath  string // 資料路徑
	Message   string // 異常訊息
}

// StateManager 狀態管理器
type StateManager struct {
	mu              sync.RWMutex
	channels        map[string]*ChannelState // 通道狀態 map[ChannelID]State
	workStationName string                   // 工作站名稱
	isConnected     bool                     // TPT 是否已連線
	tptState        string                   // TPT 連線狀態 (Online-Auto, Online-Manual, Offline)
	channelCount    int                      // 通道數量
	broadcastFunc   func(interface{})        // 廣播函數（發送到 WebSocket）
	sendToTPTFunc   func(interface{}) error  // 發送到 TPT 的函數
}

// NewStateManager 建立新的狀態管理器
func NewStateManager(channelCount int) *StateManager {
	sm := &StateManager{
		channels:     make(map[string]*ChannelState),
		channelCount: channelCount,
		isConnected:  false,
		tptState:     models.ConnOffline,
	}

	// 初始化所有通道為 OffLine 狀態
	for i := 1; i <= channelCount; i++ {
		channelID := fmt.Sprintf("CH%03d", i)
		sm.channels[channelID] = &ChannelState{
			ChannelID: channelID,
			State:     models.StateOffLine,
		}
	}

	return sm
}

// SetBroadcastFunc 設定廣播函數
func (sm *StateManager) SetBroadcastFunc(fn func(interface{})) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.broadcastFunc = fn
}

// SetSendToTPTFunc 設定發送到 TPT 的函數
func (sm *StateManager) SetSendToTPTFunc(fn func(interface{}) error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sendToTPTFunc = fn
}

// broadcast 內部廣播函數
func (sm *StateManager) broadcast(data interface{}) {
	if sm.broadcastFunc != nil {
		sm.broadcastFunc(data)
	}
}

// HandleMessage 處理收到的訊息（從 TPT）
func (sm *StateManager) HandleMessage(jsonData []byte) (interface{}, error) {
	// 解析訊息類型
	msgType, err := ParseMessageType(jsonData)
	if err != nil {
		return nil, err
	}

	log.Printf("[StateManager] Received message type: %s", msgType)

	// 廣播原始訊息到前端
	var rawMsg map[string]interface{}
	json.Unmarshal(jsonData, &rawMsg)
	sm.broadcast(map[string]interface{}{
		"direction": "TPT->MES",
		"data":      rawMsg,
	})

	// 根據訊息類型處理
	switch msgType {
	case "LINK":
		return sm.handleLink(jsonData)
	case "STATUS":
		return sm.handleStatus(jsonData)
	case "STATUS_ALL":
		return sm.handleStatusAll(jsonData)
	case "REPORT":
		return sm.handleReport(jsonData)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}

// handleLink 處理 LINK 訊息
func (sm *StateManager) handleLink(jsonData []byte) (interface{}, error) {
	var msg models.LinkMessage
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return nil, err
	}

	sm.mu.Lock()
	sm.isConnected = true
	sm.workStationName = msg.WorkStationName
	sm.tptState = msg.State
	sm.mu.Unlock()

	log.Printf("[LINK] TPT connected: %s, State: %s, Channels: %s",
		msg.WorkStationName, msg.State, msg.ChannelCount)

	// 回覆 LINK_ACK
	ack := models.LinkAckMessage{
		Type:            "LINK_ACK",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: msg.WorkStationName,
		ReplyTo:         msg.MsgID,
		Ack:             models.AckOK,
		Message:         "",
	}

	return ack, nil
}

// handleStatus 處理 STATUS 訊息（單一通道狀態更新）
func (sm *StateManager) handleStatus(jsonData []byte) (interface{}, error) {
	var msg models.StatusMessage
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return nil, err
	}

	sm.mu.Lock()
	if ch, exists := sm.channels[msg.Channel]; exists {
		ch.State = msg.State
		if msg.Message != "" {
			ch.Message = msg.Message
		}
		log.Printf("[STATUS] Channel %s -> %s (msg: %s)", msg.Channel, msg.State, msg.Message)
	}
	sm.mu.Unlock()

	// 回覆 STATUS_ACK
	ack := models.StatusAckMessage{
		Type:            "STATUS_ACK",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: msg.WorkStationName,
		ReplyTo:         msg.MsgID,
		Channel:         msg.Channel,
		Ack:             models.AckOK,
		Message:         "",
	}

	return ack, nil
}

// handleStatusAll 處理 STATUS_ALL 訊息
func (sm *StateManager) handleStatusAll(jsonData []byte) (interface{}, error) {
	var msg models.StatusAllMessage
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return nil, err
	}

	sm.mu.Lock()
	// 更新所有通道狀態
	for _, chInfo := range msg.Channels {
		// STATUS_ALL 使用 "ch": "001" 格式，需要轉換為 "CH001"
		channelID := fmt.Sprintf("CH%s", chInfo.Ch)
		if ch, exists := sm.channels[channelID]; exists {
			ch.State = chInfo.State
			log.Printf("[STATUS_ALL] Channel %s -> %s", channelID, chInfo.State)
		}
	}
	sm.mu.Unlock()

	// 回覆 STATUS_ALL_ACK
	ack := models.StatusAllAckMessage{
		Type:            "STATUS_ALL_ACK",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: msg.WorkStationName,
		ReplyTo:         msg.MsgID,
		Ack:             models.AckOK,
		Message:         "",
	}

	return ack, nil
}

// handleReport 處理 REPORT 訊息
func (sm *StateManager) handleReport(jsonData []byte) (interface{}, error) {
	var msg models.ReportMessage
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return nil, err
	}

	// 將通道轉換為大寫格式（REPORT 可能使用小寫 "ch003"）
	channelID := msg.Channel
	if len(channelID) >= 2 && channelID[:2] == "ch" {
		channelID = "CH" + channelID[2:]
	}

	sm.mu.Lock()
	if ch, exists := sm.channels[channelID]; exists {
		// 完工後設定為 Finish 或 StandBy 狀態
		ch.State = models.StateFinish
		log.Printf("[REPORT] Channel %s finished, record: %s", channelID, msg.RecordPath)
	}
	sm.mu.Unlock()

	// 回覆 REPORT_ACK
	ack := models.ReportAckMessage{
		Type:            "REPORT_ACK",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: msg.WorkStationName,
		ReplyTo:         msg.MsgID,
		Channel:         msg.Channel,
		Ack:             models.AckOK,
		Message:         "",
	}

	return ack, nil
}

// ValidateAndSendStart 驗證並發送 START 命令（Level 3 邏輯）
func (sm *StateManager) ValidateAndSendStart(channelID, barcode, process, dataPath string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 檢查 TPT 是否已連線
	if !sm.isConnected {
		return fmt.Errorf("TPT is not connected")
	}

	// 檢查通道是否存在
	ch, exists := sm.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s does not exist", channelID)
	}

	// Level 3 邏輯：檢查通道狀態
	switch ch.State {
	case models.StateRunning:
		return fmt.Errorf("channel %s is already running", channelID)
	case models.StateOffLine:
		return fmt.Errorf("channel %s is offline", channelID)
	case models.StatePaused:
		return fmt.Errorf("channel %s is paused, use RESUME instead", channelID)
	case models.StateAlarm:
		return fmt.Errorf("channel %s is in alarm state", channelID)
	}

	// 建立 START 命令
	startCmd := models.StartMessage{
		Type:            "START",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: sm.workStationName,
		Channel:         channelID,
		Barcode:         barcode,
		Process:         process,
		DataPath:        dataPath,
	}

	// 發送到 TPT
	if sm.sendToTPTFunc != nil {
		if err := sm.sendToTPTFunc(startCmd); err != nil {
			return fmt.Errorf("failed to send START command: %w", err)
		}
	}

	// 更新本地狀態（預期會變成 Running）
	ch.Barcode = barcode
	ch.Process = process
	ch.DataPath = dataPath

	log.Printf("[START] Sent to channel %s (barcode: %s, process: %s)", channelID, barcode, process)

	// 廣播到前端
	sm.broadcast(map[string]interface{}{
		"direction": "MES->TPT",
		"data":      startCmd,
	})

	return nil
}

// ValidateAndSendStop 驗證並發送 STOP 命令
func (sm *StateManager) ValidateAndSendStop(channelID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isConnected {
		return fmt.Errorf("TPT is not connected")
	}

	ch, exists := sm.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s does not exist", channelID)
	}

	// Level 3 邏輯：只有 Running 或 Paused 狀態可以停止
	if ch.State != models.StateRunning && ch.State != models.StatePaused {
		return fmt.Errorf("channel %s is not running (current state: %s)", channelID, ch.State)
	}

	stopCmd := models.StopMessage{
		Type:            "STOP",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: sm.workStationName,
		Channel:         channelID,
	}

	if sm.sendToTPTFunc != nil {
		if err := sm.sendToTPTFunc(stopCmd); err != nil {
			return fmt.Errorf("failed to send STOP command: %w", err)
		}
	}

	log.Printf("[STOP] Sent to channel %s", channelID)

	sm.broadcast(map[string]interface{}{
		"direction": "MES->TPT",
		"data":      stopCmd,
	})

	return nil
}

// ValidateAndSendPause 驗證並發送 PAUSE 命令
func (sm *StateManager) ValidateAndSendPause(channelID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isConnected {
		return fmt.Errorf("TPT is not connected")
	}

	ch, exists := sm.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s does not exist", channelID)
	}

	// 只有 Running 狀態可以暫停
	if ch.State != models.StateRunning {
		return fmt.Errorf("channel %s is not running (current state: %s)", channelID, ch.State)
	}

	pauseCmd := models.PauseMessage{
		Type:            "PAUSE",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: sm.workStationName,
		Channel:         channelID,
	}

	if sm.sendToTPTFunc != nil {
		if err := sm.sendToTPTFunc(pauseCmd); err != nil {
			return fmt.Errorf("failed to send PAUSE command: %w", err)
		}
	}

	log.Printf("[PAUSE] Sent to channel %s", channelID)

	sm.broadcast(map[string]interface{}{
		"direction": "MES->TPT",
		"data":      pauseCmd,
	})

	return nil
}

// ValidateAndSendResume 驗證並發送 RESUME 命令
func (sm *StateManager) ValidateAndSendResume(channelID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isConnected {
		return fmt.Errorf("TPT is not connected")
	}

	ch, exists := sm.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s does not exist", channelID)
	}

	// 只有 Paused 狀態可以復歸
	if ch.State != models.StatePaused {
		return fmt.Errorf("channel %s is not paused (current state: %s)", channelID, ch.State)
	}

	resumeCmd := models.ResumeMessage{
		Type:            "RESUME",
		Timestamp:       models.GetTimestamp(),
		MsgID:           models.GenerateMsgID(),
		WorkStationName: sm.workStationName,
		Channel:         channelID,
	}

	if sm.sendToTPTFunc != nil {
		if err := sm.sendToTPTFunc(resumeCmd); err != nil {
			return fmt.Errorf("failed to send RESUME command: %w", err)
		}
	}

	log.Printf("[RESUME] Sent to channel %s", channelID)

	sm.broadcast(map[string]interface{}{
		"direction": "MES->TPT",
		"data":      resumeCmd,
	})

	return nil
}

// GetAllChannels 取得所有通道狀態（用於前端顯示）
func (sm *StateManager) GetAllChannels() []ChannelState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	channels := make([]ChannelState, 0, len(sm.channels))
	for i := 1; i <= sm.channelCount; i++ {
		channelID := fmt.Sprintf("CH%03d", i)
		if ch, exists := sm.channels[channelID]; exists {
			channels = append(channels, *ch)
		}
	}
	return channels
}

// GetConnectionStatus 取得連線狀態
func (sm *StateManager) GetConnectionStatus() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"tpt_connected":     sm.isConnected,      // TPT 狀態（收到 LINK 後為 true）
		"work_station_name": sm.workStationName,
		"tpt_state":         sm.tptState,
		"channel_count":     sm.channelCount,
	}
}
