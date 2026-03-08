package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Voice options for Aliyun CosyVoice (中文语音)
const (
	VoiceLongxiaochun = "longxiaochun" // 龙小淳 - 知性女声
	VoiceLonglaotie   = "longlaotie"   // 龙老铁 - 东北老铁
	VoiceLongshu      = "longshu"      // 龙叔 - 沉稳男声
	VoiceLongxiaoxia  = "longxiaoxia"  // 龙小夏 - 活泼女声
	VoiceLongyue      = "longyue"      // 龙悦 - 温柔女声
	VoiceLongfei      = "longfei"      // 龙飞 - 阳光男声
)

// Model options
const (
	ModelCosyVoice = "cosyvoice-v1"
	ModelSambert   = "sambert-zhichu-v1"
)

// TTSService handles text-to-speech conversion using Aliyun CosyVoice
type TTSService struct {
	apiKey  string
	baseURL string
}

// TTSRequest represents a TTS API request for Aliyun
type TTSRequest struct {
	Model      string     `json:"model"`
	Input      ttsInput   `json:"input"`
	Parameters *ttsParams `json:"parameters,omitempty"`
}

type ttsInput struct {
	Text string `json:"text"`
}

type ttsParams struct {
	Voice      string  `json:"voice,omitempty"`
	Format     string  `json:"format,omitempty"`
	SampleRate int     `json:"sample_rate,omitempty"`
	Rate       float64 `json:"rate,omitempty"`
	Volume     int     `json:"volume,omitempty"`
}

type ttsResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Audio string `json:"audio"` // base64 encoded audio
	} `json:"output"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewTTSService creates a new TTS service using Aliyun
func NewTTSService(apiKey string) *TTSService {
	return &TTSService{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2audio/text-to-speech",
	}
}

// SetBaseURL allows overriding the API endpoint
func (s *TTSService) SetBaseURL(url string) {
	s.baseURL = url
}

// Speak converts text to speech and returns audio data
func (s *TTSService) Speak(text, voice, model string, speed float64) ([]byte, error) {
	if voice == "" {
		voice = VoiceLongxiaochun
	}
	if model == "" {
		model = ModelCosyVoice
	}
	if speed <= 0 {
		speed = 1.0
	}
	if speed < 0.5 {
		speed = 0.5
	}
	if speed > 2.0 {
		speed = 2.0
	}

	reqBody := TTSRequest{
		Model: model,
		Input: ttsInput{Text: text},
		Parameters: &ttsParams{
			Voice:      voice,
			Format:     "mp3",
			SampleRate: 22050,
			Rate:       speed,
			Volume:     50,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// For Aliyun, check if response is audio directly or JSON
	contentType := resp.Header.Get("Content-Type")
	if contentType == "audio/mpeg" || contentType == "audio/mp3" {
		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read audio response: %w", err)
		}
		return audioData, nil
	}

	// Otherwise parse JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp ttsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// If parsing fails, the body might be raw audio
		return body, nil
	}

	if apiResp.Code != "" {
		return nil, fmt.Errorf("TTS API error: %s - %s", apiResp.Code, apiResp.Message)
	}

	// Return audio directly if available
	return body, nil
}

// GetAvailableVoices returns list of available voices (Chinese voices)
func (s *TTSService) GetAvailableVoices() []map[string]string {
	return []map[string]string{
		{"id": VoiceLongxiaochun, "name": "龙小淳", "description": "知性女声，适合新闻、教育"},
		{"id": VoiceLongyue, "name": "龙悦", "description": "温柔女声，适合故事、陪伴"},
		{"id": VoiceLongxiaoxia, "name": "龙小夏", "description": "活泼女声，适合娱乐、广告"},
		{"id": VoiceLongshu, "name": "龙叔", "description": "沉稳男声，适合新闻、商务"},
		{"id": VoiceLongfei, "name": "龙飞", "description": "阳光男声，适合娱乐、广告"},
		{"id": VoiceLonglaotie, "name": "龙老铁", "description": "东北方言，适合娱乐"},
	}
}

// GetAvailableModels returns list of available TTS models
func (s *TTSService) GetAvailableModels() []map[string]string {
	return []map[string]string{
		{"id": ModelCosyVoice, "name": "CosyVoice", "description": "阿里云语音合成，自然流畅"},
		{"id": ModelSambert, "name": "Sambert", "description": "阿里云标准语音合成"},
	}
}
