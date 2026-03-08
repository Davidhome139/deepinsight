package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"backend/internal/config"

	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	configManager *config.ConfigManager
}

func NewSettingHandler() *SettingHandler {
	return &SettingHandler{
		configManager: config.NewConfigManager("config"),
	}
}

// GetAllSettings 获取所有设置
func (h *SettingHandler) GetAllSettings(c *gin.Context) {
	settings := h.configManager.GetAllSettings()
	c.JSON(http.StatusOK, settings)
}

// GetSettingsByType 根据类型获取设置
func (h *SettingHandler) GetSettingsByType(c *gin.Context) {
	settingType := c.Param("type")

	settings, err := h.configManager.GetSettingByType(settingType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateModelProvider 更新 AI 模型提供商（支持字段别名和健壮性验证）
func (h *SettingHandler) UpdateModelProvider(c *gin.Context) {
	providerKey := c.Param("key")

	// 读取原始 JSON 数据
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		log.Printf("UpdateModelProvider: Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	log.Printf("UpdateModelProvider: Received data for %s: %+v", providerKey, rawData)

	// 使用字段映射归一化数据
	normalizedData, err := config.NormalizeMap(rawData, config.ModelProvider{})
	if err != nil {
		log.Printf("UpdateModelProvider: Failed to normalize data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to normalize data: " + err.Error()})
		return
	}

	log.Printf("UpdateModelProvider: Normalized data: %+v", normalizedData)

	// 解码到结构体
	var req config.ModelProvider
	if err := config.DecodeWithFieldMapping(normalizedData, &req); err != nil {
		log.Printf("UpdateModelProvider: Failed to decode data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Failed to decode data: " + err.Error(),
			"normalized_data": normalizedData,
		})
		return
	}

	log.Printf("UpdateModelProvider: Decoded request: %+v", req)

	// 清理输入数据
	config.SanitizeModelProvider(&req)

	// 设置默认值
	config.SetDefaults(&req)

	// 验证配置
	validator := config.NewConfigValidator()
	if !validator.ValidateModelProvider(providerKey, req) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": validator.GetErrors(),
		})
		return
	}

	config.UpdateModelProvider(providerKey, req)
	log.Printf("Updated model provider: %s", providerKey)

	c.JSON(http.StatusOK, gin.H{
		"message": "Model provider updated",
		"key":     providerKey,
		"data":    req,
	})
}

// DeleteModelProvider 删除 AI 模型提供商
func (h *SettingHandler) DeleteModelProvider(c *gin.Context) {
	providerKey := c.Param("key")
	config.DeleteModelProvider(providerKey)
	log.Printf("Deleted model provider: %s", providerKey)
	c.JSON(http.StatusOK, gin.H{"message": "Model provider deleted", "key": providerKey})
}

