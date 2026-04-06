package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewToolRecommender(t *testing.T) {
	// 创建真实的MCP管理器
	mcpManager := NewMCPManager()

	// 创建推荐器
	recommender := NewToolRecommender(mcpManager, nil)
	assert.NotNil(t, recommender)
	assert.Equal(t, mcpManager, recommender.mcpManager)
	assert.Nil(t, recommender.aiService)
	assert.NotNil(t, recommender.cache)
	assert.Equal(t, 0, len(recommender.cache))
}

func TestToolRecommender_RecommendTool_EmptyQuery(t *testing.T) {
	mcpManager := NewMCPManager()
	recommender := NewToolRecommender(mcpManager, nil)

	result, err := recommender.RecommendTool(context.Background(), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "No MCP servers available", result.Reasoning)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestToolRecommender_RecommendTool_Basic(t *testing.T) {
	mcpManager := NewMCPManager()
	recommender := NewToolRecommender(mcpManager, nil)

	// 测试基本功能
	result, err := recommender.RecommendTool(context.Background(), "test query")
	// 由于没有配置服务器，应该返回"No MCP servers available"
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "No MCP servers available", result.Reasoning)
}
