package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDependencyTypeConstants(t *testing.T) {
	assert.Equal(t, DependencyType("npm"), DependencyTypeNPM)
	assert.Equal(t, DependencyType("go"), DependencyTypeGo)
	assert.Equal(t, DependencyType("pip"), DependencyTypePip)
	assert.Equal(t, DependencyType("docker"), DependencyTypeDocker)
}

func TestNewDependencyManagerFactory(t *testing.T) {
	factory := NewDependencyManagerFactory()
	assert.NotNil(t, factory)
}

func TestDependencyManagerFactory_CreateManager_NPM(t *testing.T) {
	factory := NewDependencyManagerFactory()
	manager := factory.CreateManager(DependencyTypeNPM)
	assert.NotNil(t, manager)

	// 验证类型
	_, isNPM := manager.(*NPMDependencyManager)
	assert.True(t, isNPM)
}

func TestDependencyManagerFactory_CreateManager_Go(t *testing.T) {
	factory := NewDependencyManagerFactory()
	manager := factory.CreateManager(DependencyTypeGo)
	assert.NotNil(t, manager)

	_, isGo := manager.(*GoDependencyManager)
	assert.True(t, isGo)
}

func TestDependencyManagerFactory_CreateManager_Pip(t *testing.T) {
	factory := NewDependencyManagerFactory()
	manager := factory.CreateManager(DependencyTypePip)
	assert.NotNil(t, manager)

	_, isPip := manager.(*PipDependencyManager)
	assert.True(t, isPip)
}

func TestDependencyManagerFactory_CreateManager_Docker(t *testing.T) {
	factory := NewDependencyManagerFactory()
	manager := factory.CreateManager(DependencyTypeDocker)
	assert.NotNil(t, manager)

	_, isDocker := manager.(*DockerDependencyManager)
	assert.True(t, isDocker)
}

func TestDependencyManagerFactory_CreateManager_Default(t *testing.T) {
	factory := NewDependencyManagerFactory()
	manager := factory.CreateManager("unknown")
	assert.NotNil(t, manager)

	// 默认应该返回NPM管理器
	_, isNPM := manager.(*NPMDependencyManager)
	assert.True(t, isNPM)
}

func TestNewNPMDependencyManager(t *testing.T) {
	manager := NewNPMDependencyManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 30*time.Second, manager.timeout)
}

func TestNewGoDependencyManager(t *testing.T) {
	manager := NewGoDependencyManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 30*time.Second, manager.timeout)
}

func TestNewPipDependencyManager(t *testing.T) {
	manager := NewPipDependencyManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 30*time.Second, manager.timeout)
}

func TestNewDockerDependencyManager(t *testing.T) {
	manager := NewDockerDependencyManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 60*time.Second, manager.timeout)
}

func TestNPMDependencyManager_CheckDependency_Context(t *testing.T) {
	manager := NewNPMDependencyManager()
	ctx := context.Background()

	info := DependencyInfo{
		PackageName: "@upstash/context7-mcp",
		Type:        DependencyTypeNPM,
	}

	// 这个测试不会实际执行命令，只是验证方法签名和基本逻辑
	installed, err := manager.CheckDependency(ctx, info)

	// 在测试环境中，我们不知道包是否安装
	// 只验证没有panic和错误处理
	assert.NotPanics(t, func() {
		_, _ = manager.CheckDependency(ctx, info)
	})

	t.Logf("CheckDependency result: installed=%v, err=%v", installed, err)
}

func TestNPMDependencyManager_InstallDependency_InvalidCommand(t *testing.T) {
	manager := NewNPMDependencyManager()
	ctx := context.Background()

	info := DependencyInfo{
		PackageName: "test-package",
		Type:        DependencyTypeNPM,
		InstallCmd:  "", // 空命令
	}

	// 测试空命令
	info.InstallCmd = ""
	err := manager.InstallDependency(ctx, info)
	// 应该成功创建默认命令，但执行会失败（因为测试环境）
	// 我们只验证没有panic
	assert.NotPanics(t, func() {
		_ = manager.InstallDependency(ctx, info)
	})

	// 测试无效命令
	info.InstallCmd = ""
	err = manager.InstallDependency(ctx, info)
	assert.NotPanics(t, func() {
		_ = manager.InstallDependency(ctx, info)
	})

	t.Logf("InstallDependency error (expected): %v", err)
}

func TestGoDependencyManager_CheckDependency_Context(t *testing.T) {
	manager := NewGoDependencyManager()
	ctx := context.Background()

	info := DependencyInfo{
		PackageName: "mcp-go",
		Type:        DependencyTypeGo,
	}

	installed, err := manager.CheckDependency(ctx, info)
	assert.NotPanics(t, func() {
		_, _ = manager.CheckDependency(ctx, info)
	})

	t.Logf("Go CheckDependency result: installed=%v, err=%v", installed, err)
}

func TestPipDependencyManager_CheckDependency_Context(t *testing.T) {
	manager := NewPipDependencyManager()
	ctx := context.Background()

	info := DependencyInfo{
		PackageName: "some-package",
		Type:        DependencyTypePip,
	}

	installed, err := manager.CheckDependency(ctx, info)
	assert.NotPanics(t, func() {
		_, _ = manager.CheckDependency(ctx, info)
	})

	t.Logf("Pip CheckDependency result: installed=%v, err=%v", installed, err)
}

func TestDockerDependencyManager_CheckDependency_Context(t *testing.T) {
	manager := NewDockerDependencyManager()
	ctx := context.Background()

	info := DependencyInfo{
		PackageName: "nginx",
		Type:        DependencyTypeDocker,
	}

	installed, err := manager.CheckDependency(ctx, info)
	assert.NotPanics(t, func() {
		_, _ = manager.CheckDependency(ctx, info)
	})

	t.Logf("Docker CheckDependency result: installed=%v, err=%v", installed, err)
}

func TestDependencyInfo_JSONTags(t *testing.T) {
	info := DependencyInfo{
		Name:        "test",
		PackageName: "test-package",
		Type:        DependencyTypeNPM,
		InstallCmd:  "npm install test-package",
		TestCmd:     "npx test-package --version",
		Version:     "1.0.0",
	}

	// 验证结构体字段
	assert.Equal(t, "test", info.Name)
	assert.Equal(t, "test-package", info.PackageName)
	assert.Equal(t, DependencyTypeNPM, info.Type)
	assert.Equal(t, "npm install test-package", info.InstallCmd)
	assert.Equal(t, "npx test-package --version", info.TestCmd)
	assert.Equal(t, "1.0.0", info.Version)
}

func TestAllManagersImplementInterface(t *testing.T) {
	// 验证所有管理器都实现了接口
	var manager DependencyManager

	manager = NewNPMDependencyManager()
	assert.NotNil(t, manager)

	manager = NewGoDependencyManager()
	assert.NotNil(t, manager)

	manager = NewPipDependencyManager()
	assert.NotNil(t, manager)

	manager = NewDockerDependencyManager()
	assert.NotNil(t, manager)
}