// UpdateSearchProvider 更新搜索提供商（支持字段别名）
func (h *SettingHandler) UpdateSearchProvider(c *gin.Context) {
	providerKey := c.Param("key")

	// 读取原始 JSON 数据
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// 使用字段映射归一化数据
	normalizedData, err := config.NormalizeMap(rawData, config.SearchProvider{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to normalize data: " + err.Error()})
		return
	}

	// 解码到结构体
	var req config.SearchProvider
	if err := config.DecodeWithFieldMapping(normalizedData, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode data: " + err.Error()})
		return
	}

	config.UpdateSearchProviderConfig(providerKey, req)
	log.Printf("Updated search provider: %s", providerKey)

	c.JSON(http.StatusOK, gin.H{"message": "Search provider updated", "key": providerKey})
}

// DeleteSearchProvider 删除搜索提供商
func (h *SettingHandler) DeleteSearchProvider(c *gin.Context) {
	providerKey := c.Param("key")
	config.DeleteSearchProviderConfig(providerKey)
	log.Printf("Deleted search provider: %s", providerKey)
	c.JSON(http.StatusOK, gin.H{"message": "Search provider deleted", "key": providerKey})
}

// TestSearchProviderConnectivity 测试搜索引擎连通性
func (h *SettingHandler) TestSearchProviderConnectivity(c *gin.Context) {
	providerKey := c.Param("key")

	log.Printf("TestSearchProviderConnectivity: Testing provider %s", providerKey)

	// 获取提供商配置
	searchs := config.GetSearchsConfig()
	if searchs == nil {
		log.Printf("TestSearchProviderConnectivity: Searchs config not loaded")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Searchs config not loaded"})
		return
	}

	provider, exists := searchs.Providers[providerKey]
	if !exists {
		log.Printf("TestSearchProviderConnectivity: Provider %s not found", providerKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	log.Printf("TestSearchProviderConnectivity: Provider config: %+v", provider)

	// 执行连通性测试
	result := testSearchProviderConnectivity(providerKey, provider)
	log.Printf("TestSearchProviderConnectivity: Test result: %+v", result)
	c.JSON(http.StatusOK, result)
}

// UpdateMCPServer 更新 MCP 服务器
func (h *SettingHandler) UpdateMCPServer(c *gin.Context) {
	serverKey := c.Param("key")

	// 读取原始 JSON 数据
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	log.Printf("UpdateMCPServer: Received data for %s: %+v", serverKey, rawData)

	// 使用字段映射归一化数据
	normalizedData, err := config.NormalizeMap(rawData, config.MCPServer{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to normalize data: " + err.Error()})
		return
	}

	// 解码到结构体
	var req config.MCPServer
	if err := config.DecodeWithFieldMapping(normalizedData, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode data: " + err.Error()})
		return
	}

	// 验证配置
	validator := config.NewConfigValidator()
	if !validator.ValidateMCPServer(serverKey, req) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": validator.GetErrors(),
		})
		return
	}

	config.UpdateMCPServer(serverKey, req)
	log.Printf("Updated MCP server: %s", serverKey)

	c.JSON(http.StatusOK, gin.H{
		"message": "MCP server updated",
		"key":     serverKey,
		"data":    req,
	})
}

// DeleteMCPServer 删除 MCP 服务器
func (h *SettingHandler) DeleteMCPServer(c *gin.Context) {
	serverKey := c.Param("key")

	// 检查服务器是否存在
	mcps := config.GetMCPServersConfig()
	if mcps == nil || mcps.Servers == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP servers config not loaded"})
		return
	}

	if _, exists := mcps.Servers[serverKey]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	config.DeleteMCPServer(serverKey)
	log.Printf("Deleted MCP server: %s", serverKey)
	c.JSON(http.StatusOK, gin.H{"message": "MCP server deleted", "key": serverKey})
}

