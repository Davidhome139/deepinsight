package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/pkg/database"
)

type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type SearchService interface {
	Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error)
}

type searchManager struct {
	providers   map[string]SearchService
	config      *config.SearchsConfig
	aiProcessor *AIProcessor
}

// NewSearchManagerFromConfig 从新的配置系统创建搜索管理器
func NewSearchManagerFromConfig() SearchService {
	searchsCfg := config.GetSearchsConfig()
	if searchsCfg == nil {
		return &searchManager{
			providers: make(map[string]SearchService),
			config:    nil,
		}
	}

	m := &searchManager{
		providers: make(map[string]SearchService),
		config:    searchsCfg,
	}

	for name, pCfg := range searchsCfg.Providers {
		if !pCfg.Enabled {
			continue
		}
		switch name {
		case "baidu", "百度web_search":
			m.providers[name] = NewBaiduService(pCfg)
		case "serper":
			m.providers[name] = NewSerperService(pCfg)
		case "juhe":
			m.providers[name] = NewJuheService(pCfg)
		case "brightdata":
			m.providers[name] = NewBrightDataService(pCfg)
		}
	}

	return m
}

// NewSearchManager 从旧的配置系统创建搜索管理器（保留用于兼容性）
func NewSearchManager(cfg *config.SearchConfig) SearchService {
	return NewSearchManagerFromConfig()
}

func (m *searchManager) Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error) {
	if provider == "" && m.config != nil {
		provider = m.config.Settings.DefaultProvider
	}
	fmt.Printf("[SearchManager] Requested provider: %s\n", provider)

	// Try to get user-specific settings for the requested provider
	if userID != 0 {
		var settings []models.ProviderSetting
		database.DB.Where("user_id = ? AND type = ? AND provider = ? AND enabled = ?", userID, "search", provider, true).Limit(1).Find(&settings)

		if len(settings) > 0 && settings[0].APIKey != "" {
			userSetting := settings[0]
			fmt.Printf("[SearchManager] Using user-specific config for provider: %s\n", provider)

			// 从系统配置获取基础配置
			var baseCfg config.SearchProvider
			if sysCfg, ok := m.config.Providers[provider]; ok {
				baseCfg = sysCfg
			}

			// 使用用户配置覆盖
			pCfg := config.SearchProvider{
				Name:         baseCfg.Name,
				Enabled:      true,
				APIKey:       userSetting.APIKey,
				SecretKey:    baseCfg.SecretKey,
				SecretID:     baseCfg.SecretID,
				BaseURL:      userSetting.BaseURL,
				SearchMode:   baseCfg.SearchMode,
				Model:        baseCfg.Model,
				SearchSource: baseCfg.SearchSource,
			}
			if pCfg.BaseURL == "" {
				pCfg.BaseURL = baseCfg.BaseURL
			}

			switch provider {
			case "baidu", "百度web_search":
				return NewBaiduService(pCfg).Search(ctx, query, userID, provider)
			case "serper":
				return NewSerperService(pCfg).Search(ctx, query, userID, provider)
			case "juhe":
				return NewJuheService(pCfg).Search(ctx, query, userID, provider)
			case "brightdata":
				return NewBrightDataService(pCfg).Search(ctx, query, userID, provider)
			case "tencent":
				fmt.Println("[SearchManager] Tencent search requested, delegating to AI model")
				return []SearchResult{}, nil // Handled natively by AI
			}
		}
	}

	if provider == "tencent" {
		fmt.Println("[SearchManager] Tencent search requested. Note: This provider only works natively with Tencent Hunyuan models. Returning empty results for manual prompt augmentation.")
		return []SearchResult{}, nil
	}

	if p, ok := m.providers[provider]; ok {
		fmt.Printf("[SearchManager] Executing search via: %s\n", provider)
		return p.Search(ctx, query, userID, provider)
	}

	fmt.Printf("[SearchManager] Provider '%s' not found in active providers map, trying default\n", provider)
	// If the requested provider isn't available, try the default one
	defaultProvider := ""
	if m.config != nil {
		defaultProvider = m.config.Settings.DefaultProvider
	}
	if provider != defaultProvider && defaultProvider != "" {
		return m.Search(ctx, query, userID, defaultProvider)
	}

	return nil, fmt.Errorf("search provider '%s' not configured", provider)
}

