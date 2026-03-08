package config

import (
	"log"
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

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 使用自定义的解码配置，支持多种字段命名格式
	var cfg ModelsConfig
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           &cfg,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			// 自定义钩子：处理字段名大小写和下划线差异
			normalizeFieldNamesHook(),
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(v.AllSettings()); err != nil {
		return nil, err
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
	modelsConfig.Providers[name] = provider

	// 保存到文件
	v := viper.New()
	v.SetConfigFile(modelsConfigPath)
	v.Set("providers", modelsConfig.Providers)
	v.Set("settings", modelsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save models config: %v", err)
	}
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
