package main

import (
	"fmt"
	"log"

	"backend/internal/config"
)

func main() {
	// 加载配置
	_, err := config.LoadModelsConfig("config/models.yaml")
	if err != nil {
		log.Fatalf("Failed to load models config: %v", err)
	}

	// 创建一个测试模型提供商
	provider := config.ModelProvider{
		Name:      "test-provider",
		Enabled:   true,
		APIKey:    "test-api-key-123",
		SecretKey: "test-secret-key-456",
		SecretID:  "test-secret-id-789",
		BaseURL:   "https://api.example.com",
		Endpoint:  "/chat/completions",
		AuthType:  "bearer",
		Timeout:   60,
		Models:    []string{"test-model"},
	}

	// 更新模型提供商
	config.UpdateModelProvider("test-provider", provider)

	fmt.Println("Test completed. Check .env file and models.yaml file.")
}
