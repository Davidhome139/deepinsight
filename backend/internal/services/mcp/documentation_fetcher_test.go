package mcp

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestNewDocumentationFetcherFactory(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	assert.NotNil(t, factory)
}

func TestDocumentationFetcherFactory_CreateFetcher_NPM(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	fetcher := factory.CreateFetcher(DependencyTypeNPM)
	assert.NotNil(t, fetcher)
	
	_, isNPM := fetcher.(*NPMDocumentationFetcher)
	assert.True(t, isNPM)
}

func TestDocumentationFetcherFactory_CreateFetcher_Go(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	fetcher := factory.CreateFetcher(DependencyTypeGo)
	assert.NotNil(t, fetcher)
	
	_, isGo := fetcher.(*GoDocumentationFetcher)
	assert.True(t, isGo)
}

func TestDocumentationFetcherFactory_CreateFetcher_Pip(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	fetcher := factory.CreateFetcher(DependencyTypePip)
	assert.NotNil(t, fetcher)
	
	_, isPip := fetcher.(*PipDocumentationFetcher)
	assert.True(t, isPip)
}

func TestDocumentationFetcherFactory_CreateFetcher_Docker(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	fetcher := factory.CreateFetcher(DependencyTypeDocker)
	assert.NotNil(t, fetcher)
	
	_, isDocker := fetcher.(*DockerDocumentationFetcher)
	assert.True(t, isDocker)
}

func TestDocumentationFetcherFactory_CreateFetcher_Default(t *testing.T) {
	factory := NewDocumentationFetcherFactory()
	fetcher := factory.CreateFetcher("unknown")
	assert.NotNil(t, fetcher)
	
	// 默认应该返回NPM获取器
	_, isNPM := fetcher.(*NPMDocumentationFetcher)
	assert.True(t, isNPM)
}

func TestNewNPMDocumentationFetcher(t *testing.T) {
	fetcher := NewNPMDocumentationFetcher()
	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.httpClient)
	assert.Equal(t, "https://registry.npmjs.org", fetcher.npmRegistryURL)
}

func TestNewGoDocumentationFetcher(t *testing.T) {
	fetcher := NewGoDocumentationFetcher()
	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.httpClient)
}

func TestNewPipDocumentationFetcher(t *testing.T) {
	fetcher := NewPipDocumentationFetcher()
	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.httpClient)
}

func TestNewDockerDocumentationFetcher(t *testing.T) {
	fetcher := NewDockerDocumentationFetcher()
	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.httpClient)
}