// Baidu Search Implementation (支持四种搜索模式)
type baiduService struct {
	config config.SearchProvider
}

func NewBaiduService(cfg config.SearchProvider) SearchService {
	return &baiduService{config: cfg}
}

func (s *baiduService) Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error) {
	fmt.Printf("[Baidu Search] Query: %s\n", query)

	// 根据配置中的 BaseURL 确定搜索模式
	// baidu_web: 百度搜索
	// baidu_image: 相似图搜索
	// baidu_ai: 智能搜索
	// baidu_performance: 高性能搜索

	// 严格使用 searchs.yaml 中的配置，不做任何自动修改
	endpoint := s.config.BaseURL
	if endpoint == "" {
		endpoint = "https://qianfan.baidubce.com/v2/ai_search/chat/completions"
	}

	// 从 BaseURL 推断搜索模式（仅用于日志记录，不修改配置）
	searchMode := "web"
	if strings.Contains(s.config.BaseURL, "image") {
		searchMode = "image"
	} else if strings.Contains(s.config.BaseURL, "chat/completions") {
		searchMode = "ai"
	} else if strings.Contains(s.config.BaseURL, "web_summary") {
		searchMode = "performance"
	}

	// Prepare request body according to the provided cURL example
	payload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": query,
			},
		},
		"stream": false,
	}

	if searchMode == "ai" {
		// AI Search specific parameters
		// 使用配置中的 searchsource，如果没有则使用默认值
		searchSource := s.config.SearchSource
		if searchSource == "" {
			searchSource = "baidu_search_v2"
		}
		payload["search_source"] = searchSource
		payload["resource_type_filter"] = []map[string]interface{}{
			{"type": "image", "top_k": 4},
			{"type": "video", "top_k": 4},
			{"type": "web", "top_k": 4},
		}
		payload["search_recency_filter"] = "year"
		payload["model"] = "ernie-4.5-turbo-32k"
		payload["enable_deep_search"] = false
		payload["enable_followup_query"] = false
		payload["temperature"] = 0.11
		payload["top_p"] = 0.55
		payload["search_mode"] = "auto"
		payload["enable_reasoning"] = true
	} else if searchMode == "performance" {
		// Performance Search (Web Summary) - Minimal payload
		payload = map[string]interface{}{
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": query,
				},
			},
			"stream": false,
		}
	} else if searchMode == "image" {
		// Image Search (Visual Search)
		payload = map[string]interface{}{
			"image": query, // Assuming query contains image data or URL as requested by the specific endpoint
		}
	} else {
		// Standard Search (Web/Performance)
		// 使用配置中的 searchsource，如果没有则使用默认值
		searchSource := s.config.SearchSource
		if searchSource == "" {
			searchSource = "baidu_search_v2"
		}
		payload["search_source"] = searchSource
		payload["resource_type_filter"] = []map[string]interface{}{
			{
				"type":  searchMode,
				"top_k": 10,
			},
		}
	}

	jsonData, _ := json.Marshal(payload)

	fmt.Printf("[Baidu Search] Mode: %s, Endpoint: %s, Version: V2 (AppBuilder)\n", searchMode, endpoint)

	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Baidu Qianfan uses Authorization header
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("baidu search request failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("[Baidu Search] Raw Response: %s\n", string(bodyBytes))

	var result struct {
		Answer  string `json:"answer"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Results []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			URL     string `json:"url"`
		} `json:"results"`
		// Try alternative fields for V2
		SearchResults []struct {
			Title   string `json:"title"`
			Snippet string `json:"content"`
			URL     string `json:"url"`
		} `json:"search_results"`
		// User provided V2 format
		References []struct {
			Title   string `json:"title"`
			Snippet string `json:"content"`
			URL     string `json:"url"`
		} `json:"references"`
		// Image Search format
		ImageResult struct {
			ResData struct {
				ResItems []struct {
					Title  string `json:"title"`
					URL    string `json:"fromurl"`
					ObjURL string `json:"objurl"`
				} `json:"res_items"`
			} `json:"res_data"`
		} `json:"result"`
		ErrorCode int         `json:"error_code"`
		ErrorMsg  string      `json:"error_msg"`
		Code      interface{} `json:"code"`    // Image search uses "code" (can be string or int)
		Message   string      `json:"message"` // Image search uses "message"
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse baidu response: %v", err)
	}

	if result.ErrorCode != 0 {
		return nil, fmt.Errorf("baidu API error: %s (code: %d)", result.ErrorMsg, result.ErrorCode)
	}

	if result.Code != nil && result.Code != "" && result.Code != "0" && result.Code != 0 {
		return nil, fmt.Errorf("baidu Image search error: %s (code: %v)", result.Message, result.Code)
	}

	var results []SearchResult
	// Process standard results
	for _, r := range result.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			Snippet: r.Snippet,
			URL:     r.URL,
		})
	}

	// Process V2 search_results if results was empty
	if len(results) == 0 {
		for _, r := range result.SearchResults {
			results = append(results, SearchResult{
				Title:   r.Title,
				Snippet: r.Snippet,
				URL:     r.URL,
			})
		}
	}

	// Process Choices (AI Search Answer) if still empty
	if len(results) == 0 {
		for _, c := range result.Choices {
			if c.Message.Content != "" {
				results = append(results, SearchResult{
					Title:   "AI Search Answer",
					Snippet: c.Message.Content,
					URL:     endpoint,
				})
			}
		}
	}

	// Process References (new V2 format) if still empty
	if len(results) == 0 {
		for _, r := range result.References {
			results = append(results, SearchResult{
				Title:   r.Title,
				Snippet: r.Snippet,
				URL:     r.URL,
			})
		}
	}

	// Process Image Search results
	if len(results) == 0 {
		for _, item := range result.ImageResult.ResData.ResItems {
			results = append(results, SearchResult{
				Title:   item.Title,
				Snippet: fmt.Sprintf("Image Source: %s | Image URL: %s", item.URL, item.ObjURL),
				URL:     item.URL,
			})
		}
	}

	// If still empty but we have an answer, use the answer as a single snippet
	if len(results) == 0 && result.Answer != "" {
		results = append(results, SearchResult{
			Title:   "Baidu AI Answer",
			Snippet: result.Answer,
			URL:     endpoint,
		})
	}

	fmt.Printf("[Baidu Search] Found %d results\n", len(results))
	return results, nil
}

// Serper Implementation
type serperService struct {
	config config.SearchProvider
}

func NewSerperService(cfg config.SearchProvider) SearchService {
	return &serperService{config: cfg}
}

func (s *serperService) Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error) {
	payload := map[string]string{"q": query}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL, bytes.NewBuffer(jsonData))
	req.Header.Set("X-API-KEY", s.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Organic []SearchResult `json:"organic"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Organic, nil
}

// Juhe AI Search Implementation
type juheService struct {
	config config.SearchProvider
}

func NewJuheService(cfg config.SearchProvider) SearchService {
	return &juheService{config: cfg}
}

func (s *juheService) Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error) {
	// Juhe API usually uses GET or POST with key param
	apiURL := fmt.Sprintf("%s?key=%s&q=%s", s.config.BaseURL, s.config.APIKey, query)

	req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Reason string `json:"reason"`
		Result []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			URL     string `json:"url"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, r := range result.Result {
		results = append(results, SearchResult{
			Title:   r.Title,
			Snippet: r.Snippet,
			URL:     r.URL,
		})
	}

	return results, nil
}

// BrightData Implementation
type brightDataService struct {
	config config.SearchProvider
}

func NewBrightDataService(cfg config.SearchProvider) SearchService {
	return &brightDataService{config: cfg}
}

func (s *brightDataService) Search(ctx context.Context, query string, userID uint, provider string) ([]SearchResult, error) {
	// BrightData SERP API
	if s.config.BaseURL == "" {
		s.config.BaseURL = "https://brd.superproxy.io:22225"
	}

	payload := map[string]string{"q": query}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL, bytes.NewBuffer(jsonData))
	// BrightData usually uses Proxy-Authorization or similar
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Organic []SearchResult `json:"organic_results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Organic, nil
}
