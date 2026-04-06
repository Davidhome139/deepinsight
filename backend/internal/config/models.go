package config

import (
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// ModelsConfig AI 模型配置
type ModelsConfig struct {
	Providers map[string]ModelProvider `mapstructure:"providers"`
	Settings  ModelSettings            `mapstructure:"settings"`
}

type ModelProvider struct {
	Name                string            `mapstructure:"name" json:"name"`
	Enabled             bool              `mapstructure:"enabled" json:"enabled"`
	APIKey              string            `mapstructure:"apikey" json:"api_key"`
	SecretKey           string            `mapstructure:"secretkey" json:"secret_key,omitempty"`
	SecretID            string            `mapstructure:"secretid" json:"secret_id,omitempty"`
	GroupID             string            `mapstructure:"groupid" json:"group_id,omitempty"`
	BaseURL             string            `mapstructure:"baseurl" json:"base_url"`
	Endpoint            string            `mapstructure:"endpoint" json:"endpoint"`
	AuthType            string            `mapstructure:"authtype" json:"auth_type"`
	TokenURL            string            `mapstructure:"tokenurl" json:"token_url,omitempty"`
	Headers             map[string]string `mapstructure:"headers" json:"headers,omitempty"`
	Timeout             int               `mapstructure:"timeout" json:"timeout"`
	Stream              bool              `mapstructure:"stream" json:"stream,omitempty"`
	MaxCompletionTokens int64             `mapstructure:"maxcompletiontokens" json:"max_completion_tokens,omitempty"`
	Temperature         float64           `mapstructure:"temperature" json:"temperature,omitempty"`
	TopP                float64           `mapstructure:"topp" json:"top_p,omitempty"`
	EnableThinking      bool              `mapstructure:"enablethinking" json:"enable_thinking,omitempty"`
	Models              []string          `mapstructure:"models" json:"models"`
}

type ModelSettings struct {
	DefaultModel string  `mapstructure:"defaultmodel" json:"default_model"`
	Timeout      int     `mapstructure:"timeout" json:"timeout"`
	MaxTokens    int     `mapstructure:"maxtokens" json:"max_tokens"`
	Temperature  float64 `mapstructure:"temperature" json:"temperature"`
	Stream       bool    `mapstructure:"stream" json:"stream"`
	RetryTimes   int     `mapstructure:"retrytimes" json:"retry_times"`
	RetryDelay   int     `mapstructure:"retrydelay" json:"retry_delay"`
}

var (
	modelsConfig     *ModelsConfig
	modelsConfigPath string
	modelsMu         sync.RWMutex
)

// LoadModelsConfig 加载模型配置（支持多种字段命名格式）
func LoadModelsConfig(path string) (*ModelsConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	SetupViperEnv(v, "MODELS")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 使用viper的Unmarshal方法来解码配置，这样会自动从环境变量中加载配置
	var cfg ModelsConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 从环境变量中读取API密钥
	for key, provider := range cfg.Providers {
		// 构建环境变量键名
		prefix := "MODELS_PROVIDERS_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		
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
		if groupID := GetEnvWithDefault(prefix+"_GROUPID", ""); groupID != "" {
			provider.GroupID = groupID
		}
		
		// 更新provider
		cfg.Providers[key] = provider
	}

	modelsMu.Lock()
	modelsConfig = &cfg
	modelsConfigPath = path
	modelsMu.Unlock()

	log.Printf("Loaded models config from %s, providers: %d", path, len(cfg.Providers))
	return &cfg, nil
}

// normalizeFieldNamesHook 创建字段名归一化钩子
func normalizeFieldNamesHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		// 只处理 map 类型
		if from.Kind() != reflect.Map || to.Kind() != reflect.Map {
			return data, nil
		}

		// 如果键是字符串类型，进行归一化
		if from.Key().Kind() == reflect.String {
			result := make(map[string]interface{})
			inputMap, ok := data.(map[string]interface{})
			if !ok {
				return data, nil
			}

			for key, value := range inputMap {
				// 归一化键名：转为小写并移除下划线
				normalizedKey := normalizeKey(key)
				result[normalizedKey] = value
			}
			return result, nil
		}

		return data, nil
	}
}

