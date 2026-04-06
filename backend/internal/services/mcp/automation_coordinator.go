package mcp

import (
	"backend/internal/config"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AutomationCoordinator 自动化协调器接口
type AutomationCoordinator interface {
	Start(ctx context.Context) error
	Stop() error
	AddMCPPackage(packageName string, depType DependencyType) error
	RemoveMCPPackage(packageName string) error
	UpdateMCPPackage(packageName string, depType DependencyType) error
	GetStatus() AutomationStatus
	GetPackageStatus(packageName string) (*PackageAutomationStatus, error)
}

// AutomationStatus 自动化状态
type AutomationStatus struct {
	Running      bool                        `json:"running"`
	TotalPackages int                        `json:"totalPackages"`
	ActiveJobs   int                         `json:"activeJobs"`
	LastSync     time.Time                   `json:"lastSync"`
	Packages     map[string]PackageAutomationStatus `json:"packages"`
}

// PackageAutomationStatus 包自动化状态
type PackageAutomationStatus struct {
	PackageName    string        `json:"packageName"`
	DependencyType DependencyType `json:"dependencyType"`
	InstallStatus  string        `json:"installStatus"`  // pending, installing, installed, failed
	UpdateStatus   string        `json:"updateStatus"`   // pending, updating, updated, failed
	ConfigStatus   string        `json:"configStatus"`   // pending, generated, failed
	ConnectionStatus string      `json:"connectionStatus"` // pending, connecting, connected, failed
	LastCheck      time.Time     `json:"lastCheck"`
	LastError      string        `json:"lastError,omitempty"`
}

// DefaultAutomationCoordinator 默认自动化协调器
type DefaultAutomationCoordinator struct {
	depManager      DependencyManager
	docFetcher      DocumentationFetcher
	configGenerator ConfigGenerator
	hotReloadManager HotReloadManager
	mcpManager      config.MCPManagerInterface
	
	mu     sync.RWMutex
	status AutomationStatus
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	configPath string
	docsPath   string
}

func NewDefaultAutomationCoordinator(
	depManager DependencyManager,
	docFetcher DocumentationFetcher,
	configGenerator ConfigGenerator,
	hotReloadManager HotReloadManager,
	mcpManager config.MCPManagerInterface,
	configPath string,
	docsPath string,
) *DefaultAutomationCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DefaultAutomationCoordinator{
		depManager:      depManager,
		docFetcher:      docFetcher,
		configGenerator: configGenerator,
		hotReloadManager: hotReloadManager,
		mcpManager:      mcpManager,
		status: AutomationStatus{
			Running:       false,
			TotalPackages: 0,
			ActiveJobs:    0,
			LastSync:      time.Time{},
			Packages:      make(map[string]PackageAutomationStatus),
		},
		ctx:        ctx,
		cancel:     cancel,
		configPath: configPath,
		docsPath:   docsPath,
	}
}

func (c *DefaultAutomationCoordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.status.Running {
		return fmt.Errorf("automation coordinator is already running")
	}
	
	// 启动热加载管理器
	if c.hotReloadManager != nil {
		if err := c.hotReloadManager.Start(ctx); err != nil {
			log.Printf("[Automation] Failed to start hot reload manager: %v", err)
		}
	}
	
	// 启动定期同步goroutine
	c.wg.Add(1)
	go c.periodicSync()
	
	c.status.Running = true
	log.Println("[Automation] Automation coordinator started")
	
	return nil
}

func (c *DefaultAutomationCoordinator) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.status.Running {
		return nil
	}
	
	// 取消上下文
	c.cancel()
	
	// 停止热加载管理器
	if c.hotReloadManager != nil {
		if err := c.hotReloadManager.Stop(); err != nil {
			log.Printf("[Automation] Error stopping hot reload manager: %v", err)
		}
	}
	
	// 等待所有goroutine完成
	c.wg.Wait()
	
	c.status.Running = false
	log.Println("[Automation] Automation coordinator stopped")
	
	return nil
}

func (c *DefaultAutomationCoordinator) AddMCPPackage(packageName string, depType DependencyType) error {
	c.mu.Lock()
	
	// 检查包是否已存在
	if _, exists := c.status.Packages[packageName]; exists {
		c.mu.Unlock()
		return fmt.Errorf("package %s already exists", packageName)
	}
	
	// 添加包状态
	c.status.Packages[packageName] = PackageAutomationStatus{
		PackageName:    packageName,
		DependencyType: depType,
		InstallStatus:  "pending",
		UpdateStatus:   "pending",
		ConfigStatus:   "pending",
		ConnectionStatus: "pending",
		LastCheck:      time.Now(),
	}
	
	c.status.TotalPackages = len(c.status.Packages)
	c.status.ActiveJobs++
	
	c.mu.Unlock()
	
	log.Printf("[Automation] Added MCP package: %s (type: %v)", packageName, depType)
	
	// 启动自动化流程
	go c.automatePackage(packageName, depType)
	
	return nil
}

