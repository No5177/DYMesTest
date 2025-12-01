package core

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// ReadMessage 從 TCP 連線讀取一個完整的訊息
// 格式: [JSON]\r\n
func ReadMessage(reader io.Reader) ([]byte, error) {
	// 確保使用 bufio.Reader
	bufReader, ok := reader.(*bufio.Reader)
	if !ok {
		log.Printf("[Protocol] Creating new bufio.Reader")
		bufReader = bufio.NewReader(reader)
	} else {
		log.Printf("[Protocol] Using existing bufio.Reader")
	}

	// 檢查 buffer 中是否已有資料
	buffered := bufReader.Buffered()
	log.Printf("[Protocol] Buffered data available: %d bytes", buffered)

	// 持續讀取直到獲得完整訊息
	// 使用真正的 \r\n (0x0D 0x0A) 作為結束符號
	loopCount := 0
	var lineBytes []byte

	for {
		loopCount++
		log.Printf("[Protocol] ========== Read Loop #%d ==========", loopCount)
		log.Printf("[Protocol] >>> Reading byte by byte until \\r\\n (0x0D 0x0A)...")
		os.Stdout.Sync()

		// 逐 byte 讀取，直到遇到 \r\n (0x0D 0x0A)
		for {
			b, err := bufReader.ReadByte()
			if err != nil {
				// 如果是 EOF 或其他讀取錯誤，直接返回
				log.Printf("[Protocol] ❌ Read error: %v", err)
				os.Stdout.Sync()
				return nil, fmt.Errorf("failed to read byte: %w", err)
			}

			lineBytes = append(lineBytes, b)

			// 檢查是否已收到完整的 \r\n 結束符號 (0x0D 0x0A)
			if len(lineBytes) >= 2 &&
				lineBytes[len(lineBytes)-2] == 0x0D && // \r (Carriage Return)
				lineBytes[len(lineBytes)-1] == 0x0A { // \n (Line Feed)
				log.Printf("[Protocol] ✓ Found \\r\\n terminator (0x0D 0x0A)!")
				os.Stdout.Sync()
				break
			}

			// 防止無限讀取（最大 10MB）
			if len(lineBytes) > 10*1024*1024 {
				log.Printf("[Protocol] ❌ Message too large: %d bytes", len(lineBytes))
				os.Stdout.Sync()
				return nil, fmt.Errorf("message too large: %d bytes", len(lineBytes))
			}
		}

		log.Printf("[Protocol] <<< Complete message received!")
		log.Printf("[Protocol] ✓ Received %d bytes (including terminator)", len(lineBytes))
		os.Stdout.Sync()

		// 記錄原始資料（Hex dump 前 100 bytes）
		dumpLen := min(100, len(lineBytes))
		log.Printf("[Protocol] Raw bytes (hex): %s", hex.EncodeToString(lineBytes[:dumpLen]))
		log.Printf("[Protocol] Raw string: %q", string(lineBytes[:dumpLen]))

		// 顯示最後 2 bytes（應該是 \r\n）
		log.Printf("[Protocol] Last 2 bytes (hex): %s", hex.EncodeToString(lineBytes[len(lineBytes)-2:]))
		log.Printf("[Protocol] Last 2 bytes (string): %q", string(lineBytes[len(lineBytes)-2:]))

		// 移除結尾的 \r\n (2 bytes)
		lineBytes = lineBytes[:len(lineBytes)-2]
		log.Printf("[Protocol] ✓ Removed terminator, message length: %d bytes", len(lineBytes))

		// 轉換為字串
		line := string(lineBytes)

		// 如果是空行，繼續讀取下一行
		if len(line) == 0 {
			log.Printf("[Protocol] Empty line, reading next...")
			continue
		}

		// 轉換為 bytes
		jsonData := []byte(line)

		// 驗證長度合理性（防止記憶體溢出攻擊）
		if len(jsonData) > 10*1024*1024 { // 最大 10MB
			log.Printf("[Protocol] ❌ Message too large: %d bytes", len(jsonData))
			return nil, fmt.Errorf("message too large: %d bytes", len(jsonData))
		}

		// 檢查是否以 { 開頭（基本的 JSON 格式檢查）
		if jsonData[0] != '{' {
			log.Printf("[Protocol] ❌ Invalid JSON: does not start with '{', got: %c", jsonData[0])
			return nil, fmt.Errorf("invalid JSON format: does not start with '{', got: %s", string(jsonData[:min(50, len(jsonData))]))
		}

		log.Printf("[Protocol] ✓ Valid JSON format detected, length: %d", len(jsonData))

		// 修正不合法的 JSON 跳脫字元（處理 Windows 路徑）
		jsonData = fixInvalidEscapeSequences(jsonData)

		return jsonData, nil
	}
}

// min 輔助函數
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 輔助函數
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// fixInvalidEscapeSequences 修正 JSON 中不合法的跳脫字元
// 主要處理 Windows 路徑中的單一反斜線問題
func fixInvalidEscapeSequences(data []byte) []byte {
	var result bytes.Buffer
	inString := false
	escaped := false
	fixedCount := 0

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
				fixedCount++
				log.Printf("[Protocol] Fixed invalid escape sequence: \\%c", ch)
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

	if fixedCount > 0 {
		log.Printf("[Protocol] ✓ Fixed %d invalid escape sequences", fixedCount)
	}

	return result.Bytes()
}

// WriteMessage 將訊息寫入 TCP 連線
// 格式: [JSON] + \r\n (0x0D 0x0A)
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

	// 3. 寫入真正的 \r\n (0x0D 0x0A)
	_, err = writer.Write([]byte{0x0D, 0x0A}) // \r\n
	if err != nil {
		return fmt.Errorf("failed to write terminator: %w", err)
	}

	log.Printf("[Protocol] ✓ Sent message with \\r\\n terminator (0x0D 0x0A)")

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
