package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// SearchsConfig 搜索引擎配置
type SearchsConfig struct {
	Providers map[string]SearchProvider `mapstructure:"providers"`
	Settings  SearchSettings            `mapstructure:"settings"`
}

type ResourceTypeFilter struct {
	Type string `mapstructure:"type" json:"type"`
	TopK int    `mapstructure:"topk" json:"top_k"`
}

type SearchProvider struct {
	Name                string               `mapstructure:"name" json:"name"`
	Enabled             bool                 `mapstructure:"enabled" json:"enabled"`
	APIKey              string               `mapstructure:"apikey" json:"api_key,omitempty"`
	SecretKey           string               `mapstructure:"secretkey" json:"secret_key,omitempty"`
	SecretID            string               `mapstructure:"secretid" json:"secret_id,omitempty"`
	BaseURL             string               `mapstructure:"baseurl" json:"base_url"`
	SearchMode          string               `mapstructure:"searchmode" json:"search_mode,omitempty"`
	Model               string               `mapstructure:"model" json:"model,omitempty"`
	EnableDeepSearch    bool                 `mapstructure:"enabledeepsearch" json:"enable_deep_search,omitempty"`
	EnableReasoning     bool                 `mapstructure:"enablereasoning" json:"enable_reasoning,omitempty"`
	ResponseFormat      string               `mapstructure:"responseformat" json:"response_format,omitempty"`
	Stream              bool                 `mapstructure:"stream" json:"stream,omitempty"`
	Instruction         string               `mapstructure:"instruction" json:"instruction,omitempty"`
	SearchSource        string               `mapstructure:"searchsource" json:"search_source,omitempty"`
	ResourceTypeFilter  []ResourceTypeFilter `mapstructure:"resourcetypefilter" json:"resource_type_filter,omitempty"`
	SearchRecencyFilter string               `mapstructure:"searchrecencyfilter" json:"search_recency_filter,omitempty"`
	EnableCornerMarkers bool                 `mapstructure:"enablecornermarkers" json:"enable_corner_markers,omitempty"`
	EnableFollowupQuery bool                 `mapstructure:"enablefollowupquery" json:"enable_followup_query,omitempty"`
	Temperature         float64              `mapstructure:"temperature" json:"temperature,omitempty"`
	TopP                float64              `mapstructure:"topp" json:"top_p,omitempty"`
}

type SearchSettings struct {
	DefaultProvider string `mapstructure:"defaultprovider"`
	Timeout         int    `mapstructure:"timeout"`
	MaxResults      int    `mapstructure:"maxresults"`
	SafeSearch      bool   `mapstructure:"safesearch"`
}

var (
	searchsConfig     *SearchsConfig
	searchsConfigPath string
	searchsMu         sync.RWMutex
)

// LoadSearchsConfig 加载搜索配置
func LoadSearchsConfig(path string) (*SearchsConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg SearchsConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	searchsMu.Lock()
	searchsConfig = &cfg
	searchsConfigPath = path
	searchsMu.Unlock()

	log.Printf("Loaded searchs config from %s, providers: %d", path, len(cfg.Providers))
	return &cfg, nil
}

// GetSearchsConfig 获取搜索配置
func GetSearchsConfig() *SearchsConfig {
	searchsMu.RLock()
	defer searchsMu.RUnlock()
	return searchsConfig
}

// UpdateSearchProviderConfig 更新搜索提供商
func UpdateSearchProviderConfig(name string, provider SearchProvider) {
	searchsMu.Lock()
	defer searchsMu.Unlock()

	if searchsConfig == nil {
		searchsConfig = &SearchsConfig{
			Providers: make(map[string]SearchProvider),
		}
	}

	searchsConfig.Providers[name] = provider

	v := viper.New()
	v.SetConfigFile(searchsConfigPath)
	v.Set("providers", searchsConfig.Providers)
	v.Set("settings", searchsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save searchs config: %v", err)
	}
}

// DeleteSearchProviderConfig 删除搜索提供商
func DeleteSearchProviderConfig(name string) {
	searchsMu.Lock()
	defer searchsMu.Unlock()

	if searchsConfig == nil || searchsConfig.Providers == nil {
		return
	}
	delete(searchsConfig.Providers, name)

	v := viper.New()
	v.SetConfigFile(searchsConfigPath)
	v.Set("providers", searchsConfig.Providers)
	v.Set("settings", searchsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save searchs config: %v", err)
	}
}