// TestMCPServerConnectivity 测试 MCP 服务器连通性
func (h *SettingHandler) TestMCPServerConnectivity(c *gin.Context) {
	serverKey := c.Param("key")

	log.Printf("TestMCPServerConnectivity: Testing server %s", serverKey)

	// 获取服务器配置
	mcps := config.GetMCPServersConfig()
	if mcps == nil || mcps.Servers == nil {
		log.Printf("TestMCPServerConnectivity: MCP servers config not loaded")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MCP servers config not loaded"})
		return
	}

	server, exists := mcps.Servers[serverKey]
	if !exists {
		log.Printf("TestMCPServerConnectivity: Server %s not found", serverKey)
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	log.Printf("TestMCPServerConnectivity: Server config: %+v", server)

	// 执行连通性测试
	result := testMCPServerConnectivity(serverKey, server)
	log.Printf("TestMCPServerConnectivity: Test result: %+v", result)
	c.JSON(http.StatusOK, result)
}

// UpdateSkill 更新技能
func (h *SettingHandler) UpdateSkill(c *gin.Context) {
	skillKey := c.Param("key")

	var req config.Skill
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.UpdateSkill(skillKey, req)
	log.Printf("Updated skill: %s", skillKey)

	c.JSON(http.StatusOK, gin.H{"message": "Skill updated", "key": skillKey})
}

// DeleteSkill 删除技能
func (h *SettingHandler) DeleteSkill(c *gin.Context) {
	skillKey := c.Param("key")
	config.DeleteSkill(skillKey)
	log.Printf("Deleted skill: %s", skillKey)
	c.JSON(http.StatusOK, gin.H{"message": "Skill deleted", "key": skillKey})
}

// UpdateAgent 更新智能体
func (h *SettingHandler) UpdateAgent(c *gin.Context) {
	agentKey := c.Param("key")

	var req config.Agent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.UpdateAgent(agentKey, req)
	log.Printf("Updated agent: %s", agentKey)

	c.JSON(http.StatusOK, gin.H{"message": "Agent updated", "key": agentKey})
}

// DeleteAgent 删除智能体
func (h *SettingHandler) DeleteAgent(c *gin.Context) {
	agentKey := c.Param("key")
	config.DeleteAgent(agentKey)
	log.Printf("Deleted agent: %s", agentKey)
	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted", "key": agentKey})
}

// Legacy methods for backward compatibility

// GetProviderSettingsLegacy 兼容旧版 API
func (h *SettingHandler) GetProviderSettingsLegacy(c *gin.Context) {
	providerType := c.Query("type")
	if providerType == "" {
		providerType = "ai"
	}

	var result []gin.H
	if providerType == "ai" || providerType == "models" {
		if models := config.GetModelsConfig(); models != nil {
			for key, p := range models.Providers {
				result = append(result, gin.H{
					"provider":   key,
					"type":       "ai",
					"enabled":    p.Enabled,
					"api_key":    p.APIKey,
					"secret_key": p.SecretKey,
					"secret_id":  p.SecretID,
					"base_url":   p.BaseURL,
				})
			}
		}
	} else if providerType == "search" {
		if searchs := config.GetSearchsConfig(); searchs != nil {
			for key, p := range searchs.Providers {
				result = append(result, gin.H{
					"provider": key,
					"type":     "search",
					"enabled":  p.Enabled,
					"api_key":  p.APIKey,
					"base_url": p.BaseURL,
				})
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// SaveProviderSettingLegacy 兼容旧版 API
func (h *SettingHandler) SaveProviderSettingLegacy(c *gin.Context) {
	var req struct {
		Provider  string `json:"provider"`
		Type      string `json:"type"`
		Enabled   bool   `json:"enabled"`
		APIKey    string `json:"api_key"`
		SecretKey string `json:"secret_key"`
		SecretID  string `json:"secret_id"`
		BaseURL   string `json:"base_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type == "ai" || req.Type == "models" {
		config.UpdateModelProvider(req.Provider, config.ModelProvider{
			Name:      req.Provider,
			Enabled:   req.Enabled,
			APIKey:    req.APIKey,
			SecretKey: req.SecretKey,
			SecretID:  req.SecretID,
			BaseURL:   req.BaseURL,
		})
	} else if req.Type == "search" {
		config.UpdateSearchProviderConfig(req.Provider, config.SearchProvider{
			Name:    req.Provider,
			Enabled: req.Enabled,
			APIKey:  req.APIKey,
			BaseURL: req.BaseURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider saved", "provider": req.Provider})
}

// DeleteProviderSettingLegacy 兼容旧版 API
func (h *SettingHandler) DeleteProviderSettingLegacy(c *gin.Context) {
	provider := c.Param("provider")
	providerType := c.Query("type")
	if providerType == "" {
		providerType = "ai"
	}

	if providerType == "ai" || providerType == "models" {
		config.DeleteModelProvider(provider)
	} else if providerType == "search" {
		config.DeleteSearchProviderConfig(provider)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted", "provider": provider})
}

// TestModelProviderConnectivity 测试模型提供商连通性
func (h *SettingHandler) TestModelProviderConnectivity(c *gin.Context) {
	providerKey := c.Param("key")

	// 获取提供商配置
	models := config.GetModelsConfig()
	if models == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Models config not loaded"})
		return
	}

	provider, exists := models.Providers[providerKey]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// 执行连通性测试
	result := testProviderConnectivity(providerKey, provider)
	c.JSON(http.StatusOK, result)
}

// testProviderConnectivity 执行实际的连通性测试
func testProviderConnectivity(key string, provider config.ModelProvider) gin.H {
	// 检查基本配置
	if provider.APIKey == "" {
		return gin.H{
			"success":  false,
			"message":  "API Key 未配置",
			"provider": key,
		}
	}

	if provider.BaseURL == "" {
		return gin.H{
			"success":  false,
			"message":  "Base URL 未配置",
			"provider": key,
		}
	}

	// 根据提供商类型执行不同的测试
	switch key {
	case "openai", "deepseek", "moonshot", "zhipu", "doubao":
		return testOpenAICompatible(key, provider)
	case "claude":
		return testClaude(key, provider)
	case "baidu":
		return testBaidu(key, provider)
	case "aliyun":
		return testAliyun(key, provider)
	case "tencent":
		return testTencentNewAPI(key, provider)
	case "tencentcloud":
		return testTencentCloudAPI(key, provider)
	case "minimax":
		return testMiniMax(key, provider)
	default:
		return testOpenAICompatible(key, provider)
	}
}

// testOpenAICompatible 测试 OpenAI 兼容 API
func testOpenAICompatible(key string, provider config.ModelProvider) gin.H {
	url := provider.BaseURL + provider.Endpoint
	if provider.Endpoint == "" {
		url = provider.BaseURL + "/v1/models"
	} else {
		// 尝试使用 models 端点或直接用 chat completions 端点做简单测试
		url = provider.BaseURL + "/v1/models"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	// 设置认证头
	authHeader := "Bearer " + provider.APIKey
	if provider.AuthType == "custom" && key == "zhipu" {
		// 智谱需要特殊处理，这里简化处理
		authHeader = provider.APIKey
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testClaude 测试 Claude API
func testClaude(key string, provider config.ModelProvider) gin.H {
	url := provider.BaseURL + "/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	req.Header.Set("x-api-key", provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testBaidu 测试百度文心 API
func testBaidu(key string, provider config.ModelProvider) gin.H {
	// 百度千帆 API v2 版本使用 Bearer Token 认证
	// api_key 格式: bce-v3/ALTAK-xxx/xxx
	url := provider.BaseURL + provider.Endpoint

	// 构造一个简单的测试请求
	testBody := map[string]interface{}{
		"model": provider.Models[0],
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"max_tokens": 1,
	}

	jsonBody, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	// 使用 Bearer Token 认证，api_key 就是 token
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查响应
	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	// 解析错误信息
	var errResp map[string]interface{}
	if err := json.Unmarshal(body, &errResp); err == nil {
		// 检查是否是认证错误
		if errData, ok := errResp["error"].(map[string]interface{}); ok {
			if errCode, ok := errData["code"].(string); ok {
				switch errCode {
				case "invalid_token", "token_expired":
					return gin.H{
						"success":  false,
						"message":  "API Key 无效或已过期",
						"provider": key,
						"status":   resp.StatusCode,
					}
				case "invalid_model":
					// 模型不存在，但认证通过了，说明连接成功
					return gin.H{
						"success":  true,
						"message":  "连接成功 (认证通过，但配置的模型可能不存在或无权限访问)",
						"provider": key,
						"status":   resp.StatusCode,
					}
				default:
					if errMsg, ok := errData["message"].(string); ok {
						return gin.H{
							"success":  false,
							"message":  "API 错误: " + errMsg,
							"provider": key,
							"status":   resp.StatusCode,
						}
					}
				}
			}
		}
		// 兼容旧版错误格式
		if errMsg, ok := errResp["error_msg"].(string); ok {
			return gin.H{
				"success":  false,
				"message":  "API 错误: " + errMsg,
				"provider": key,
				"status":   resp.StatusCode,
			}
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testAliyun 测试阿里云 API
func testAliyun(key string, provider config.ModelProvider) gin.H {
	// 阿里云使用 Bearer Token 认证
	url := provider.BaseURL + provider.Endpoint

	// 构造一个简单的请求体来测试
	testBody := map[string]interface{}{
		"model": provider.Models[0],
		"input": map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		},
		"parameters": map[string]interface{}{
			"max_tokens": 1,
		},
	}

	jsonBody, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 阿里云即使返回 400 也可能是正常的（因为模型参数问题），只要认证通过就行
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		var respData map[string]interface{}
		if err := json.Unmarshal(body, &respData); err == nil {
			if _, hasError := respData["error"]; hasError {
				errMap := respData["error"].(map[string]interface{})
				if code, ok := errMap["code"].(string); ok && code == "InvalidApiKey" {
					return gin.H{"success": false, "message": "API Key 无效", "provider": key}
				}
			}
		}
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testTencent 测试腾讯混元 API
func testTencent(key string, provider config.ModelProvider) gin.H {
	// 判断使用哪种认证方式
	// 新版混元 API (api.hunyuan.cloud.tencent.com) 使用 Bearer Token (APIKey)
	// 旧版腾讯云 API (hunyuan.tencentcloudapi.com) 使用 TC3-HMAC-SHA256 签名 (SecretID/SecretKey)

	if strings.Contains(provider.BaseURL, "api.hunyuan.cloud.tencent.com") {
		// 新版混元 API - 使用 Bearer Token
		return testTencentNewAPI(key, provider)
	} else {
		// 旧版腾讯云 API - 使用 TC3 签名
		return testTencentCloudAPI(key, provider)
	}
}

// testTencentNewAPI 测试新版腾讯混元 API (Bearer Token)
func testTencentNewAPI(key string, provider config.ModelProvider) gin.H {
	if provider.APIKey == "" {
		return gin.H{"success": false, "message": "API Key 未配置", "provider": key}
	}

	url := provider.BaseURL + "/chat/completions"

	// 构造测试请求
	testBody := map[string]interface{}{
		"model": provider.Models[0],
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"stream": false,
	}

	jsonBody, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	// 使用 Bearer Token 认证
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查响应
	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功 (Bearer Token 认证)",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	// 解析错误信息
	var errResp map[string]interface{}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errData, ok := errResp["error"].(map[string]interface{}); ok {
			if errMsg, ok := errData["message"].(string); ok {
				return gin.H{
					"success":  false,
					"message":  "API 错误: " + errMsg,
					"provider": key,
					"status":   resp.StatusCode,
				}
			}
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testTencentCloudAPI 测试旧版腾讯云 API (TC3-HMAC-SHA256)
func testTencentCloudAPI(key string, provider config.ModelProvider) gin.H {
	// 腾讯云 API 需要 SecretID 和 SecretKey
	if provider.SecretID == "" || provider.SecretKey == "" {
		return gin.H{"success": false, "message": "SecretID 或 SecretKey 未配置", "provider": key}
	}

	// 腾讯云使用 TC3-HMAC-SHA256 签名认证
	var host string
	var requestURL string

	if provider.BaseURL != "" {
		// 解析 base_url 获取 host
		if len(provider.BaseURL) > 8 {
			urlPart := provider.BaseURL
			if strings.HasPrefix(urlPart, "https://") {
				urlPart = urlPart[8:]
			} else if strings.HasPrefix(urlPart, "http://") {
				urlPart = urlPart[7:]
			}
			parts := strings.Split(urlPart, "/")
			host = parts[0]
		}
		requestURL = provider.BaseURL + "/chat/completions"
	} else {
		host = "hunyuan.tencentcloudapi.com"
		requestURL = "https://hunyuan.tencentcloudapi.com/v1/chat/completions"
	}

	// 构造测试请求
	testBody := map[string]interface{}{
		"Model": provider.Models[0],
		"Messages": []map[string]string{
			{"Role": "user", "Content": "Hello"},
		},
		"Stream": false,
	}

	jsonBody, _ := json.Marshal(testBody)

	// 生成腾讯云签名
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	service := "hunyuan"
	algorithm := "TC3-HMAC-SHA256"

	// 1. 构建规范请求串
	httpRequestMethod := "POST"
	canonicalURI := "/chat/completions"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", "application/json", host)
	signedHeaders := "content-type;host"
	hashedRequestPayload := fmt.Sprintf("%x", sha256.Sum256(jsonBody))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// 2. 构建待签名字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// 3. 计算签名
	secretDate := hmacSHA256([]byte("TC3"+provider.SecretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := fmt.Sprintf("%x", hmacSHA256(secretSigning, []byte(stringToSign)))

	// 4. 构建 Authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		provider.SecretID,
		credentialScope,
		signedHeaders,
		signature)

	// 发送请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", "ChatCompletions")
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", "2023-09-01")
	req.Header.Set("X-TC-Region", "ap-guangzhou")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查响应
	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功 (TC3-HMAC-SHA256 签名认证)",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	// 解析错误信息
	var errResp map[string]interface{}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if response, ok := errResp["Response"].(map[string]interface{}); ok {
			if errData, ok := response["Error"].(map[string]interface{}); ok {
				if errMsg, ok := errData["Message"].(string); ok {
					return gin.H{
						"success":  false,
						"message":  "API 错误: " + errMsg,
						"provider": key,
						"status":   resp.StatusCode,
					}
				}
			}
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// hmacSHA256 计算 HMAC-SHA256
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// testSearchProviderConnectivity 测试搜索引擎连通性
func testSearchProviderConnectivity(key string, provider config.SearchProvider) gin.H {
	// 检查基本配置
	if provider.APIKey == "" {
		return gin.H{
			"success":  false,
			"message":  "API Key 未配置",
			"provider": key,
		}
	}

	// 根据不同搜索引擎执行不同的测试
	switch key {
	case "baidu", "百度web_search":
		return testBaiduSearch(key, provider)
	case "google":
		return testGoogleSearch(key, provider)
	case "bing":
		return testBingSearch(key, provider)
	case "duckduckgo":
		return testDuckDuckGoSearch(key, provider)
	default:
		return testGenericSearch(key, provider)
	}
}

// testBaiduSearch 测试百度千帆 AI 搜索
func testBaiduSearch(key string, provider config.SearchProvider) gin.H {
	log.Printf("testBaiduSearch: Testing Baidu search for provider %s", key)
	log.Printf("testBaiduSearch: Provider config - APIKey: %s, BaseURL: %s",
		maskAPIKey(provider.APIKey), provider.BaseURL)

	// 使用配置的 base_url 测试百度服务
	testURL := provider.BaseURL
	if testURL == "" {
		return gin.H{"success": false, "message": "BaseURL 未配置", "provider": key}
	}

	// 构造测试请求（按照百度千帆 API 文档）
	testBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "test"},
		},
		"search_source": provider.SearchSource,
		"stream":        false,
	}

	jsonBody, _ := json.Marshal(testBody)
	log.Printf("testBaiduSearch: Request URL: %s", testURL)

	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("testBaiduSearch: Failed to create request: %v", err)
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	// 百度千帆使用 Authorization 头部
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("testBaiduSearch: Request failed: %v", err)
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("testBaiduSearch: Response status: %d", resp.StatusCode)
	log.Printf("testBaiduSearch: Response body: %s", string(body))

	// 检查认证错误
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return gin.H{
			"success":  false,
			"message":  fmt.Sprintf("认证失败 (状态码: %d)，请检查 API Key 是否正确。注意：百度千帆需要使用有效的 Bearer Token", resp.StatusCode),
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	// 200 或 400 都可能表示认证通过（400 可能是参数问题）
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// maskAPIKey 隐藏 API Key 的敏感部分
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 10 {
		return "***"
	}
	return apiKey[:5] + "..." + apiKey[len(apiKey)-5:]
}

// testGoogleSearch 测试 Google 搜索
func testGoogleSearch(key string, provider config.SearchProvider) gin.H {
	// Google 搜索 API 测试
	testURL := "https://www.googleapis.com/customsearch/v1"

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	// 添加 API Key 作为查询参数（不带实际搜索词，只测试认证）
	q := req.URL.Query()
	q.Add("key", provider.APIKey)
	q.Add("cx", "test")
	q.Add("q", "test")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 401 或 403 表示认证失败（API Key 无效）
	// 400 表示请求参数错误（但认证通过）
	if resp.StatusCode == http.StatusBadRequest {
		return gin.H{
			"success":  true,
			"message":  "连接成功 (API Key 有效，但缺少搜索参数)",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testBingSearch 测试 Bing 搜索
func testBingSearch(key string, provider config.SearchProvider) gin.H {
	// Bing 搜索 API 测试
	testURL := "https://api.bing.microsoft.com/v7.0/search"

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", provider.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 401 表示认证失败
	// 400 表示缺少搜索词（但认证通过）
	if resp.StatusCode == http.StatusBadRequest {
		return gin.H{
			"success":  true,
			"message":  "连接成功 (API Key 有效，但缺少搜索参数)",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("API 返回错误 (状态码: %d): %s", resp.StatusCode, string(body)),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testDuckDuckGoSearch 测试 DuckDuckGo 搜索
func testDuckDuckGoSearch(key string, provider config.SearchProvider) gin.H {
	// DuckDuckGo 没有官方 API，使用网页测试
	testURL := "https://duckduckgo.com"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(testURL)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功 (DuckDuckGo 服务可访问)",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("服务返回错误 (状态码: %d)", resp.StatusCode),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testGenericSearch 通用搜索引擎测试
func testGenericSearch(key string, provider config.SearchProvider) gin.H {
	// 通用测试：检查 base_url 是否可访问
	if provider.BaseURL == "" {
		return gin.H{
			"success":  false,
			"message":  "Base URL 未配置",
			"provider": key,
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(provider.BaseURL)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	return gin.H{
		"success":  false,
		"message":  fmt.Sprintf("服务返回错误 (状态码: %d)", resp.StatusCode),
		"provider": key,
		"status":   resp.StatusCode,
	}
}

// testMCPServerConnectivity 测试 MCP 服务器连通性
func testMCPServerConnectivity(key string, server config.MCPServer) gin.H {
	// 检查基本配置
	if server.Type != "builtin" && server.Command == "" {
		return gin.H{
			"success": false,
			"message": "命令未配置",
			"server":  key,
		}
	}

	// builtin 类型直接返回成功
	if server.Type == "builtin" {
		return gin.H{
			"success": true,
			"message": "内置服务器无需测试",
			"server":  key,
		}
	}

	// 测试命令是否存在
	cmdPath, err := exec.LookPath(server.Command)
	if err != nil {
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("命令 '%s' 未找到: %v", server.Command, err),
			"server":  key,
		}
	}

	// 尝试启动进程并检查是否可以初始化 MCP 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, server.Args...)

	// 设置环境变量
	for k, v := range server.Env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	// 启动进程
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("无法创建 stdin pipe: %v", err),
			"server":  key,
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("无法创建 stdout pipe: %v", err),
			"server":  key,
		}
	}

	if err := cmd.Start(); err != nil {
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("无法启动进程: %v", err),
			"server":  key,
		}
	}

	// 发送初始化请求
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	jsonReq, _ := json.Marshal(initReq)
	if _, err := stdin.Write(append(jsonReq, '\n')); err != nil {
		cmd.Process.Kill()
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("发送初始化请求失败: %v", err),
			"server":  key,
		}
	}

	// 读取响应
	decoder := json.NewDecoder(stdout)
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		cmd.Process.Kill()
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("读取响应失败: %v", err),
			"server":  key,
		}
	}

	// 发送退出通知
	exitReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/exit",
	}
	jsonExit, _ := json.Marshal(exitReq)
	stdin.Write(append(jsonExit, '\n'))

	// 清理进程
	cmd.Process.Kill()
	cmd.Wait()

	// 检查响应
	if errorData, ok := response["error"]; ok {
		return gin.H{
			"success": false,
			"message": fmt.Sprintf("MCP 初始化错误: %v", errorData),
			"server":  key,
		}
	}

	if _, ok := response["result"]; ok {
		return gin.H{
			"success": true,
			"message": "MCP 服务器连接成功",
			"server":  key,
		}
	}

	return gin.H{
		"success":  false,
		"message":  "MCP 服务器响应格式不正确",
		"server":   key,
		"response": response,
	}
}

// testMiniMax 测试 MiniMax API
func testMiniMax(key string, provider config.ModelProvider) gin.H {
	url := provider.BaseURL + provider.Endpoint

	// MiniMax 使用 Bearer Token + Group ID
	testBody := map[string]interface{}{
		"model": provider.Models[0],
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"max_tokens": 1,
	}

	jsonBody, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return gin.H{"success": false, "message": "创建请求失败: " + err.Error(), "provider": key}
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return gin.H{"success": false, "message": "连接失败: " + err.Error(), "provider": key}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return gin.H{
			"success":  true,
			"message":  "连接成功",
			"provider": key,
			"status":   resp.StatusCode,
		}
	}

	// MiniMax 可能返回 400 等错误，但只要不是认证错误就算连通
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		var respData map[string]interface{}
		if err := json.Unmarshal(body, &respData); err == nil {
			if base, ok := respData["base_resp"].(map[string]interface{}); ok {
				if statusCode, ok := base["status_code"].(float64); ok && statusCode == 401 {
					return gin.H{"success": false, "message": "API Key 无效", "provider": key}
				}
			}
		}
	}

	return gin.H{
		"success":  true,
		"message":  fmt.Sprintf("连接成功 (返回状态码: %d，可能是参数问题)", resp.StatusCode),
		"provider": key,
		"status":   resp.StatusCode,
	}
}
