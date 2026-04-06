package config

import (
	"log"
	"path/filepath"
	"strings"
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
	SetupViperEnv(v, "SEARCHS")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg SearchsConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 从环境变量中读取API密钥
	for key, provider := range cfg.Providers {
		// 构建环境变量键名
		prefix := "SEARCHS_PROVIDERS_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		
		// 从环境变量中读取API密钥
		if apiKey := GetEnvWithDefault(prefix+"_APIKEY", ""); apiKey != "" {
			provider.APIKey = apiKey
		}
		if secretKey := GetEnvWithDefault(prefix+"_SECRETKEY", ""); secretKey != "" {
			provider.SecretKey = secretKey
		}
		if secretID := GetEnvWithDefault(prefix+"_SECRETID", ""); secretID != "" {
			provider.SecretID = secretID
		}
		
		// 更新provider
		cfg.Providers[key] = provider
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
	
	// 提取API密钥并保存到.env文件
	configDir := filepath.Dir(searchsConfigPath)
	keyValuePairs := make(map[string]string)
	
	// 构建环境变量键名
	prefix := "SEARCHS_PROVIDERS_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	
	// 保存API密钥到.env
	if provider.APIKey != "" {
		keyValuePairs[prefix+"_APIKEY"] = provider.APIKey
	}
	if provider.SecretKey != "" {
		keyValuePairs[prefix+"_SECRETKEY"] = provider.SecretKey
	}
	if provider.SecretID != "" {
		keyValuePairs[prefix+"_SECRETID"] = provider.SecretID
	}
	
	// 更新.env文件
	if len(keyValuePairs) > 0 {
		if err := UpdateEnvFile(configDir, keyValuePairs); err != nil {
			log.Printf("Failed to update .env file: %v", err)
		}
	}
	
	// 清空敏感信息后再保存到searchs.yaml
	sanitizedProvider := provider
	sanitizedProvider.APIKey = ""
	sanitizedProvider.SecretKey = ""
	sanitizedProvider.SecretID = ""
	
	// 保存到searchs.yaml
	v := viper.New()
	v.SetConfigFile(searchsConfigPath)
	
	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Failed to read searchs config: %v", err)
	}
	
	// 更新指定提供商的配置，使用sanitizedProvider
	providers := v.Get("providers").(map[string]interface{})
	providers[name] = sanitizedProvider
	v.Set("providers", providers)
	
	// 保存到文件
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save searchs config: %v", err)
	}
	
	// 更新内存中的配置
	searchsConfig.Providers[name] = sanitizedProvider
	
	// 重新加载配置，确保从环境变量中读取API密钥
	if _, err := LoadSearchsConfig(searchsConfigPath); err != nil {
		log.Printf("Failed to reload searchs config: %v", err)
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
