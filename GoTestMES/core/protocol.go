package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// ReadMessage 從 TCP 連線讀取一個完整的訊息
// 格式: [8-byte ASCII length] + [JSON payload]
func ReadMessage(reader io.Reader) ([]byte, error) {
	// 1. 讀取 8-byte 長度標頭
	lengthHeader := make([]byte, 8)
	_, err := io.ReadFull(reader, lengthHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to read length header: %w", err)
	}

	// 2. 將 8-byte ASCII 字串轉換為整數
	lengthStr := string(lengthHeader)
	dataLength, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid length header '%s': %w", lengthStr, err)
	}

	// 3. 驗證長度合理性（防止記憶體溢出攻擊）
	if dataLength <= 0 || dataLength > 10*1024*1024 { // 最大 10MB
		return nil, fmt.Errorf("invalid data length: %d", dataLength)
	}

	// 4. 讀取 JSON payload
	jsonData := make([]byte, dataLength)
	_, err = io.ReadFull(reader, jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON payload: %w", err)
	}

	// 5. 修正不合法的 JSON 跳脫字元（處理 Windows 路徑）
	jsonData = fixInvalidEscapeSequences(jsonData)

	return jsonData, nil
}

// fixInvalidEscapeSequences 修正 JSON 中不合法的跳脫字元
// 主要處理 Windows 路徑中的單一反斜線問題
func fixInvalidEscapeSequences(data []byte) []byte {
	var result bytes.Buffer
	inString := false
	escaped := false
	
	for i := 0; i < len(data); i++ {
		ch := data[i]
		
		// 檢查是否在字串內（簡化版本，不處理巢狀情況）
		if ch == '"' && !escaped {
			inString = !inString
			result.WriteByte(ch)
			continue
		}
		
		// 如果不在字串內，直接寫入
		if !inString {
			result.WriteByte(ch)
			escaped = false
			continue
		}
		
		// 在字串內處理跳脫字元
		if escaped {
			// 檢查是否為合法的跳脫字元
			validEscapes := map[byte]bool{
				'"': true, '\\': true, '/': true, 'b': true,
				'f': true, 'n': true, 'r': true, 't': true, 'u': true,
			}
			
			if !validEscapes[ch] {
				// 不合法的跳脫字元，在反斜線前再加一個反斜線
				result.WriteByte('\\')
			}
			result.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
			result.WriteByte(ch)
		} else {
			result.WriteByte(ch)
		}
	}
	
	return result.Bytes()
}

// WriteMessage 將訊息寫入 TCP 連線
// 格式: [8-byte ASCII length] + [JSON payload]
func WriteMessage(writer io.Writer, data interface{}) error {
	// 1. 將資料序列化為 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// 2. 計算 JSON 長度
	dataLength := len(jsonData)

	// 3. 建立 8-byte 長度標頭（補零到 8 位）
	lengthHeader := fmt.Sprintf("%08d", dataLength)
	if len(lengthHeader) != 8 {
		return fmt.Errorf("length header overflow: %d bytes", dataLength)
	}

	// 4. 寫入長度標頭
	_, err = writer.Write([]byte(lengthHeader))
	if err != nil {
		return fmt.Errorf("failed to write length header: %w", err)
	}

	// 5. 寫入 JSON payload
	_, err = writer.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write JSON payload: %w", err)
	}

	return nil
}

// ParseMessageType 解析訊息類型（不完全反序列化）
func ParseMessageType(jsonData []byte) (string, error) {
	var baseMsg struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(jsonData, &baseMsg)
	if err != nil {
		return "", fmt.Errorf("failed to parse message type: %w", err)
	}
	return baseMsg.Type, nil
}

// FormatMessage 格式化訊息為完整的傳輸格式（用於日誌）
func FormatMessage(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	dataLength := len(jsonData)
	lengthHeader := fmt.Sprintf("%08d", dataLength)
	return lengthHeader + string(jsonData), nil
}