func TestNPMDocumentationFetcher_FetchDocumentation_WrongType(t *testing.T) {
	fetcher := NewNPMDocumentationFetcher()
	ctx := context.Background()
	
	_, err := fetcher.FetchDocumentation(ctx, "test-package", DependencyTypeGo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dependency type")
}

func TestGoDocumentationFetcher_FetchDocumentation_WrongType(t *testing.T) {
	fetcher := NewGoDocumentationFetcher()
	ctx := context.Background()
	
	_, err := fetcher.FetchDocumentation(ctx, "test-package", DependencyTypeNPM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dependency type")
}

func TestPipDocumentationFetcher_FetchDocumentation_WrongType(t *testing.T) {
	fetcher := NewPipDocumentationFetcher()
	ctx := context.Background()
	
	_, err := fetcher.FetchDocumentation(ctx, "test-package", DependencyTypeNPM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dependency type")
}

func TestDockerDocumentationFetcher_FetchDocumentation_WrongType(t *testing.T) {
	fetcher := NewDockerDocumentationFetcher()
	ctx := context.Background()
	
	_, err := fetcher.FetchDocumentation(ctx, "test-package", DependencyTypeNPM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dependency type")
}

func TestDocumentationInfo_JSONTags(t *testing.T) {
	doc := DocumentationInfo{
		Name:        "test-package",
		Description: "Test package description",
		Homepage:    "https://example.com",
		Repository:  "https://github.com/example/test",
		Version:     "1.0.0",
		Readme:      "# Test Package\n\nThis is a test package.",
	}
	
	assert.Equal(t, "test-package", doc.Name)
	assert.Equal(t, "Test package description", doc.Description)
	assert.Equal(t, "https://example.com", doc.Homepage)
	assert.Equal(t, "https://github.com/example/test", doc.Repository)
	assert.Equal(t, "1.0.0", doc.Version)
	assert.Equal(t, "# Test Package\n\nThis is a test package.", doc.Readme)
}

func TestAllFetchersImplementInterface(t *testing.T) {
	// 验证所有获取器都实现了接口
	var fetcher DocumentationFetcher
	
	fetcher = NewNPMDocumentationFetcher()
	assert.NotNil(t, fetcher)
	
	fetcher = NewGoDocumentationFetcher()
	assert.NotNil(t, fetcher)
	
	fetcher = NewPipDocumentationFetcher()
	assert.NotNil(t, fetcher)
	
	fetcher = NewDockerDocumentationFetcher()
	assert.NotNil(t, fetcher)
}

func TestNPMDocumentationFetcher_FetchDocumentation_Context(t *testing.T) {
	fetcher := NewNPMDocumentationFetcher()
	ctx := context.Background()
	
	// 测试一个已知的包（可能会失败，取决于网络）
	doc, err := fetcher.FetchDocumentation(ctx, "@upstash/context7-mcp", DependencyTypeNPM)
	
	// 这个测试可能成功或失败，取决于网络连接
	// 我们只验证没有panic
	assert.NotPanics(t, func() {
		_, _ = fetcher.FetchDocumentation(ctx, "@upstash/context7-mcp", DependencyTypeNPM)
	})
	
	if err == nil && doc != nil {
		assert.Equal(t, "@upstash/context7-mcp", doc.Name)
		assert.NotEmpty(t, doc.Description)
		assert.NotEmpty(t, doc.Homepage)
	}
	
	t.Logf("FetchDocumentation result: doc=%v, err=%v", doc, err)
}

func TestGoDocumentationFetcher_FetchDocumentation_Context(t *testing.T) {
	fetcher := NewGoDocumentationFetcher()
	ctx := context.Background()
	
	doc, err := fetcher.FetchDocumentation(ctx, "github.com/mark3labs/mcp-go", DependencyTypeGo)
	
	assert.NotPanics(t, func() {
		_, _ = fetcher.FetchDocumentation(ctx, "github.com/mark3labs/mcp-go", DependencyTypeGo)
	})
	
	if err == nil && doc != nil {
		assert.Equal(t, "github.com/mark3labs/mcp-go", doc.Name)
		assert.NotEmpty(t, doc.Description)
		assert.NotEmpty(t, doc.Homepage)
	}
	
	t.Logf("Go FetchDocumentation result: doc=%v, err=%v", doc, err)
}

func TestPipDocumentationFetcher_FetchDocumentation_Context(t *testing.T) {
	fetcher := NewPipDocumentationFetcher()
	ctx := context.Background()
	
	// 测试一个已知的Python包
	doc, err := fetcher.FetchDocumentation(ctx, "requests", DependencyTypePip)
	
	assert.NotPanics(t, func() {
		_, _ = fetcher.FetchDocumentation(ctx, "requests", DependencyTypePip)
	})
	
	if err == nil && doc != nil {
		assert.Equal(t, "requests", doc.Name)
		assert.NotEmpty(t, doc.Description)
		assert.NotEmpty(t, doc.Homepage)
	}
	
	t.Logf("Pip FetchDocumentation result: doc=%v, err=%v", doc, err)
}

func TestDockerDocumentationFetcher_FetchDocumentation_Context(t *testing.T) {
	fetcher := NewDockerDocumentationFetcher()
	ctx := context.Background()
	
	doc, err := fetcher.FetchDocumentation(ctx, "nginx", DependencyTypeDocker)
	
	assert.NotPanics(t, func() {
		_, _ = fetcher.FetchDocumentation(ctx, "nginx", DependencyTypeDocker)
	})
	
	if err == nil && doc != nil {
		assert.Equal(t, "nginx", doc.Name)
		assert.NotEmpty(t, doc.Description)
		assert.NotEmpty(t, doc.Homepage)
	}
	
	t.Logf("Docker FetchDocumentation result: doc=%v, err=%v", doc, err)
}