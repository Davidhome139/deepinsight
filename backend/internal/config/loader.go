package config

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
)

// ConfigManager 统一管理所有配置
type ConfigManager struct {
	ConfigDir string
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configDir string) *ConfigManager {
	return &ConfigManager{
		ConfigDir: configDir,
	}
}

// LoadAll 加载所有配置文件
func (cm *ConfigManager) LoadAll() error {
	log.Printf("Loading configs from directory: %s", cm.ConfigDir)

	// 加载主配置
	mainConfigPath := filepath.Join(cm.ConfigDir, "config.yaml")
	log.Printf("Loading main config from: %s", mainConfigPath)
	if _, err := LoadConfig(mainConfigPath); err != nil {
		log.Printf("Warning: Failed to load main config: %v", err)
	} else {
		log.Printf("Successfully loaded main config")
	}

	// 加载模型配置
	modelsPath := filepath.Join(cm.ConfigDir, "models.yaml")
	log.Printf("Loading models config from: %s", modelsPath)
	if cfg, err := LoadModelsConfig(modelsPath); err != nil {
		log.Printf("Warning: Failed to load models config: %v", err)
	} else {
		log.Printf("Successfully loaded models config with %d providers", len(cfg.Providers))
	}

	// 加载搜索配置
	searchsPath := filepath.Join(cm.ConfigDir, "searchs.yaml")
	log.Printf("Loading searchs config from: %s", searchsPath)
	if cfg, err := LoadSearchsConfig(searchsPath); err != nil {
		log.Printf("Warning: Failed to load searchs config: %v", err)
	} else {
		log.Printf("Successfully loaded searchs config with %d providers", len(cfg.Providers))
	}

	// 加载 MCP 服务器配置
	mcpsPath := filepath.Join(cm.ConfigDir, "mcpservers.yaml")
	log.Printf("Loading MCP servers config from: %s", mcpsPath)
	if cfg, err := LoadMCPServersConfig(mcpsPath); err != nil {
		log.Printf("Warning: Failed to load MCP servers config: %v", err)
	} else {
		log.Printf("Successfully loaded MCP servers config with %d servers", len(cfg.Servers))
	}

	// 加载技能配置
	skillsPath := filepath.Join(cm.ConfigDir, "skills.yaml")
	log.Printf("Loading skills config from: %s", skillsPath)
	if cfg, err := LoadSkillsConfig(skillsPath); err != nil {
		log.Printf("Warning: Failed to load skills config: %v", err)
	} else {
		log.Printf("Successfully loaded skills config with %d sources", len(cfg.Sources))
	}

	// 加载智能体配置
	agentsPath := filepath.Join(cm.ConfigDir, "agents.yaml")
	log.Printf("Loading agents config from: %s", agentsPath)
	if cfg, err := LoadAgentsConfig(agentsPath); err != nil {
		log.Printf("Warning: Failed to load agents config: %v", err)
	} else {
		log.Printf("Successfully loaded agents config with %d agents", len(cfg.Agents))
	}

	log.Println("All configurations loaded")
	return nil
}

