package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DependencyType 依赖类型
type DependencyType string

const (
	DependencyTypeNPM    DependencyType = "npm"
	DependencyTypeGo     DependencyType = "go"
	DependencyTypePip    DependencyType = "pip"
	DependencyTypeDocker DependencyType = "docker"
)

// DependencyInfo 依赖信息
type DependencyInfo struct {
	Name        string         `json:"name"`
	PackageName string         `json:"packageName"`
	Type        DependencyType `json:"type"`
	InstallCmd  string         `json:"installCmd"`
	TestCmd     string         `json:"testCmd"`
	Version     string         `json:"version,omitempty"`
}

// DependencyManager 依赖管理器接口
type DependencyManager interface {
	CheckDependency(ctx context.Context, info DependencyInfo) (bool, error)
	InstallDependency(ctx context.Context, info DependencyInfo) error
	UpdateDependency(ctx context.Context, info DependencyInfo) error
	RemoveDependency(ctx context.Context, info DependencyInfo) error
	GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error)
}

// NPMDependencyManager NPM依赖管理器
type NPMDependencyManager struct {
	timeout time.Duration
}

func NewNPMDependencyManager() *NPMDependencyManager {
	return &NPMDependencyManager{
		timeout: 30 * time.Second,
	}
}

func (m *NPMDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 检查是否全局安装
	cmd := exec.CommandContext(ctx, "npm", "list", "-g", info.PackageName, "--depth=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果命令执行失败，可能表示包未安装
		return false, nil
	}

	// 检查输出中是否包含包名
	outputStr := string(output)
	return strings.Contains(outputStr, info.PackageName), nil
}

func (m *NPMDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 使用提供的安装命令，如果为空则使用默认命令
	installCmd := info.InstallCmd
	if installCmd == "" {
		installCmd = fmt.Sprintf("npm install -g %s", info.PackageName)
	}

	// 解析命令
	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid install command: %s", installCmd)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *NPMDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npm", "update", "-g", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *NPMDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npm", "uninstall", "-g", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *NPMDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 使用提供的测试命令，如果为空则使用默认命令
	testCmd := info.TestCmd
	if testCmd == "" {
		testCmd = fmt.Sprintf("npx -y %s --version", info.PackageName)
	}

	// 解析命令
	parts := strings.Fields(testCmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid test command: %s", testCmd)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %v", info.PackageName, err)
	}

	version := strings.TrimSpace(string(output))
	return version, nil
}

// GoDependencyManager Go依赖管理器
type GoDependencyManager struct {
	timeout time.Duration
}

func NewGoDependencyManager() *GoDependencyManager {
	return &GoDependencyManager{
		timeout: 30 * time.Second,
	}
}

func (m *GoDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 检查是否在PATH中
	cmd := exec.CommandContext(ctx, "which", info.PackageName)
	_, err := cmd.CombinedOutput()
	if err == nil {
		return true, nil
	}

	// 或者检查go bin目录
	cmd = exec.CommandContext(ctx, "go", "version", "-m", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}

	return len(output) > 0, nil
}

func (m *GoDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 使用提供的安装命令，如果为空则使用默认命令
	installCmd := info.InstallCmd
	if installCmd == "" {
		installCmd = fmt.Sprintf("go install %s@latest", info.PackageName)
	}

	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid install command: %s", installCmd)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *GoDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	// 对于Go，更新和安装是一样的
	return m.InstallDependency(ctx, info)
}

func (m *GoDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 查找二进制文件路径
	cmd := exec.CommandContext(ctx, "which", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dependency %s not found: %v", info.PackageName, err)
	}

	binaryPath := strings.TrimSpace(string(output))
	if binaryPath == "" {
		return fmt.Errorf("dependency %s not found", info.PackageName)
	}

	// 删除二进制文件
	cmd = exec.CommandContext(ctx, "rm", "-f", binaryPath)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove binary %s: %v", binaryPath, err)
	}

	return nil
}

func (m *GoDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, info.PackageName, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %v", info.PackageName, err)
	}

	version := strings.TrimSpace(string(output))
	return version, nil
}

// PipDependencyManager Pip依赖管理器
type PipDependencyManager struct {
	timeout time.Duration
}

func NewPipDependencyManager() *PipDependencyManager {
	return &PipDependencyManager{
		timeout: 30 * time.Second,
	}
}

func (m *PipDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pip", "show", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}

	return len(output) > 0, nil
}

func (m *PipDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	installCmd := info.InstallCmd
	if installCmd == "" {
		installCmd = fmt.Sprintf("pip install %s", info.PackageName)
	}

	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid install command: %s", installCmd)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *PipDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pip", "install", "--upgrade", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *PipDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pip", "uninstall", "-y", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *PipDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pip", "show", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %v", info.PackageName, err)
	}

	// 解析版本信息
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Version:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("version not found for %s", info.PackageName)
}

// DockerDependencyManager Docker依赖管理器
type DockerDependencyManager struct {
	timeout time.Duration
}

func NewDockerDependencyManager() *DockerDependencyManager {
	return &DockerDependencyManager{
		timeout: 60 * time.Second,
	}
}

func (m *DockerDependencyManager) CheckDependency(ctx context.Context, info DependencyInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "images", "--format", "{{.Repository}}", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, image := range images {
		if image == info.PackageName {
			return true, nil
		}
	}

	return false, nil
}

func (m *DockerDependencyManager) InstallDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	installCmd := info.InstallCmd
	if installCmd == "" {
		installCmd = fmt.Sprintf("docker pull %s", info.PackageName)
	}

	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid install command: %s", installCmd)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *DockerDependencyManager) UpdateDependency(ctx context.Context, info DependencyInfo) error {
	// 对于Docker，更新就是重新拉取
	return m.InstallDependency(ctx, info)
}

func (m *DockerDependencyManager) RemoveDependency(ctx context.Context, info DependencyInfo) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "rmi", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove dependency %s: %v\nOutput: %s", info.PackageName, err, string(output))
	}

	return nil
}

func (m *DockerDependencyManager) GetDependencyVersion(ctx context.Context, info DependencyInfo) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.RepoTags}}", info.PackageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %v", info.PackageName, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// DependencyManagerFactory 依赖管理器工厂
type DependencyManagerFactory struct{}

func NewDependencyManagerFactory() *DependencyManagerFactory {
	return &DependencyManagerFactory{}
}

func (f *DependencyManagerFactory) CreateManager(depType DependencyType) DependencyManager {
	switch depType {
	case DependencyTypeNPM:
		return NewNPMDependencyManager()
	case DependencyTypeGo:
		return NewGoDependencyManager()
	case DependencyTypePip:
		return NewPipDependencyManager()
	case DependencyTypeDocker:
		return NewDockerDependencyManager()
	default:
		// 默认返回NPM管理器
		return NewNPMDependencyManager()
	}
}