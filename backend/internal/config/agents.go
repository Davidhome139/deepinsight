package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// AgentsConfig 智能体配置
type AgentsConfig struct {
	Agents   map[string]Agent `mapstructure:"agents"`
	Settings AgentSettings    `mapstructure:"settings"`
}

type Agent struct {
	Name         string `mapstructure:"name"`
	Enabled      bool   `mapstructure:"enabled"`
	Description  string `mapstructure:"description"`
	Role         string `mapstructure:"role"`
	Model        string `mapstructure:"model"`
	SystemPrompt string `mapstructure:"system_prompt"`
}

type AgentSettings struct {
	DefaultAgent      string `mapstructure:"default_agent"`
	MaxIterations     int    `mapstructure:"max_iterations"`
	AutoConfirm       bool   `mapstructure:"auto_confirm"`
	ParallelExecution bool   `mapstructure:"parallel_execution"`
}

var (
	agentsConfig     *AgentsConfig
	agentsConfigPath string
	agentsMu         sync.RWMutex
)

// LoadAgentsConfig 加载智能体配置
func LoadAgentsConfig(path string) (*AgentsConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg AgentsConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	agentsMu.Lock()
	agentsConfig = &cfg
	agentsConfigPath = path
	agentsMu.Unlock()

	log.Printf("Loaded agents config from %s, agents: %d", path, len(cfg.Agents))
	return &cfg, nil
}

// GetAgentsConfig 获取智能体配置
func GetAgentsConfig() *AgentsConfig {
	agentsMu.RLock()
	defer agentsMu.RUnlock()
	return agentsConfig
}

// UpdateAgent 更新智能体
func UpdateAgent(name string, agent Agent) {
	agentsMu.Lock()
	defer agentsMu.Unlock()

	if agentsConfig == nil {
		agentsConfig = &AgentsConfig{
			Agents: make(map[string]Agent),
		}
	}
	agentsConfig.Agents[name] = agent

	v := viper.New()
	v.SetConfigFile(agentsConfigPath)
	v.Set("agents", agentsConfig.Agents)
	v.Set("settings", agentsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save agents config: %v", err)
	}
}

// DeleteAgent 删除智能体
func DeleteAgent(name string) {
	agentsMu.Lock()
	defer agentsMu.Unlock()

	if agentsConfig == nil || agentsConfig.Agents == nil {
		return
	}
	delete(agentsConfig.Agents, name)

	v := viper.New()
	v.SetConfigFile(agentsConfigPath)
	v.Set("agents", agentsConfig.Agents)
	v.Set("settings", agentsConfig.Settings)
	if err := v.WriteConfig(); err != nil {
		log.Printf("Failed to save agents config: %v", err)
	}
}