// GetAllSettings 获取所有设置（用于前端展示）
func (cm *ConfigManager) GetAllSettings() map[string]interface{} {
	result := make(map[string]interface{})

	// AI 模型
	if models := GetModelsConfig(); models != nil {
		modelList := make([]map[string]interface{}, 0)

		// 先收集所有的 key 并排序
		keys := make([]string, 0, len(models.Providers))
		for key := range models.Providers {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// 按排序后的 key 构建列表
		for _, key := range keys {
			provider := models.Providers[key]
			// 构建返回给前端的配置项
			item := map[string]interface{}{
				"key":       key,
				"name":      provider.Name,
				"enabled":   provider.Enabled,
				"api_key":   provider.APIKey,
				"base_url":  provider.BaseURL,
				"endpoint":  provider.Endpoint,
				"auth_type": provider.AuthType,
				"timeout":   provider.Timeout,
				"models":    provider.Models,
				"type":      "models",
			}
			// 根据不同提供商决定哪些字段始终显示
			// 腾讯混元需要 secret_id 和 secret_key
			if key == "tencent" {
				item["secret_id"] = provider.SecretID
				item["secret_key"] = provider.SecretKey
			}
			// 腾讯云 API 需要 secret_id 和 secret_key
			if key == "tencentcloud" {
				item["secret_id"] = provider.SecretID
				item["secret_key"] = provider.SecretKey
			}
			// MiniMax 需要 group_id
			if key == "minimax" {
				item["group_id"] = provider.GroupID
			}

			// 阿里云 Qwen 需要 enable_thinking
			if key == "aliyun" {
				item["enable_thinking"] = provider.EnableThinking
			}

			// 添加可选字段（如果有值）
			if provider.Stream {
				item["stream"] = provider.Stream
			}
			if provider.MaxCompletionTokens > 0 {
				item["max_completion_tokens"] = provider.MaxCompletionTokens
			}
			if provider.Temperature > 0 {
				item["temperature"] = provider.Temperature
			}
			if provider.TopP > 0 {
				item["top_p"] = provider.TopP
			}
			// 其他可选字段（非必需的认证字段）
			if key != "tencent" && key != "tencentcloud" && provider.SecretKey != "" {
				item["secret_key"] = provider.SecretKey
			}
			if key != "tencent" && key != "tencentcloud" && provider.SecretID != "" {
				item["secret_id"] = provider.SecretID
			}
			if key != "minimax" && provider.GroupID != "" {
				item["group_id"] = provider.GroupID
			}
			if provider.TokenURL != "" {
				item["token_url"] = provider.TokenURL
			}
			if len(provider.Headers) > 0 {
				item["headers"] = provider.Headers
			}
			modelList = append(modelList, item)
		}
		result["models"] = modelList
	}

	// 搜索引擎
	if searchs := GetSearchsConfig(); searchs != nil {
		searchList := make([]map[string]interface{}, 0)

		// 排序
		keys := make([]string, 0, len(searchs.Providers))
		for key := range searchs.Providers {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			provider := searchs.Providers[key]
			item := map[string]interface{}{
				"key":                   key,
				"name":                  provider.Name,
				"enabled":               provider.Enabled,
				"api_key":               provider.APIKey,
				"secret_key":            provider.SecretKey,
				"secret_id":             provider.SecretID,
				"base_url":              provider.BaseURL,
				"search_mode":           provider.SearchMode,
				"model":                 provider.Model,
				"search_source":         provider.SearchSource,
				"resource_type_filter":  provider.ResourceTypeFilter,
				"search_recency_filter": provider.SearchRecencyFilter,
				"stream":                provider.Stream,
				"enable_deep_search":    provider.EnableDeepSearch,
				"enable_reasoning":      provider.EnableReasoning,
				"enable_followup_query": provider.EnableFollowupQuery,
				"enable_corner_markers": provider.EnableCornerMarkers,
				"response_format":       provider.ResponseFormat,
				"instruction":           provider.Instruction,
				"temperature":           provider.Temperature,
				"top_p":                 provider.TopP,
				"type":                  "searchs",
			}

			searchList = append(searchList, item)
		}
		result["searchs"] = searchList
	}

	// MCP 服务器
	if mcps := GetMCPServersConfig(); mcps != nil {
		mcpList := make([]map[string]interface{}, 0)

		// 排序
		keys := make([]string, 0, len(mcps.Servers))
		for key := range mcps.Servers {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			server := mcps.Servers[key]
			mcpList = append(mcpList, map[string]interface{}{
				"key":           key,
				"name":          server.Name,
				"enabled":       server.Enabled,
				"type":          "mcpservers",
				"server_type":   server.Type,
				"command":       server.Command,
				"args":          server.Args,
				"env":           server.Env,
				"allowed_paths": server.AllowedPaths,
			})
		}
		result["mcpservers"] = mcpList
	}

	// 技能源
	if skills := GetSkillsConfig(); skills != nil {
		skillList := make([]map[string]interface{}, 0)

		// 排序
		keys := make([]string, 0, len(skills.Sources))
		for key := range skills.Sources {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			source := skills.Sources[key]
			skillList = append(skillList, map[string]interface{}{
				"key":             key,
				"name":            key,
				"enabled":         source.Enabled,
				"source_type":     source.Type,
				"path":            source.Path,
				"repo":            source.Repo,
				"ignored_paths":   source.IgnoredPaths,
				"disabled_skills": source.DisabledSkills,
				"enabled_skills":  source.EnabledSkills,
				"type":            "skill",
			})
		}
		result["skills"] = skillList
	}

	// 智能体
	if agents := GetAgentsConfig(); agents != nil {
		agentList := make([]map[string]interface{}, 0)

		// 排序
		keys := make([]string, 0, len(agents.Agents))
		for key := range agents.Agents {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			agent := agents.Agents[key]
			agentList = append(agentList, map[string]interface{}{
				"key":           key,
				"name":          agent.Name,
				"enabled":       agent.Enabled,
				"description":   agent.Description,
				"role":          agent.Role,
				"model":         agent.Model,
				"system_prompt": agent.SystemPrompt,
				"type":          "agent",
			})
		}
		result["agents"] = agentList
	}

	return result
}

// GetSettingByType 根据类型获取设置
func (cm *ConfigManager) GetSettingByType(settingType string) ([]map[string]interface{}, error) {
	switch settingType {
	case "models":
		return cm.getModelsList(), nil
	case "searchs":
		return cm.getSearchsList(), nil
	case "mcpservers":
		return cm.getMCPServersList(), nil
	case "skills":
		return cm.getSkillsList(), nil
	case "agents":
		return cm.getAgentsList(), nil
	default:
		return nil, fmt.Errorf("unknown setting type: %s", settingType)
	}
}

func (cm *ConfigManager) getModelsList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	if models := GetModelsConfig(); models != nil {
		for key, provider := range models.Providers {
			list = append(list, map[string]interface{}{
				"key":        key,
				"name":       provider.Name,
				"enabled":    provider.Enabled,
				"api_key":    provider.APIKey,
				"secret_key": provider.SecretKey,
				"secret_id":  provider.SecretID,
				"base_url":   provider.BaseURL,
				"models":     provider.Models,
				"type":       "models",
			})
		}
	}
	return list
}