// normalizeKey 归一化键名：转为小写并统一处理下划线
func normalizeKey(key string) string {
	// 转为小写
	key = strings.ToLower(key)
	// 移除所有下划线，使 api_key 和 apikey 等价
	key = strings.ReplaceAll(key, "_", "")
	return key
}

// GetModelsConfig 获取模型配置
func GetModelsConfig() *ModelsConfig {
	modelsMu.RLock()
	defer modelsMu.RUnlock()
	return modelsConfig
}

// UpdateModelProvider 更新模型提供商
func UpdateModelProvider(name string, provider ModelProvider) {
	modelsMu.Lock()
	defer modelsMu.Unlock()

	if modelsConfig == nil {
		modelsConfig = &ModelsConfig{
			Providers: make(map[string]ModelProvider),
		}
	}
	
	// 提取API密钥并保存到.env文件
	configDir := filepath.Dir(modelsConfigPath)
	keyValuePairs := make(map[string]string)
	
	// 构建环境变量键名
	prefix := "MODELS_PROVIDERS_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	
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
	if provider.GroupID != "" {
		keyValuePairs[prefix+"_GROUPID"] = provider.GroupID
	}
	
	// 更新.env文件
	if len(keyValuePairs) > 0 {
		if err := UpdateEnvFile(configDir, keyValuePairs); err != nil {
			log.Printf("Failed to update .env file: %v", err)
		}
	}
	
	// 清空敏感信息后再保存到models.yaml
	sanitizedProvider := provider
	sanitizedProvider.APIKey = ""
	sanitizedProvider.SecretKey = ""
	sanitizedProvider.SecretID = ""
	sanitizedProvider.GroupID = ""
	
	// 保存到models.yaml
	v := viper.New()
	v.SetConfigFile(modelsConfigPath)
	
	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Failed to read models config: %v", err)
	}
	
	// 更新指定提供商的配置，使用sanitizedProvider
	providers := v.Get("providers").(map[string]interface{})
	providers[name] = sanitizedProvider
	v.Set("providers", providers)
	
	// 保存到文件
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save models config: %v", err)
	}
	
	// 更新内存中的配置，直接从环境变量中读取API密钥
	updatedProvider := sanitizedProvider
	// 从环境变量中读取API密钥
	if apiKey := GetEnvWithDefault(prefix+"_APIKEY", ""); apiKey != "" {
		updatedProvider.APIKey = apiKey
	}
	if secretKey := GetEnvWithDefault(prefix+"_SECRETKEY", ""); secretKey != "" {
		updatedProvider.SecretKey = secretKey
	}
	if secretID := GetEnvWithDefault(prefix+"_SECRETID", ""); secretID != "" {
		updatedProvider.SecretID = secretID
	}
	if groupID := GetEnvWithDefault(prefix+"_GROUPID", ""); groupID != "" {
		updatedProvider.GroupID = groupID
	}
	
	// 更新内存中的配置
	modelsConfig.Providers[name] = updatedProvider
	
	log.Printf("Updated model provider: %s, API key: %s", name, updatedProvider.APIKey)
}

// DeleteModelProvider 删除模型提供商
func DeleteModelProvider(name string) {
	modelsMu.Lock()
	defer modelsMu.Unlock()

	if modelsConfig == nil || modelsConfig.Providers == nil {
		return
	}
	delete(modelsConfig.Providers, name)

	// 保存到文件
	v := viper.New()
	v.SetConfigFile(modelsConfigPath)
	v.Set("providers", modelsConfig.Providers)
	v.Set("settings", modelsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save models config: %v", err)
	}
}
