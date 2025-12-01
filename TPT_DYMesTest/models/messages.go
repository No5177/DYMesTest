package models

import "time"

// ChannelState 通道狀態常數
const (
	StateStandBy          = "StandBy"          // 待機
	StateRunning          = "Running"          // 運轉中
	StatePaused           = "Paused"           // 暫停
	StateStartFailed      = "StartFailed"      // 啟動失敗
	StateChangeStepFailed = "ChangeStepFailed" // 換段失敗
	StateResumeFailed     = "ResumeFailed"     // 回復失敗
	StateAlarm            = "Alarm"            // 故障
	StateNoLoad           = "NoLoad"           // 無載
	StateFinish           = "Finish"           // 完工
	StateReversePolarity  = "ReversePolarity"  // 逆極性
	StateOffLine          = "OffLine"          // 未連線
)

// ConnectionState 連線狀態
const (
	ConnOnlineAuto   = "Online-Auto"   // 連線自動模式
	ConnOnlineManual = "Online-Manual" // 連線手動模式
	ConnOffline      = "Offline"       // 離線
)

// AckStatus ACK 狀態
const (
	AckOK = "OK" // 成功
	AckNG = "NG" // 失敗
)

// BaseMessage 基礎訊息結構（所有訊息共用欄位）
type BaseMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
}

// LinkMessage LINK 訊息 (TPT -> MES)
type LinkMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	State           string `json:"state"`            // Online-Auto, Online-Manual, Offline
	ChannelCount    string `json:"channel_count"`    // 通道數量
	SoftwareVersion string `json:"software_version"` // 軟體版本
}

// LinkAckMessage LINK_ACK 訊息 (MES -> TPT)
type LinkAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Ack             string `json:"ack"`     // OK/NG
	Message         string `json:"message"` // 錯誤訊息（成功時為空）
}

// ChannelInfo STATUS_ALL 中的通道資訊
type ChannelInfo struct {
	Ch    string `json:"ch"`    // 通道編號 (例如: "001")
	State string `json:"state"` // 通道狀態
}

// StatusAllMessage STATUS_ALL 訊息 (TPT -> MES)
type StatusAllMessage struct {
	Type            string        `json:"type"`
	Timestamp       string        `json:"timestamp"`
	MsgID           string        `json:"msg_id"`
	WorkStationName string        `json:"work_station_name"`
	ConnectionState string        `json:"connection_state"` // 連線狀態 (16進位字串)
	Channels        []ChannelInfo `json:"channels"`         // 通道列表
}

// StatusAllAckMessage STATUS_ALL_ACK 訊息 (MES -> TPT)
type StatusAllAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// StatusMessage STATUS 訊息 (TPT -> MES) - 單一通道狀態更新
type StatusMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"`           // 通道編號 (例如: "CH005")
	State           string `json:"state"`             // 通道狀態
	Message         string `json:"message,omitempty"` // 異常訊息（選填）
}

// StatusAckMessage STATUS_ACK 訊息 (MES -> TPT)
type StatusAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// StartMessage START 命令 (MES -> TPT)
type StartMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"`   // 通道編號 (例如: "CH003")
	Barcode         string `json:"barcode"`   // 條碼
	Process         string `json:"process"`   // 製程名稱
	DataPath        string `json:"data_path"` // 資料路徑
}

// StartAckMessage START_ACK 訊息 (TPT -> MES)
type StartAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// StopMessage STOP 命令 (MES -> TPT)
type StopMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"` // 通道編號 (例如: "CH003")
}

// StopAckMessage STOP_ACK 訊息 (TPT -> MES)
type StopAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// PauseMessage PAUSE 命令 (MES -> TPT)
type PauseMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"` // 通道編號 (例如: "CH003")
}

// PauseAckMessage PAUSE_ACK 訊息 (TPT -> MES)
type PauseAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// ResumeMessage RESUME 命令 (MES -> TPT)
type ResumeMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"` // 通道編號 (例如: "CH003")
}

// ResumeAckMessage RESUME_ACK 訊息 (TPT -> MES)
type ResumeAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// ReportMessage REPORT 訊息 (TPT -> MES)
type ReportMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	Channel         string `json:"channel"`     // 通道編號 (例如: "ch003")
	RecordPath      string `json:"record_path"` // 記錄檔案路徑
}

// ReportAckMessage REPORT_ACK 訊息 (MES -> TPT)
type ReportAckMessage struct {
	Type            string `json:"type"`
	Timestamp       string `json:"timestamp"`
	MsgID           string `json:"msg_id"`
	WorkStationName string `json:"work_station_name"`
	ReplyTo         string `json:"reply_to"`
	Channel         string `json:"channel"`
	Ack             string `json:"ack"`
	Message         string `json:"message"`
}

// Helper Functions

// GetTimestamp 取得當前時間戳記（ISO 8601 格式）
func GetTimestamp() string {
	// 使用台北時區 (UTC+8)
	loc, _ := time.LoadLocation("Asia/Taipei")
	return time.Now().In(loc).Format("2006-01-02T15:04:05+08:00")
}

// GenerateMsgID 產生訊息 ID (16 碼 HEX)
func GenerateMsgID() string {
	return time.Now().Format("20060102150405") + "01" // 簡化版本，實際可用 UUID
}
