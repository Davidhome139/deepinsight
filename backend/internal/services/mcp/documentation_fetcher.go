package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DocumentationInfo 文档信息
type DocumentationInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Repository  string `json:"repository,omitempty"`
	Version     string `json:"version,omitempty"`
	Readme      string `json:"readme,omitempty"`
}

// DocumentationFetcher 文档获取器接口
type DocumentationFetcher interface {
	FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error)
}

// NPMDocumentationFetcher NPM文档获取器
type NPMDocumentationFetcher struct {
	httpClient *http.Client
	npmRegistryURL string
}

func NewNPMDocumentationFetcher() *NPMDocumentationFetcher {
	return &NPMDocumentationFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		npmRegistryURL: "https://registry.npmjs.org",
	}
}

func (f *NPMDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	if depType != DependencyTypeNPM {
		return nil, fmt.Errorf("unsupported dependency type for NPM fetcher: %s", depType)
	}

	// 从NPM注册表获取包信息
	url := fmt.Sprintf("%s/%s", f.npmRegistryURL, packageName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析NPM响应
	var npmResponse map[string]interface{}
	if err := json.Unmarshal(body, &npmResponse); err != nil {
		return nil, fmt.Errorf("failed to parse npm response: %v", err)
	}

	// 提取最新版本信息
	distTags, ok := npmResponse["dist-tags"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid npm response: missing dist-tags")
	}

	latestVersion, ok := distTags["latest"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid npm response: missing latest version")
	}

	versions, ok := npmResponse["versions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid npm response: missing versions")
	}

	latestVersionInfo, ok := versions[latestVersion].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid npm response: missing version info for %s", latestVersion)
	}

	// 构建文档信息
	doc := &DocumentationInfo{
		Name:        packageName,
		Version:     latestVersion,
	}

	// 提取描述
	if description, ok := latestVersionInfo["description"].(string); ok {
		doc.Description = description
	} else if description, ok := npmResponse["description"].(string); ok {
		doc.Description = description
	} else {
		doc.Description = fmt.Sprintf("MCP server: %s", packageName)
	}

	// 提取主页
	if homepage, ok := latestVersionInfo["homepage"].(string); ok {
		doc.Homepage = homepage
	} else if homepage, ok := npmResponse["homepage"].(string); ok {
		doc.Homepage = homepage
	} else {
		doc.Homepage = fmt.Sprintf("https://www.npmjs.com/package/%s", packageName)
	}

	// 提取仓库信息
	if repository, ok := latestVersionInfo["repository"].(map[string]interface{}); ok {
		if repoURL, ok := repository["url"].(string); ok {
			doc.Repository = strings.TrimPrefix(strings.TrimSuffix(repoURL, ".git"), "git+")
		}
	} else if repository, ok := npmResponse["repository"].(map[string]interface{}); ok {
		if repoURL, ok := repository["url"].(string); ok {
			doc.Repository = strings.TrimPrefix(strings.TrimSuffix(repoURL, ".git"), "git+")
		}
	}

	return doc, nil
}

// GoDocumentationFetcher Go文档获取器
type GoDocumentationFetcher struct {
	httpClient *http.Client
}

func NewGoDocumentationFetcher() *GoDocumentationFetcher {
	return &GoDocumentationFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (f *GoDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	if depType != DependencyTypeGo {
		return nil, fmt.Errorf("unsupported dependency type for Go fetcher: %s", depType)
	}

	// Go包通常托管在GitHub上
	// 简化版本：返回基本信息
	doc := &DocumentationInfo{
		Name:        packageName,
		Description: fmt.Sprintf("Go MCP server: %s", packageName),
		Homepage:    fmt.Sprintf("https://pkg.go.dev/%s", packageName),
		Repository:  fmt.Sprintf("https://%s", strings.TrimPrefix(packageName, "github.com/")),
	}

	// 尝试从GitHub API获取更多信息
	if strings.HasPrefix(packageName, "github.com/") {
		repoPath := strings.TrimPrefix(packageName, "github.com/")
		githubURL := fmt.Sprintf("https://api.github.com/repos/%s", repoPath)
		
		req, err := http.NewRequestWithContext(ctx, "GET", githubURL, nil)
		if err != nil {
			// 如果失败，返回基本信息
			return doc, nil
		}
		
		req.Header.Set("User-Agent", "MCP-Automation-System")
		
		resp, err := f.httpClient.Do(req)
		if err != nil {
			return doc, nil
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return doc, nil
			}
			
			var githubResponse map[string]interface{}
			if err := json.Unmarshal(body, &githubResponse); err != nil {
				return doc, nil
			}
			
			// 更新描述
			if description, ok := githubResponse["description"].(string); ok && description != "" {
				doc.Description = description
			}
			
			// 更新主页
			if homepage, ok := githubResponse["homepage"].(string); ok && homepage != "" {
				doc.Homepage = homepage
			}
		}
	}

	return doc, nil
}

