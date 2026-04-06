package mcp

import (
	"context"
)

// FactoryDependencyManager 工厂依赖管理器适配器
type FactoryDependencyManager struct {
	factory *DependencyManagerFactory
}

// NewFactoryDependencyManager 创建工厂依赖管理器适配器
func NewFactoryDependencyManager(factory *DependencyManagerFactory) *FactoryDependencyManager {
	return &FactoryDependencyManager{
		factory: factory,
	}
}

func (m *FactoryDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	manager := m.factory.CreateManager(info.Type)
	return manager.CheckDependency(ctx, info)
}

func (m *FactoryDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	manager := m.factory.CreateManager(info.Type)
	return manager.InstallDependency(ctx, info)
}

func (m *FactoryDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	manager := m.factory.CreateManager(info.Type)
	return manager.UpdateDependency(ctx, info)
}

func (m *FactoryDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	manager := m.factory.CreateManager(info.Type)
	return manager.RemoveDependency(ctx, info)
}

func (m *FactoryDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	manager := m.factory.CreateManager(info.Type)
	return manager.GetDependencyVersion(ctx, info)
}

// FactoryDocumentationFetcher 工厂文档获取器适配器
type FactoryDocumentationFetcher struct {
	factory *DocumentationFetcherFactory
}

// NewFactoryDocumentationFetcher 创建工厂文档获取器适配器
func NewFactoryDocumentationFetcher(factory *DocumentationFetcherFactory) *FactoryDocumentationFetcher {
	return &FactoryDocumentationFetcher{
		factory: factory,
	}
}

func (f *FactoryDocumentationFetcher) FetchDocumentation(ctx context.Context, packageName string, depType DependencyType) (*DocumentationInfo, error) {
	fetcher := f.factory.CreateFetcher(depType)
	return fetcher.FetchDocumentation(ctx, packageName, depType)
}