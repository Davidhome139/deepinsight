package tts

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// CosyVoice v3 音色
	VoiceLonganyang = "longanyang" // 龙艳阳 - 标准女声
)

// Model options
const (
	ModelCosyVoice = "cosyvoice-v1"
	ModelCosyVoiceV3 = "cosyvoice-v3-flash"
	ModelSambert   = "sambert-zhichu-v1"
)

// TTSService handles text-to-speech conversion using Aliyun CosyVoice
type TTSService struct {
	apiKey    string
	voiceToken string
	appKey     string
}

// NewTTSService creates a new TTS service using Aliyun
func NewTTSService(apiKey string, voiceToken string, appKey string) *TTSService {
	return &TTSService{
		apiKey:    apiKey,
		voiceToken: voiceToken,
		appKey:     appKey,
	}
}

// SpeakRequest represents a TTS API request for Aliyun
type SpeakRequest struct {
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

// SpeakResponse represents a TTS API response from Aliyun
type SpeakResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Audio string `json:"audio"` // base64 encoded audio
	} `json:"output"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Speak converts text to speech and returns audio data
func (s *TTSService) Speak(text, voice, model string, speed float64) ([]byte, error) {
	// Set default values
	if voice == "" {
		voice = "xiaoyun" // 使用默认发音人
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

	// Log the parameters being used
	fmt.Printf("TTS service parameters: text length=%d, voice=%s, speed=%f, voiceToken=%s, appKey=%s\n", len(text), voice, speed, s.voiceToken, s.appKey)

	// 检查是否有语音token和appkey
	if s.voiceToken == "" || s.appKey == "" {
		return nil, fmt.Errorf("voice token and appkey are required for TTS service")
	}

	// 分段处理文本，每段不超过300字符
	var allAudio []byte
	const maxLength = 300
	
	for i := 0; i < len(text); i += maxLength {
		end := i + maxLength
		if end > len(text) {
			end = len(text)
		}
		
		segment := text[i:end]
		fmt.Printf("Processing segment %d-%d: %s\n", i, end, segment)
		
		audio, err := s.speakSegment(segment, voice, speed)
		if err != nil {
			return nil, fmt.Errorf("failed to process segment %d-%d: %w", i, end, err)
		}
		
		allAudio = append(allAudio, audio...)
	}

	// 返回拼接后的音频数据
	fmt.Printf("TTS service success, total audio data length: %d\n", len(allAudio))
	return allAudio, nil
}

// speakSegment converts a single segment of text to speech
func (s *TTSService) speakSegment(text, voice string, speed float64) ([]byte, error) {
	// 使用正确的API端点
	baseURL := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/tts"
	
	// 构建查询参数
	params := url.Values{}
	params.Add("appkey", s.appKey)
	params.Add("token", s.voiceToken)
	params.Add("text", text)
	params.Add("format", "mp3")
	params.Add("sample_rate", "16000") // 使用16000Hz采样率
	params.Add("voice", voice)
	params.Add("volume", "50")
	params.Add("speech_rate", "0") // 使用默认语速
	params.Add("pitch_rate", "0") // 使用默认语调
	
	// 构建完整URL
	fullURL := baseURL + "?" + params.Encode()
	
	// Log the full request URL
	fmt.Printf("TTS request URL: %s\n", fullURL)
	
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := readAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 检查响应是否为音频数据
	contentType := resp.Header.Get("Content-Type")
	if contentType != "audio/mpeg" {
		return nil, fmt.Errorf("TTS API error: %s", string(body))
	}

	// 返回音频数据
	return body, nil
}

// readAll reads all data from the reader
func readAll(r io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	return buf.Bytes(), err
}

// GetAvailableVoices returns list of available voices (Chinese voices)
func (s *TTSService) GetAvailableVoices() []map[string]string {
	return []map[string]string{
		{"id": VoiceLonganyang, "name": "龙艳阳", "description": "标准女声，适合多种场景"},
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
		{"id": ModelCosyVoiceV3, "name": "CosyVoice V3", "description": "阿里云语音合成，自然流畅"},
		{"id": ModelCosyVoice, "name": "CosyVoice V1", "description": "阿里云语音合成，自然流畅"},
		{"id": ModelSambert, "name": "Sambert", "description": "阿里云标准语音合成"},
	}
}
