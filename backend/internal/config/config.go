package config

import (
	"log"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	AI       AIConfig       `mapstructure:"ai"`
	Search   SearchConfig   `mapstructure:"search"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Cache    CacheConfig    `mapstructure:"cache"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	OS       string         `mapstructure:"os"` // Detected operating system
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type SearchConfig struct {
	DefaultProvider string                          `mapstructure:"default_provider"`
	Providers       map[string]SearchProviderConfig `mapstructure:"providers"`
}

type SearchProviderConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	APIKey       string `mapstructure:"api_key"`
	BaseURL      string `mapstructure:"base_url"`
	SearchSource string `mapstructure:"searchsource"`
}

type ServerConfig struct {
	Port        int      `mapstructure:"port"`
	Mode        string   `mapstructure:"mode"`
	CorsOrigins []string `mapstructure:"cors_origins"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

type PostgresConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	DBName         string `mapstructure:"dbname"`
	SSLMode        string `mapstructure:"sslmode"`
	MaxConnections int    `mapstructure:"max_connections"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AIConfig struct {
	Providers map[string]AIProviderConfig `mapstructure:"providers"`
}

type AIProviderConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
	SecretID  string `mapstructure:"secret_id"`
	BaseURL   string `mapstructure:"base_url"`
}

type StorageConfig struct {
	UploadDir    string   `mapstructure:"upload_dir"`
	MaxSize      int64    `mapstructure:"max_size"`
	AllowedTypes []string `mapstructure:"allowed_types"`
}

type CacheConfig struct {
	TTL map[string]int `mapstructure:"ttl"`
}

var GlobalConfig *Config
var configPath string

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Detect operating system (this will override any config file setting)
	config.OS = runtime.GOOS

	GlobalConfig = &config
	configPath = path
	return GlobalConfig, nil
}

// SaveConfig saves the current configuration to the config file
func SaveConfig() error {
	if GlobalConfig == nil {
		log.Println("SaveConfig: GlobalConfig is nil")
		return nil
	}

	if configPath == "" {
		log.Println("SaveConfig: configPath is empty")
		return nil
	}

	log.Printf("SaveConfig: Attempting to save to path: %s", configPath)

	// Sync GlobalConfig to viper before saving
	syncConfigToViper()

	// Try to write with WriteConfig (overwrites if exists)
	log.Println("SaveConfig: Attempting to write config using viper.WriteConfig()")
	err := viper.WriteConfig()
	if err != nil {
		// If WriteConfig fails, try WriteConfigAs to the same path
		log.Printf("SaveConfig: viper.WriteConfig() failed: %v", err)
		log.Printf("SaveConfig: Attempting to write with viper.WriteConfigAs(%s)", configPath)
		err = viper.WriteConfigAs(configPath)
	}

	if err != nil {
		log.Printf("SaveConfig: All attempts to write config failed: %v", err)
		return err
	}

	log.Println("SaveConfig: Config file saved successfully")
	return nil
}

// syncConfigToViper syncs GlobalConfig to viper's internal state
func syncConfigToViper() {
	if GlobalConfig == nil {
		return
	}

	// Sync AI providers
	if GlobalConfig.AI.Providers != nil {
		viper.Set("ai.providers", GlobalConfig.AI.Providers)
	}

	// Sync Search providers
	if GlobalConfig.Search.Providers != nil {
		viper.Set("search.providers", GlobalConfig.Search.Providers)
	}

	// Sync other config sections
	viper.Set("server", GlobalConfig.Server)
	viper.Set("database", GlobalConfig.Database)
	viper.Set("jwt", GlobalConfig.JWT)
	viper.Set("cache", GlobalConfig.Cache)
	viper.Set("storage", GlobalConfig.Storage)
}

// UpdateAIProvider updates an AI provider configuration
func UpdateAIProvider(name string, settings AIProviderConfig) {
	if GlobalConfig == nil {
		return
	}
	if GlobalConfig.AI.Providers == nil {
		GlobalConfig.AI.Providers = make(map[string]AIProviderConfig)
	}
	GlobalConfig.AI.Providers[name] = settings
	// Also update viper configuration - use map to ensure proper serialization
	viper.Set("ai.providers."+name+".enabled", settings.Enabled)
	viper.Set("ai.providers."+name+".api_key", settings.APIKey)
	viper.Set("ai.providers."+name+".secret_key", settings.SecretKey)
	viper.Set("ai.providers."+name+".secret_id", settings.SecretID)
	viper.Set("ai.providers."+name+".base_url", settings.BaseURL)
	log.Printf("UpdateAIProvider: Updated provider %s in config", name)
}

// UpdateSearchProvider updates a search provider configuration
func UpdateSearchProvider(name string, settings SearchProviderConfig) {
	if GlobalConfig == nil {
		return
	}
	if GlobalConfig.Search.Providers == nil {
		GlobalConfig.Search.Providers = make(map[string]SearchProviderConfig)
	}
	GlobalConfig.Search.Providers[name] = settings
	// Also update viper configuration - use map to ensure proper serialization
	viper.Set("search.providers."+name+".enabled", settings.Enabled)
	viper.Set("search.providers."+name+".api_key", settings.APIKey)
	viper.Set("search.providers."+name+".base_url", settings.BaseURL)
	log.Printf("UpdateSearchProvider: Updated provider %s in config", name)
}

// DeleteAIProvider deletes an AI provider configuration
func DeleteAIProvider(name string) {
	if GlobalConfig == nil || GlobalConfig.AI.Providers == nil {
		return
	}
	delete(GlobalConfig.AI.Providers, name)
	// Also delete from viper configuration
	viper.Set("ai.providers."+name, nil)
	log.Printf("DeleteAIProvider: Deleted provider %s from config", name)
}

// DeleteSearchProvider deletes a search provider configuration
func DeleteSearchProvider(name string) {
	if GlobalConfig == nil || GlobalConfig.Search.Providers == nil {
		return
	}
	delete(GlobalConfig.Search.Providers, name)
	// Also delete from viper configuration
	viper.Set("search.providers."+name, nil)
	log.Printf("DeleteSearchProvider: Deleted provider %s from config", name)
}