func (c *DefaultAutomationCoordinator) RemoveMCPPackage(packageName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.status.Packages[packageName]; !exists {
		return fmt.Errorf("package %s not found", packageName)
	}
	
	// 从状态中移除
	delete(c.status.Packages, packageName)
	c.status.TotalPackages = len(c.status.Packages)
	
	log.Printf("[Automation] Removed MCP package: %s", packageName)
	
	return nil
}

func (c *DefaultAutomationCoordinator) UpdateMCPPackage(packageName string, depType DependencyType) error {
	c.mu.Lock()
	
	// 检查包是否存在
	pkgStatus, exists := c.status.Packages[packageName]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("package %s not found", packageName)
	}
	
	// 更新包状态
	pkgStatus.DependencyType = depType
	pkgStatus.UpdateStatus = "pending"
	pkgStatus.LastCheck = time.Now()
	c.status.Packages[packageName] = pkgStatus
	
	c.status.ActiveJobs++
	
	c.mu.Unlock()
	
	log.Printf("[Automation] Updating MCP package: %s (type: %v)", packageName, depType)
	
	// 启动更新流程
	go c.updatePackage(packageName, depType)
	
	return nil
}

func (c *DefaultAutomationCoordinator) GetStatus() AutomationStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *DefaultAutomationCoordinator) GetPackageStatus(packageName string) (*PackageAutomationStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	pkgStatus, exists := c.status.Packages[packageName]
	if !exists {
		return nil, fmt.Errorf("package %s not found", packageName)
	}
	
	return &pkgStatus, nil
}

func (c *DefaultAutomationCoordinator) automatePackage(packageName string, depType DependencyType) {
	defer func() {
		c.mu.Lock()
		c.status.ActiveJobs--
		c.mu.Unlock()
	}()
	
	log.Printf("[Automation] Starting automation for package: %s", packageName)
	
	// 1. 检查依赖
	c.updatePackageStatus(packageName, "installStatus", "installing", "")
	if err := c.checkAndInstallDependency(packageName, depType); err != nil {
		c.updatePackageStatus(packageName, "installStatus", "failed", err.Error())
		return
	}
	c.updatePackageStatus(packageName, "installStatus", "installed", "")
	
	// 2. 获取文档
	c.updatePackageStatus(packageName, "configStatus", "pending", "")
	docInfo, err := c.fetchDocumentation(packageName, depType)
	if err != nil {
		c.updatePackageStatus(packageName, "configStatus", "failed", err.Error())
		return
	}
	
	// 3. 生成配置
	depInfo := &DependencyInfo{
		Name:        packageName,
		PackageName: packageName,
		Type:        depType,
		InstallCmd:  c.getInstallCommand(packageName, depType),
		TestCmd:     c.getTestCommand(packageName, depType),
	}
	
	serverConfig, err := c.configGenerator.GenerateServerConfig(docInfo, depInfo)
	if err != nil {
		c.updatePackageStatus(packageName, "configStatus", "failed", err.Error())
		return
	}
	
	// 4. 保存配置
	if err := c.configGenerator.SaveConfigToFile(serverConfig, c.configPath); err != nil {
		c.updatePackageStatus(packageName, "configStatus", "failed", err.Error())
		return
	}
	c.updatePackageStatus(packageName, "configStatus", "generated", "")
	
	// 5. 连接服务器
	c.updatePackageStatus(packageName, "connectionStatus", "connecting", "")
	if err := c.connectServer(serverConfig.Name); err != nil {
		c.updatePackageStatus(packageName, "connectionStatus", "failed", err.Error())
		return
	}
	c.updatePackageStatus(packageName, "connectionStatus", "connected", "")
	
	log.Printf("[Automation] Automation completed for package: %s", packageName)
}

func (c *DefaultAutomationCoordinator) updatePackage(packageName string, depType DependencyType) {
	defer func() {
		c.mu.Lock()
		c.status.ActiveJobs--
		c.mu.Unlock()
	}()
	
	log.Printf("[Automation] Starting update for package: %s", packageName)
	
	// 1. 更新依赖
	c.updatePackageStatus(packageName, "updateStatus", "updating", "")
	if err := c.updateDependency(packageName, depType); err != nil {
		c.updatePackageStatus(packageName, "updateStatus", "failed", err.Error())
		return
	}
	c.updatePackageStatus(packageName, "updateStatus", "updated", "")
	
	// 2. 重新连接服务器
	c.updatePackageStatus(packageName, "connectionStatus", "connecting", "")
	if err := c.connectServer(packageName); err != nil {
		c.updatePackageStatus(packageName, "connectionStatus", "failed", err.Error())
		return
	}
	c.updatePackageStatus(packageName, "connectionStatus", "connected", "")
	
	log.Printf("[Automation] Update completed for package: %s", packageName)
}

