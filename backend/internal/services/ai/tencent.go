package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/config"
)

type tencentService struct {
	config config.AIProviderConfig
}

func NewTencentService(cfg config.AIProviderConfig) AIService {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://hunyuan.tencentcloudapi.com"
	}
	return &tencentService{config: cfg}
}

func (s *tencentService) GetAvailableModels() []string {
	return []string{"hunyuan-lite", "hunyuan-standard", "hunyuan-pro", "hunyuan-turbo"}
}

// Tencent Cloud API V3 Signature
func (s *tencentService) sign(httpMethod, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, hashedRequestPayload, timestamp, date string) string {
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	algorithm := "TC3-HMAC-SHA256"
	service := "hunyuan"
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256Hex(canonicalRequest)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	secretDate := hmacSHA256([]byte("TC3"+s.config.SecretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		s.config.SecretID,
		credentialScope,
		signedHeaders,
		signature)

	fmt.Printf("[Tencent SDK] Signature Debug:\n")
	fmt.Printf("  - Canonical Request Hash: %s\n", hashedCanonicalRequest)
	fmt.Printf("  - String to Sign: %s\n", stringToSign[:100]+"...")
	fmt.Printf("  - Signature: %s\n", signature)

	return authorization
}

func sha256Hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type tencentChatRequest struct {
	Model    string           `json:"Model"`
	Messages []tencentMessage `json:"Messages"`
	Stream   bool             `json:"Stream"`
	// Tencent Hunyuan uses "EnableEnhancement" not "EnableEnhance"
	EnableEnhancement bool `json:"EnableEnhancement,omitempty"`
}

type tencentMessage struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

func (s *tencentService) ChatStream(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error) {
	ch := make(chan ChatChunk)

	// Validate credentials
	if s.config.SecretID == "" || s.config.SecretKey == "" {
		return nil, fmt.Errorf("Tencent SecretID or SecretKey is empty. Please configure in Settings.")
	}

	fmt.Printf("[Tencent SDK] SecretID: %s..., SecretKey length: %d\n",
		s.config.SecretID[:minInt(10, len(s.config.SecretID))], len(s.config.SecretKey))

	messages := make([]tencentMessage, 0, len(req.Messages))

	// Note: Custom system prompt is already merged into message history at chat service level
	// No need to prepend it separately here

	for _, m := range req.Messages {
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		} else if role == "system" {
			role = "system"
		} else {
			role = "user"
		}
		messages = append(messages, tencentMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	payload := tencentChatRequest{
		Model:             req.Model,
		Messages:          messages,
		Stream:            true,
		EnableEnhancement: req.WebSearch, // Enable Tencent's native search
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[Tencent SDK] Request payload (full): %s\n", string(jsonData))

	// Tencent Cloud API V3 requires specific headers
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	date := time.Unix(time.Now().Unix(), 0).UTC().Format("2006-01-02")
	host := "hunyuan.tencentcloudapi.com"
	httpMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	action := "ChatCompletions"

	// Prepare headers - MUST be in alphabetical order
	contentType := "application/json; charset=utf-8"
	hashedRequestPayload := sha256Hex(string(jsonData))

	fmt.Printf("[Tencent SDK] Payload preview: %s\n", string(jsonData)[:minInt(200, len(jsonData))]+"...")
	fmt.Printf("[Tencent SDK] Payload hash: %s\n", hashedRequestPayload)

	// Canonical headers must be lowercase and sorted alphabetically
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", contentType, host)
	signedHeaders := "content-type;host"

	authorization := s.sign(httpMethod, canonicalURI, canonicalQueryString,
		canonicalHeaders, signedHeaders, hashedRequestPayload, timestamp, date)

	fmt.Printf("[Tencent SDK] Request details:\n")
	fmt.Printf("  - Endpoint: %s\n", s.config.BaseURL)
	fmt.Printf("  - Model: %s\n", req.Model)
	fmt.Printf("  - EnableEnhancement: %v\n", req.WebSearch)
	fmt.Printf("  - Timestamp: %s\n", timestamp)
	fmt.Printf("  - Date: %s\n", date)
	fmt.Printf("  - Payload length: %d bytes\n", len(jsonData))
	fmt.Printf("  - Authorization: %s\n", authorization[:50]+"...")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-TC-Action", action)
	httpReq.Header.Set("X-TC-Version", "2023-09-01")
	httpReq.Header.Set("X-TC-Timestamp", timestamp)
	httpReq.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("[Tencent SDK] HTTP request failed: %v\n", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Tencent SDK] Error response (%d): %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Tencent Cloud API error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[Tencent SDK] Successfully connected, starting to stream response\n")
	fmt.Printf("[Tencent SDK] Response Status: %d\n", resp.StatusCode)
	fmt.Printf("[Tencent SDK] Response Headers: %v\n", resp.Header)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		// Use bufio.Scanner for real-time line-by-line streaming
		scanner := bufio.NewScanner(resp.Body)
		lineCount := 0

		for scanner.Scan() {
			line := scanner.Text()
			lineCount++

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			fmt.Printf("[Tencent SDK] Line %d: %s\n", lineCount, line)

			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				fmt.Println("[Tencent SDK] Received [DONE] signal")
				ch <- ChatChunk{Done: true}
				return
			}

			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content string `json:"Content"`
					} `json:"Delta"`
				} `json:"Choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				fmt.Printf("[Tencent SDK] Failed to parse response: %s, data: %s\n", err, data)
				continue
			}

			fmt.Printf("[Tencent SDK] Parsed response: %+v\n", streamResp)
			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					fmt.Printf("[Tencent SDK] Sending content chunk: %s\n", content)
					ch <- ChatChunk{Content: content}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("[Tencent SDK] Scanner error: %v\n", err)
		}

		fmt.Println("[Tencent SDK] Stream ended")
	}()

	return ch, nil
}
