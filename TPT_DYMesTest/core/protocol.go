package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ReadMessage 從 TCP 連線讀取一個完整的訊息
// 格式: [JSON]\r\n
func ReadMessage(reader io.Reader) ([]byte, error) {
	// 使用 bufio.Reader 讀取一行（直到 \r\n）
	bufReader, ok := reader.(*bufio.Reader)
	if !ok {
		bufReader = bufio.NewReader(reader)
	}

	// 讀取直到 \r\n
	line, err := bufReader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// 移除結尾的 \r\n 或 \n
	line = strings.TrimSuffix(line, "\r\n")
	line = strings.TrimSuffix(line, "\n")

	// 轉換為 bytes
	jsonData := []byte(line)

	// 驗證長度合理性（防止記憶體溢出攻擊）
	if len(jsonData) == 0 || len(jsonData) > 10*1024*1024 { // 最大 10MB
		return nil, fmt.Errorf("invalid message length: %d", len(jsonData))
	}

	// 修正不合法的 JSON 跳脫字元（處理 Windows 路徑）
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
// 格式: [JSON]\r\n
func WriteMessage(writer io.Writer, data interface{}) error {
	// 1. 將資料序列化為 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// 2. 寫入 JSON payload
	_, err = writer.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write JSON payload: %w", err)
	}

	// 3. 寫入結束符號 \r\n
	_, err = writer.Write([]byte("\r\n"))
	if err != nil {
		return fmt.Errorf("failed to write terminator: %w", err)
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
	return string(jsonData) + "\r\n", nil
}