func (c *DefaultAutomationCoordinator) checkAndInstallDependency(packageName string, depType DependencyType) error {
	ctx := context.Background()
	
	// 创建依赖信息
	depInfo := &DependencyInfo{
		Name:        packageName,
		PackageName: packageName,
		Type:        depType,
	}
	
	// 检查依赖是否已安装
	installed, err := c.depManager.CheckDependency(ctx, *depInfo)
	if err != nil {
		return fmt.Errorf("failed to check dependency: %v", err)
	}
	
	if !installed {
		log.Printf("[Automation] Installing dependency: %s", packageName)
		if err := c.depManager.InstallDependency(ctx, *depInfo); err != nil {
			return fmt.Errorf("failed to install dependency: %v", err)
		}
		log.Printf("[Automation] Dependency installed: %s", packageName)
	} else {
		log.Printf("[Automation] Dependency already installed: %s", packageName)
	}
	
	return nil
}

func (c *DefaultAutomationCoordinator) updateDependency(packageName string, depType DependencyType) error {
	ctx := context.Background()
	
	// 创建依赖信息
	depInfo := &DependencyInfo{
		Name:        packageName,
		PackageName: packageName,
		Type:        depType,
	}
	
	log.Printf("[Automation] Updating dependency: %s", packageName)
	if err := c.depManager.UpdateDependency(ctx, *depInfo); err != nil {
		return fmt.Errorf("failed to update dependency: %v", err)
	}
	
	log.Printf("[Automation] Dependency updated: %s", packageName)
	return nil
}

func (c *DefaultAutomationCoordinator) fetchDocumentation(packageName string, depType DependencyType) (*DocumentationInfo, error) {
	ctx := context.Background()
	
	log.Printf("[Automation] Fetching documentation for: %s", packageName)
	docInfo, err := c.docFetcher.FetchDocumentation(ctx, packageName, depType)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documentation: %v", err)
	}
	
	log.Printf("[Automation] Documentation fetched for: %s", packageName)
	return docInfo, nil
}

func (c *DefaultAutomationCoordinator) connectServer(serverName string) error {
	if c.mcpManager == nil {
		return fmt.Errorf("MCP manager not available")
	}
	
	log.Printf("[Automation] Connecting to server: %s", serverName)
	if err := c.mcpManager.ConnectToServer(serverName); err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	
	log.Printf("[Automation] Connected to server: %s", serverName)
	return nil
}

func (c *DefaultAutomationCoordinator) updatePackageStatus(packageName, field, value, errorMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	pkgStatus, exists := c.status.Packages[packageName]
	if !exists {
		return
	}
	
	pkgStatus.LastCheck = time.Now()
	
	switch field {
	case "installStatus":
		pkgStatus.InstallStatus = value
	case "updateStatus":
		pkgStatus.UpdateStatus = value
	case "configStatus":
		pkgStatus.ConfigStatus = value
	case "connectionStatus":
		pkgStatus.ConnectionStatus = value
	}
	
	if errorMsg != "" {
		pkgStatus.LastError = errorMsg
	} else {
		pkgStatus.LastError = ""
	}
	
	c.status.Packages[packageName] = pkgStatus
}

func (c *DefaultAutomationCoordinator) getInstallCommand(packageName string, depType DependencyType) string {
	switch depType {
	case DependencyTypeNPM:
		return fmt.Sprintf("npm install -g %s", packageName)
	case DependencyTypeGo:
		return fmt.Sprintf("go install %s@latest", packageName)
	case DependencyTypePip:
		return fmt.Sprintf("pip install %s", packageName)
	case DependencyTypeDocker:
		return fmt.Sprintf("docker pull %s", packageName)
	default:
		return ""
	}
}

func (c *DefaultAutomationCoordinator) getTestCommand(packageName string, depType DependencyType) string {
	switch depType {
	case DependencyTypeNPM:
		return fmt.Sprintf("npx -y %s --version", packageName)
	case DependencyTypeGo:
		// 提取二进制名称
		binaryName := packageName
		if len(packageName) > 0 {
			// 如果是完整路径，取最后一部分
			if idx := len(packageName) - 1; idx >= 0 {
				// 简单处理：使用包名作为二进制名
				binaryName = packageName
			}
		}
		return fmt.Sprintf("%s --version", binaryName)
	case DependencyTypePip:
		return fmt.Sprintf("python -m %s --version", packageName)
	case DependencyTypeDocker:
		return fmt.Sprintf("docker run --rm %s --version", packageName)
	default:
		return ""
	}
}

func (c *DefaultAutomationCoordinator) periodicSync() {
	defer c.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	log.Println("[Automation] Periodic sync started")
	
	for {
		select {
		case <-c.ctx.Done():
			log.Println("[Automation] Periodic sync stopped")
			return
			
		case <-ticker.C:
			c.syncPackages()
		}
	}
}

func (c *DefaultAutomationCoordinator) syncPackages() {
	c.mu.Lock()
	c.status.LastSync = time.Now()
	c.mu.Unlock()
	
	log.Println("[Automation] Syncing packages...")
	
	// 这里可以添加包同步逻辑，例如：
	// 1. 检查所有包的更新
	// 2. 验证服务器连接状态
	// 3. 清理失败的任务
	
	// 简单实现：更新所有包的最后检查时间
	c.mu.Lock()
	for packageName, pkgStatus := range c.status.Packages {
		pkgStatus.LastCheck = time.Now()
		c.status.Packages[packageName] = pkgStatus
	}
	c.mu.Unlock()
	
	log.Println("[Automation] Packages synced")
}