// PipDocumentationFetcher Pip文档获取器
type PipDocumentationFetcher struct {
	httpClient *http.Client
}

func NewPipDocumentationFetcher() *PipDocumentationFetcher {
	return &PipDocumentationFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (f *PipDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	if depType != DependencyTypePip {
		return nil, fmt.Errorf("unsupported dependency type for Pip fetcher: %s", depType)
	}

	// PyPI API
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		// 如果API失败，返回基本信息
		return &DocumentationInfo{
			Name:        packageName,
			Description: fmt.Sprintf("Python MCP server: %s", packageName),
			Homepage:    fmt.Sprintf("https://pypi.org/project/%s/", packageName),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 如果API返回错误，返回基本信息
		return &DocumentationInfo{
			Name:        packageName,
			Description: fmt.Sprintf("Python MCP server: %s", packageName),
			Homepage:    fmt.Sprintf("https://pypi.org/project/%s/", packageName),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var pypiResponse map[string]interface{}
	if err := json.Unmarshal(body, &pypiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse PyPI response: %v", err)
	}

	info, ok := pypiResponse["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid PyPI response: missing info")
	}

	doc := &DocumentationInfo{
		Name: packageName,
	}

	// 提取版本
	if version, ok := info["version"].(string); ok {
		doc.Version = version
	}

	// 提取描述
	if description, ok := info["description"].(string); ok {
		doc.Description = description
	} else if summary, ok := info["summary"].(string); ok {
		doc.Description = summary
	} else {
		doc.Description = fmt.Sprintf("Python MCP server: %s", packageName)
	}

	// 提取主页
	if homepage, ok := info["home_page"].(string); ok && homepage != "" {
		doc.Homepage = homepage
	} else {
		doc.Homepage = fmt.Sprintf("https://pypi.org/project/%s/", packageName)
	}

	// 提取项目URLs
	if projectURLs, ok := info["project_urls"].(map[string]interface{}); ok {
		if repository, ok := projectURLs["Repository"].(string); ok {
			doc.Repository = repository
		} else if homepage, ok := projectURLs["Homepage"].(string); ok {
			doc.Homepage = homepage
		}
	}

	return doc, nil
}

// DockerDocumentationFetcher Docker文档获取器
type DockerDocumentationFetcher struct {
	httpClient *http.Client
}

func NewDockerDocumentationFetcher() *DockerDocumentationFetcher {
	return &DockerDocumentationFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (f *DockerDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	if depType != DependencyTypeDocker {
		return nil, fmt.Errorf("unsupported dependency type for Docker fetcher: %s", depType)
	}

	// Docker Hub API (简化版本)
	// 注意：Docker Hub API需要认证，这里返回基本信息
	doc := &DocumentationInfo{
		Name:        packageName,
		Description: fmt.Sprintf("Docker MCP server: %s", packageName),
		Homepage:    fmt.Sprintf("https://hub.docker.com/_/%s", strings.Split(packageName, ":")[0]),
	}

	// 尝试从Docker Hub获取描述
	// 简化版本：只返回基本信息
	return doc, nil
}

// DocumentationFetcherFactory 文档获取器工厂
type DocumentationFetcherFactory struct{}

func NewDocumentationFetcherFactory() *DocumentationFetcherFactory {
	return &DocumentationFetcherFactory{}
}

func (f *DocumentationFetcherFactory) CreateFetcher(depType DependencyType) DocumentationFetcher {
	switch depType {
	case DependencyTypeNPM:
		return NewNPMDocumentationFetcher()
	case DependencyTypeGo:
		return NewGoDocumentationFetcher()
	case DependencyTypePip:
		return NewPipDocumentationFetcher()
	case DependencyTypeDocker:
		return NewDockerDocumentationFetcher()
	default:
		// 默认返回NPM获取器
		return NewNPMDocumentationFetcher()
	}
}