func (cm *ConfigManager) getSearchsList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	if searchs := GetSearchsConfig(); searchs != nil {
		for key, provider := range searchs.Providers {
			list = append(list, map[string]interface{}{
				"key":         key,
				"name":        provider.Name,
				"enabled":     provider.Enabled,
				"api_key":     provider.APIKey,
				"secret_key":  provider.SecretKey,
				"secret_id":   provider.SecretID,
				"base_url":    provider.BaseURL,
				"search_mode": provider.SearchMode,
				"type":        "searchs",
			})
		}
	}
	return list
}

func (cm *ConfigManager) getMCPServersList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	if mcps := GetMCPServersConfig(); mcps != nil {
		for key, server := range mcps.Servers {
			list = append(list, map[string]interface{}{
				"key":           key,
				"name":          server.Name,
				"enabled":       server.Enabled,
				"server_type":   server.Type,
				"command":       server.Command,
				"args":          server.Args,
				"env":           server.Env,
				"allowed_paths": server.AllowedPaths,
				"type":          "mcpservers",
			})
		}
	}
	return list
}

func (cm *ConfigManager) getSkillsList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	if skills := GetSkillsConfig(); skills != nil {
		for key, source := range skills.Sources {
			list = append(list, map[string]interface{}{
				"key":             key,
				"name":            key,
				"enabled":         source.Enabled,
				"source_type":     source.Type,
				"path":            source.Path,
				"repo":            source.Repo,
				"ignored_paths":   source.IgnoredPaths,
				"disabled_skills": source.DisabledSkills,
				"enabled_skills":  source.EnabledSkills,
				"type":            "skills",
			})
		}
	}
	return list
}

func (cm *ConfigManager) getAgentsList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0)
	if agents := GetAgentsConfig(); agents != nil {
		for key, agent := range agents.Agents {
			list = append(list, map[string]interface{}{
				"key":           key,
				"name":          agent.Name,
				"enabled":       agent.Enabled,
				"description":   agent.Description,
				"role":          agent.Role,
				"model":         agent.Model,
				"system_prompt": agent.SystemPrompt,
				"type":          "agents",
			})
		}
	}
	return list
}
