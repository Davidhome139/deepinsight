package handlers

import (
	"backend/internal/services/mcp"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MCPAutomationHandler MCP自动化处理器
type MCPAutomationHandler struct {
	coordinator mcp.AutomationCoordinator
}

// NewMCPAutomationHandler 创建MCP自动化处理器
func NewMCPAutomationHandler(coordinator mcp.AutomationCoordinator) *MCPAutomationHandler {
	return &MCPAutomationHandler{
		coordinator: coordinator,
	}
}

// AddMCPPackageRequest 添加MCP包请求
type AddMCPPackageRequest struct {
	PackageName string `json:"packageName" binding:"required"`
	PackageType string `json:"packageType" binding:"required,oneof=npm go pip docker"`
}

// AddMCPPackageResponse 添加MCP包响应
type AddMCPPackageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Package struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"package"`
}

// AddMCPPackage 添加MCP包
// @Summary 添加MCP包
// @Description 添加新的MCP包并启动自动化流程
// @Tags mcp-automation
// @Accept json
// @Produce json
// @Param request body AddMCPPackageRequest true "添加MCP包请求"
// @Success 200 {object} AddMCPPackageResponse
// @Router /mcp-automation/add [post]
func (h *MCPAutomationHandler) AddMCPPackage(c *gin.Context) {
	var req AddMCPPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换包类型
	var depType mcp.DependencyType
	switch req.PackageType {
	case "npm":
		depType = mcp.DependencyTypeNPM
	case "go":
		depType = mcp.DependencyTypeGo
	case "pip":
		depType = mcp.DependencyTypePip
	case "docker":
		depType = mcp.DependencyTypeDocker
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package type"})
		return
	}

	// 添加包
	err := h.coordinator.AddMCPPackage(req.PackageName, depType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := AddMCPPackageResponse{
		Success: true,
		Message: "MCP package added successfully",
	}
	response.Package.Name = req.PackageName
	response.Package.Type = req.PackageType

	c.JSON(http.StatusOK, response)
}

// RemoveMCPPackageRequest 移除MCP包请求
type RemoveMCPPackageRequest struct {
	PackageName string `json:"packageName" binding:"required"`
}

// RemoveMCPPackageResponse 移除MCP包响应
type RemoveMCPPackageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RemoveMCPPackage 移除MCP包
// @Summary 移除MCP包
// @Description 移除MCP包并停止相关服务
// @Tags mcp-automation
// @Accept json
// @Produce json
// @Param request body RemoveMCPPackageRequest true "移除MCP包请求"
// @Success 200 {object} RemoveMCPPackageResponse
// @Router /mcp-automation/remove [delete]
func (h *MCPAutomationHandler) RemoveMCPPackage(c *gin.Context) {
	var req RemoveMCPPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 移除包
	err := h.coordinator.RemoveMCPPackage(req.PackageName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := RemoveMCPPackageResponse{
		Success: true,
		Message: "MCP package removed successfully",
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMCPPackageRequest 更新MCP包请求
type UpdateMCPPackageRequest struct {
	PackageName string `json:"packageName" binding:"required"`
	PackageType string `json:"packageType" binding:"required,oneof=npm go pip docker"`
}

// UpdateMCPPackageResponse 更新MCP包响应
type UpdateMCPPackageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateMCPPackage 更新MCP包
// @Summary 更新MCP包
// @Description 更新MCP包类型并重新配置
// @Tags mcp-automation
// @Accept json
// @Produce json
// @Param request body UpdateMCPPackageRequest true "更新MCP包请求"
// @Success 200 {object} UpdateMCPPackageResponse
// @Router /mcp-automation/update [put]
func (h *MCPAutomationHandler) UpdateMCPPackage(c *gin.Context) {
	var req UpdateMCPPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换包类型
	var depType mcp.DependencyType
	switch req.PackageType {
	case "npm":
		depType = mcp.DependencyTypeNPM
	case "go":
		depType = mcp.DependencyTypeGo
	case "pip":
		depType = mcp.DependencyTypePip
	case "docker":
		depType = mcp.DependencyTypeDocker
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package type"})
		return
	}

	// 更新包
	err := h.coordinator.UpdateMCPPackage(req.PackageName, depType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := UpdateMCPPackageResponse{
		Success: true,
		Message: "MCP package updated successfully",
	}

	c.JSON(http.StatusOK, response)
}

// PackageStatusResponse 包状态响应
type PackageStatusResponse struct {
	PackageName      string    `json:"packageName"`
	PackageType      string    `json:"packageType"`
	InstallStatus    string    `json:"installStatus"`
	ConfigStatus     string    `json:"configStatus"`
	ConnectionStatus string    `json:"connectionStatus"`
	UpdateStatus     string    `json:"updateStatus"`
	LastError        string    `json:"lastError,omitempty"`
	LastCheck        time.Time `json:"lastCheck"`
}

// GetPackageStatus 获取包状态
// @Summary 获取包状态
// @Description 获取指定MCP包的状态信息
// @Tags mcp-automation
// @Produce json
// @Param packageName path string true "包名称"
// @Success 200 {object} PackageStatusResponse
// @Router /mcp-automation/status/{packageName} [get]
func (h *MCPAutomationHandler) GetPackageStatus(c *gin.Context) {
	packageName := c.Param("packageName")

	// 获取包状态
	status, err := h.coordinator.GetPackageStatus(packageName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 转换包类型
	var packageType string
	switch status.DependencyType {
	case mcp.DependencyTypeNPM:
		packageType = "npm"
	case mcp.DependencyTypeGo:
		packageType = "go"
	case mcp.DependencyTypePip:
		packageType = "pip"
	case mcp.DependencyTypeDocker:
		packageType = "docker"
	default:
		packageType = "unknown"
	}

	response := PackageStatusResponse{
		PackageName:      status.PackageName,
		PackageType:      packageType,
		InstallStatus:    status.InstallStatus,
		ConfigStatus:     status.ConfigStatus,
		ConnectionStatus: status.ConnectionStatus,
		UpdateStatus:     status.UpdateStatus,
		LastError:        status.LastError,
		LastCheck:        status.LastCheck,
	}

	c.JSON(http.StatusOK, response)
}

// AutomationStatusResponse 自动化状态响应
type AutomationStatusResponse struct {
	Running      bool                                `json:"running"`
	TotalPackages int                                `json:"totalPackages"`
	ActiveJobs   int                                `json:"activeJobs"`
	Packages     map[string]PackageStatusResponse `json:"packages"`
}

// GetAutomationStatus 获取自动化状态
// @Summary 获取自动化状态
// @Description 获取MCP自动化系统的整体状态
// @Tags mcp-automation
// @Produce json
// @Success 200 {object} AutomationStatusResponse
// @Router /mcp-automation/status [get]
func (h *MCPAutomationHandler) GetAutomationStatus(c *gin.Context) {
	// 获取自动化状态
	status := h.coordinator.GetStatus()

	// 转换包状态
	packages := make(map[string]PackageStatusResponse)
	for name, pkgStatus := range status.Packages {
		// 转换包类型
		var packageType string
		switch pkgStatus.DependencyType {
		case mcp.DependencyTypeNPM:
			packageType = "npm"
		case mcp.DependencyTypeGo:
			packageType = "go"
		case mcp.DependencyTypePip:
			packageType = "pip"
		case mcp.DependencyTypeDocker:
			packageType = "docker"
		default:
			packageType = "unknown"
		}

		packages[name] = PackageStatusResponse{
			PackageName:      pkgStatus.PackageName,
			PackageType:      packageType,
			InstallStatus:    pkgStatus.InstallStatus,
			ConfigStatus:     pkgStatus.ConfigStatus,
			ConnectionStatus: pkgStatus.ConnectionStatus,
			UpdateStatus:     pkgStatus.UpdateStatus,
			LastError:        pkgStatus.LastError,
			LastCheck:        pkgStatus.LastCheck,
		}
	}

	response := AutomationStatusResponse{
		Running:       status.Running,
		TotalPackages: status.TotalPackages,
		ActiveJobs:    status.ActiveJobs,
		Packages:      packages,
	}

	c.JSON(http.StatusOK, response)
}

// StartAutomationResponse 启动自动化响应
type StartAutomationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StartAutomation 启动自动化
// @Summary 启动自动化
// @Description 启动MCP自动化协调器
// @Tags mcp-automation
// @Produce json
// @Success 200 {object} StartAutomationResponse
// @Router /mcp-automation/start [post]
func (h *MCPAutomationHandler) StartAutomation(c *gin.Context) {
	// 启动协调器
	err := h.coordinator.Start(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := StartAutomationResponse{
		Success: true,
		Message: "MCP automation started successfully",
	}

	c.JSON(http.StatusOK, response)
}

// StopAutomationResponse 停止自动化响应
type StopAutomationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StopAutomation 停止自动化
// @Summary 停止自动化
// @Description 停止MCP自动化协调器
// @Tags mcp-automation
// @Produce json
// @Success 200 {object} StopAutomationResponse
// @Router /mcp-automation/stop [post]
func (h *MCPAutomationHandler) StopAutomation(c *gin.Context) {
	// 停止协调器
	err := h.coordinator.Stop()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := StopAutomationResponse{
		Success: true,
		Message: "MCP automation stopped successfully",
	}

	c.JSON(http.StatusOK, response)
}