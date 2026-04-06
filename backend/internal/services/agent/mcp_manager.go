package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"backend/internal/config"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPManager manages MCP server connections
type MCPManager struct {
	servers    map[string]*config.MCPServer
	mu         sync.RWMutex
	discovered bool
	discoverMu sync.RWMutex

	// Documentation storage
	docsPath string
	// Separate documentation storage
	documentations map[string]*config.MCPServerDocumentation

	// Resource management
	resourceManager *ResourceManager
	circuitBreakers map[string]*CircuitBreaker
	monitor         *Monitor

	// Configuration validation
	configValidator *ConfigValidator

	// Health check management
	healthCheckCtx    context.Context
	healthCheckCancel context.CancelFunc
	healthCheckWg     sync.WaitGroup
}

// NewMCPManager creates a new MCP manager
func NewMCPManager() *MCPManager {
	// Use log.Println to ensure output is captured in Docker logs
	log.Println("[MCP] ========== NewMCPManager called ==========")

	// Create context for health checks
	healthCtx, healthCancel := context.WithCancel(context.Background())

	m := &MCPManager{
		servers:           make(map[string]*config.MCPServer),
		docsPath:          "/app/config/mcp_docs", // Docker container path
		documentations:    make(map[string]*config.MCPServerDocumentation),
		resourceManager:   NewResourceManager(DefaultResourceLimits),
		circuitBreakers:   make(map[string]*CircuitBreaker),
		monitor:           NewMonitor(),
		configValidator:   NewConfigValidator(),
		healthCheckCtx:    healthCtx,
		healthCheckCancel: healthCancel,
	}

	// 启用健康检查
	m.startHealthCheck()
	log.Println("[MCP] Health check ENABLED")

	log.Println("[MCP] MCPManager created (discovery will be triggered on demand)")
	log.Println("[MCP] ========== NewMCPManager returning ==========")
	return m
}

// startHealthCheck starts the health check goroutine
func (m *MCPManager) startHealthCheck() {
	m.healthCheckWg.Add(1)
	go func() {
		defer m.healthCheckWg.Done()

		ticker := time.NewTicker(60 * time.Second) // Check every minute
		defer ticker.Stop()

		log.Println("[MCP] Health check goroutine started")

		for {
			select {
			case <-m.healthCheckCtx.Done():
				log.Println("[MCP] Health check goroutine stopped")
				return
			case <-ticker.C:
				m.performHealthCheck()
			}
		}
	}()
}

// performHealthCheck checks the health of all servers and attempts to reconnect if needed
func (m *MCPManager) performHealthCheck() {
	m.mu.RLock()
	servers := make([]string, 0, len(m.servers))
	for name := range m.servers {
		servers = append(servers, name)
	}
	m.mu.RUnlock()

	for _, serverName := range servers {
		server, ok := m.GetServer(serverName)
		if !ok {
			continue
		}

		// Check if server needs reconnection
		if server.NeedsReconnect {
			// Check if enough time has passed since last reconnect attempt
			if time.Since(server.LastReconnectAttempt) > 5*time.Minute {
				log.Printf("[MCP] Health check: server %s needs reconnection, attempting...", serverName)

				// Update last reconnect attempt time
				server.LastReconnectAttempt = time.Now()
				m.mu.Lock()
				m.servers[serverName] = server
				m.mu.Unlock()

				// Attempt reconnection in background
				go func(name string) {
					err := m.ConnectToServer(name)
					if err != nil {
						log.Printf("[MCP] Health check: failed to reconnect to server %s: %v", name, err)
					} else {
						log.Printf("[MCP] Health check: successfully reconnected to server %s", name)

						// Update server state
						updatedServer, ok := m.GetServer(name)
						if ok {
							updatedServer.NeedsReconnect = false
							updatedServer.LastError = ""
							m.mu.Lock()
							m.servers[name] = updatedServer
							m.mu.Unlock()
						}
					}
				}(serverName)
			}
		}
	}
}

// Discover discovers and connects to all enabled MCP servers
func (m *MCPManager) Discover() {
	// Use log.Println to ensure output is captured in Docker logs
	log.Println("[MCP] ========== Starting MCP discovery ==========")

	// Get MCP servers config
	cfg := config.GetMCPServersConfig()
	if cfg == nil {
		log.Println("[MCP] No MCP servers config found")
		return
	}

	log.Printf("[MCP] Found %d MCP servers in config\n", len(cfg.Servers))

	// Connect to each enabled server asynchronously
	// This ensures that failure of one server doesn't block others
	for name, server := range cfg.Servers {
		if !server.Enabled {
			log.Printf("[MCP] Server %s is disabled, skipping\n", name)
			continue
		}

		// Check if we should skip this server (already connected or circuit breaker open)
		if m.shouldSkipServer(name) {
			continue
		}

		log.Printf("[MCP] Processing server: %s (type: %s)\n", name, server.Type)

		// Create a copy of the server to avoid race conditions
		serverCopy := server
		serverCopy.Name = name

		// Add server to map immediately (even if not connected yet)
		m.mu.Lock()
		m.servers[name] = &serverCopy
		m.mu.Unlock()

		// Connect to server asynchronously (don't wait for completion)
		// This ensures that failure of one server doesn't block the whole application
		log.Printf("[MCP] Starting asynchronous connection to %s...", name)
		go func(s config.MCPServer) {
			// Add a small delay to ensure goroutine is scheduled
			time.Sleep(100 * time.Millisecond)
			m.connectServer(s)
		}(serverCopy)
	}

	log.Println("[MCP] ========== MCP discovery started (async) ==========")
}

// Refresh forces a re-discovery of MCP servers
func (m *MCPManager) Refresh() {
	log.Println("[MCP] Refreshing MCP discovery...")
	m.Discover()
}

// discoverExternalMCPs discovers MCP servers via npx
func (m *MCPManager) discoverExternalMCPs() {
	commonServers := []config.MCPServer{
		{
			Name:    "filesystem",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/workspace"},
		},
		{
			Name:    "web-search",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-brave-search"},
		},
		{
			Name:    "github",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		},
		{
			Name:    "postgres",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-postgres", "postgresql://localhost/db"},
		},
	}

	for _, server := range commonServers {
		go m.connectServer(server)
	}
}

// discoverFromEnvironment discovers MCPs from environment variables
// Format: MCP_SERVER_<NAME>=command:arg1:arg2:...
func (m *MCPManager) discoverFromEnvironment() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "MCP_SERVER_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			name := strings.ToLower(strings.TrimPrefix(parts[0], "MCP_SERVER_"))
			configParts := strings.Split(parts[1], ":")
			if len(configParts) < 1 {
				continue
			}

			server := config.MCPServer{
				Name:    name,
				Command: configParts[0],
				Args:    configParts[1:],
			}

			log.Printf("[MCP] Found environment MCP: %s\n", name)
			go m.connectServer(server)
		}
	}
}

// discoverFromConfig discovers MCPs from configuration files
func (m *MCPManager) discoverFromConfig() {
	log.Println("[MCP] ========== discoverFromConfig called ==========")
	// First, load from mcpservers.yaml (application config)
	m.discoverFromMCPServersConfig()

	// Then, load from JSON config files (legacy/external config)
	configPaths := []string{
		"./config/mcp-servers.json",
		"/etc/mcp/servers.json",
		os.ExpandEnv("$HOME/.mcp/servers.json"),
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			// Try parsing as array first
			var servers []config.MCPServer
			if err := json.Unmarshal(data, &servers); err == nil {
				log.Printf("[MCP] Loaded %d MCPs from %s\n", len(servers), path)
				for _, server := range servers {
					go m.connectServer(server)
				}
				return
			}

			// Try parsing as object with "servers" field
			var configData struct {
				Servers []config.MCPServer `json:"servers"`
			}
			if err := json.Unmarshal(data, &configData); err == nil {
				log.Printf("[MCP] Loaded %d MCPs from %s\n", len(configData.Servers), path)
				for _, server := range configData.Servers {
					go m.connectServer(server)
				}
				return
			}

			log.Printf("[MCP] Failed to parse config %s: invalid format\n", path)
		}
	}
	log.Println("[MCP] ========== discoverFromConfig completed ==========")
}

// discoverFromMCPServersConfig loads enabled MCP servers from mcpservers.json
func (m *MCPManager) discoverFromMCPServersConfig() {
	log.Println("[MCP] Loading MCP servers from config...")
	cfg := config.GetMCPServersConfig()
	if cfg == nil || cfg.Servers == nil {
		log.Println("[MCP] No MCP servers config loaded from mcpservers.json")
		return
	}
	log.Printf("[MCP] Found %d MCP servers in config\n", len(cfg.Servers))

	// 创建验证器
	validator := config.NewMCPServerValidator()

	enabledCount := 0
	validationErrors := 0

	for name, server := range cfg.Servers {
		if !server.Enabled {
			continue
		}

		// Skip builtin servers (they are handled separately)
		if server.Type == "builtin" {
			log.Printf("[MCP] Skipping builtin server: %s (type: %s)\n", name, server.Type)
			continue
		}

		// 验证服务器配置
		server.Name = name
		errors := validator.ValidateServer(&server)

		if len(errors) > 0 {
			validationErrors++
			log.Printf("[MCP] Validation errors for server %s:\n", name)
			for _, err := range errors {
				log.Printf("[MCP]   - %s: %s (%s)\n", err.Field, err.Message, err.Severity)
			}

			// 如果有严重错误，跳过此服务器
			hasCriticalError := false
			for _, err := range errors {
				if err.Severity == "error" {
					hasCriticalError = true
					break
				}
			}

			if hasCriticalError {
				log.Printf("[MCP] Skipping server %s due to critical validation errors\n", name)
				continue
			}

			log.Printf("[MCP] Proceeding with server %s despite warnings\n", name)
		}

		log.Printf("[MCP] Loading enabled MCP server from config: %s (type: %s)\n", name, server.Type)
		log.Printf("[MCP] Starting goroutine to connect to server: %s\n", name)
		go func(s config.MCPServer) {
			log.Printf("[MCP] Goroutine started for server: %s\n", s.Name)
			// Add a small delay to ensure goroutine is scheduled
			time.Sleep(100 * time.Millisecond)
			m.connectServer(s)
			log.Printf("[MCP] Goroutine completed for server: %s\n", s.Name)
		}(server)
		enabledCount++
	}

	if enabledCount > 0 {
		log.Printf("[MCP] Loaded %d enabled MCP servers from mcpservers.json\n", enabledCount)
	}

	if validationErrors > 0 {
		log.Printf("[MCP] Found validation issues in %d server(s)\n", validationErrors)
	}
}

// convertToMCPServerWithDocs converts an MCPServer to MCPServerWithDocs
func convertToMCPServerWithDocs(server config.MCPServer) *config.MCPServerWithDocs {
	return &config.MCPServerWithDocs{
		MCPServer: server,
	}
}

// discoverFromDocker discovers MCPs running as Docker containers
func (m *MCPManager) discoverFromDocker() {
	// Check if Docker socket is available
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return // Docker not available
	}

	// This would require Docker client library
	// For now, just log that we checked
	log.Println("[MCP] Docker socket available, but Docker discovery not implemented")
}

// addMockFilesystemMCP adds a built-in filesystem MCP that doesn't require external processes
func (m *MCPManager) addMockFilesystemMCP() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if _, exists := m.servers["filesystem-local"]; exists {
		return
	}

	// Add a mock server that uses local filesystem operations
	m.servers["filesystem-local"] = &config.MCPServer{
		Name:      "filesystem-local",
		Connected: true,
		Tools: []mcp.Tool{
			mcp.NewTool("read_file",
				mcp.WithDescription("Read a file from the local filesystem"),
				mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
			),
			mcp.NewTool("write_file",
				mcp.WithDescription("Write a file to the local filesystem"),
				mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
				mcp.WithString("content", mcp.Required(), mcp.Description("Content to write")),
			),
			mcp.NewTool("list_directory",
				mcp.WithDescription("List files in a directory"),
				mcp.WithString("path", mcp.Required(), mcp.Description("Path to the directory")),
			),
		},
	}

	log.Println("[MCP] Added built-in filesystem-local server")
}

// AddBuiltInMCPs adds built-in MCP servers that don't require external processes
func (m *MCPManager) AddBuiltInMCPs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Terminal MCP - executes shell commands
	if _, exists := m.servers["terminal"]; !exists {
		m.servers["terminal"] = &config.MCPServer{
			Name:      "terminal",
			Connected: true,
			Tools: []mcp.Tool{
				mcp.NewTool("execute_command",
					mcp.WithDescription("Execute a shell command"),
					mcp.WithString("command", mcp.Required(), mcp.Description("Command to execute")),
					mcp.WithString("working_dir", mcp.Description("Working directory")),
				),
			},
		}
		log.Println("[MCP] Added built-in terminal server")
	}

	// Code Analysis MCP
	if _, exists := m.servers["code-analysis"]; !exists {
		m.servers["code-analysis"] = &config.MCPServer{
			Name:      "code-analysis",
			Connected: true,
			Tools: []mcp.Tool{
				mcp.NewTool("analyze_code",
					mcp.WithDescription("Analyze code for issues"),
					mcp.WithString("code", mcp.Required(), mcp.Description("Code to analyze")),
					mcp.WithString("language", mcp.Required(), mcp.Description("Programming language")),
				),
				mcp.NewTool("suggest_improvements",
					mcp.WithDescription("Suggest code improvements"),
					mcp.WithString("code", mcp.Required(), mcp.Description("Code to improve")),
					mcp.WithString("language", mcp.Required(), mcp.Description("Programming language")),
				),
			},
		}
		log.Println("[MCP] Added built-in code-analysis server")
	}

	// Search MCP
	if _, exists := m.servers["search"]; !exists {
		m.servers["search"] = &config.MCPServer{
			Name:      "search",
			Connected: true,
			Tools: []mcp.Tool{
				mcp.NewTool("web_search",
					mcp.WithDescription("Search the web for information"),
					mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
					mcp.WithNumber("num_results", mcp.Description("Number of results")),
				),
			},
		}
		log.Println("[MCP] Added built-in search server")
	}
}

// connectServer attempts to connect to an MCP server
func (m *MCPManager) connectServer(server config.MCPServer) {
	// Use fmt.Println to ensure output is captured
	fmt.Println("[MCP] ========== connectServer called for server:", server.Name, "==========")

	// First check if already connected (with read lock)
	m.mu.RLock()
	if existing, ok := m.servers[server.Name]; ok && existing.Connected {
		log.Printf("[MCP] Server %s already connected, skipping\n", server.Name)
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	// 检查熔断器状态
	cb := m.getOrCreateCircuitBreaker(server.Name)
	if cb.GetState() == StateOpen {
		log.Printf("[MCP] Circuit breaker is open for %s, skipping connection attempt", server.Name)
		return
	}

	// 验证配置
	validationResult := m.configValidator.ValidateBeforeConnect(&server)

	// 记录验证结果
	if !validationResult.IsValid {
		log.Printf("[MCP] Config validation failed for %s:", server.Name)
		for _, err := range validationResult.Errors {
			log.Printf("[MCP]   - %s", err.String())
		}

		// 记录到监控系统
		m.monitor.RecordValidationFailure(server.Name, validationResult)

		// 向后兼容：验证失败不阻止连接尝试，但记录警告
		log.Printf("[MCP] Warning: Proceeding with connection despite validation failures (backward compatibility)")
	} else if validationResult.HasWarnings() {
		log.Printf("[MCP] Config validation warnings for %s:", server.Name)
		for _, warning := range validationResult.Warnings {
			log.Printf("[MCP]   - %s", warning)
		}
	}

	log.Printf("[MCP] ========== Connecting to server: %s ==========\n", server.Name)
	log.Printf("[MCP] Type: %s, Command: %s, Args: %v, Env: %v, URL: %s\n",
		server.Type, server.Command, server.Args, server.Env, server.URL)

	// Create client with environment variables
	var cli *client.Client
	var err error

	// Check if we need to set environment variables
	// 对于Playwright，不设置环境变量（像MCPtest一样）
	if len(server.Env) > 0 && server.Name != "playwright" {
		log.Printf("[MCP] Setting environment variables for %s: %v", server.Name, server.Env)
		// Set environment variables at process level before creating client
		// Note: Environment variable names should be uppercase
		for k, v := range server.Env {
			// Convert key to uppercase for environment variables
			envKey := strings.ToUpper(k)
			os.Setenv(envKey, v)
			log.Printf("[MCP] Set environment variable: %s=%s", envKey, v)
		}
	} else if server.Name == "playwright" {
		log.Printf("[MCP] Skipping environment variable setting for playwright (MCPtest style)")
	}

	// Determine transport type based on server type
	// Note: Currently only stdio transport is supported by mcp-go
	if server.Command != "" {
		// Stdio transport
		log.Printf("[MCP] Creating stdio MCP client for %s with command: %s, args: %v", server.Name, server.Command, server.Args)

		// Use NewStdioMCPClient as shown in the official example
		// Build command line arguments - split command and args properly
		allArgs := make([]string, 0, 1+len(server.Args))
		allArgs = append(allArgs, server.Command)
		allArgs = append(allArgs, server.Args...)

		log.Printf("[MCP] Creating stdio MCP client with full command: %v", allArgs)

		// Extract command and arguments properly for NewStdioMCPClient
		if len(allArgs) == 0 {
			log.Printf("[MCP] ERROR: No command provided for %s\n", server.Name)
			server.LastError = "no command provided"
			m.mu.Lock()
			m.servers[server.Name] = &server
			m.mu.Unlock()
			// 记录熔断器失败
			cb.Execute(func() error { return fmt.Errorf("no command provided") })
			return
		}

		cmd := allArgs[0]
		args := []string{}
		if len(allArgs) > 1 {
			args = allArgs[1:]
		}

		// Prepare environment variables
		// For playwright-mcp, we need to pass nil like in the successful example
		var env []string
		if len(server.Env) > 0 {
			// Convert server.Env map to string slice
			for k, v := range server.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		// Use the transport.NewStdio approach with correct signature: command, env, args...
		// For playwright, use environment variables
		var t *transport.Stdio

		// 特殊处理Playwright：完全复制MCPtest的方式
		if server.Name == "playwright" {
			log.Printf("[MCP] Using EXACT MCPtest-style connection for playwright")
			// 完全复制MCPtest：transport.NewStdio("playwright-mcp", nil)
			// MCPtest没有传递环境变量，环境变量在Docker容器级别设置
			t = transport.NewStdio("playwright-mcp", nil)
		} else {
			t = transport.NewStdio(cmd, env, args...)
		}

		cli = client.NewClient(t)
	} else {
		log.Printf("[MCP] ERROR: No valid connection method for %s (type: %s, command: %s)\n",
			server.Name, server.Type, server.Command)
		server.LastError = "no valid connection method (command is empty)"
		m.mu.Lock()
		m.servers[server.Name] = &server
		m.mu.Unlock()
		// 记录熔断器失败
		cb.Execute(func() error { return fmt.Errorf("no valid connection method") })
		return
	}

	// Start the client
	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startCancel()

	if err := cli.Start(startCtx); err != nil {
		log.Printf("[MCP] Failed to start client for %s: %v\n", server.Name, err)
		server.LastError = err.Error()
		m.mu.Lock()
		m.servers[server.Name] = &server
		m.mu.Unlock()
		// 记录熔断器失败
		cb.Execute(func() error { return err })
		return
	}

	log.Printf("[MCP] Successfully created and started client for %s", server.Name)

	// Check if client is nil
	if cli == nil {
		log.Printf("[MCP] ERROR: Client is nil for %s\n", server.Name)
		server.LastError = "client is nil"
		m.mu.Lock()
		m.servers[server.Name] = &server
		m.mu.Unlock()
		// 记录熔断器失败
		cb.Execute(func() error { return fmt.Errorf("client is nil") })
		return
	}

	// Initialize the server with proper error handling and debugging
	log.Printf("[MCP] Initializing %s...\n", server.Name)

	// Create proper InitializeRequest with required parameters
	initReq := mcp.InitializeRequest{}

	// For context7 and playwright, we may need to use different protocol version or parameters
	if server.Name == "context7" || server.Name == "playwright" {
		// 完全复制MCPtest的初始化参数
		initReq.Params.ProtocolVersion = "2024-11-05" // MCPtest使用的版本
		initReq.Params.ClientInfo = mcp.Implementation{
			Name:    "playwright-go-client", // MCPtest使用的名称
			Version: "1.0.0",
		}
		// Use minimal capabilities for context7 and playwright
		initReq.Params.Capabilities = mcp.ClientCapabilities{}

		// For context7 and playwright, use unified timeout
		initTimeout := 15 * time.Second

		// Special handling for Context7 - extended timeout
		if server.Name == "context7" {
			initTimeout = 60 * time.Second
			log.Printf("[MCP] Initializing context7 with extended timeout (%v)...", initTimeout)
		} else {
			log.Printf("[MCP] Initializing %s with unified timeout (%v)...", server.Name, initTimeout)
		}

		initCtx, initCancel := context.WithTimeout(context.Background(), initTimeout)
		defer initCancel()

		_, err = cli.Initialize(initCtx, initReq)

		if err != nil {
			log.Printf("[MCP] Initialization failed for %s: %v", server.Name, err)

			// 对于Context7，尝试重试最多3次
			if server.Name == "context7" {
				log.Printf("[MCP] Context7 initialization failed, will retry up to 3 times")

				// 重试逻辑
				var retryErr error
				for attempt := 1; attempt <= 3; attempt++ {
					if attempt > 1 {
						log.Printf("[MCP] Context7 retry attempt %d/3", attempt)
						time.Sleep(1 * time.Second)
					}

					retryCtx, retryCancel := context.WithTimeout(context.Background(), initTimeout)
					_, retryErr = cli.Initialize(retryCtx, initReq)
					retryCancel()

					if retryErr == nil {
						log.Printf("[MCP] Context7 initialization successful on attempt %d", attempt)
						err = nil
						break
					}

					log.Printf("[MCP] Context7 initialization failed on attempt %d: %v", attempt, retryErr)
				}

				if retryErr != nil {
					err = retryErr
					log.Printf("[MCP] Context7 initialization failed after 3 attempts: %v", err)
				}
			}

			if err != nil {
				// 如果初始化失败（包括重试后），跳过工具发现
				log.Printf("[MCP] Skipping tool discovery for %s due to initialization failure", server.Name)

				// 存储空工具并标记为连接但有错误
				server.Tools = []mcp.Tool{}
				server.LastError = fmt.Sprintf("initialization failed: %v", err)
				server.Client = cli
				server.Connected = true // 标记为连接但有初始化错误

				m.mu.Lock()
				m.servers[server.Name] = &server
				m.mu.Unlock()

				// 存储连接到资源管理器
				if err := m.resourceManager.StoreConnection(server.Name, cli); err != nil {
					log.Printf("[MCP] Warning: Failed to store connection for %s in resource manager: %v", server.Name, err)
				}

				// 记录熔断器成功（虽然初始化失败，但连接已建立）
				cb.Execute(func() error { return nil })

				log.Printf("[MCP] Connected to server %s with initialization error: %v", server.Name, err)
				return
			}
		}

		log.Printf("[MCP] Initialization successful for %s", server.Name)
	} else {
		// Standard initialization for other servers
		initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initReq.Params.ClientInfo = mcp.Implementation{
			Name:    "DouBao MCP Client",
			Version: "1.0.0",
		}
		initReq.Params.Capabilities = mcp.ClientCapabilities{}

		// Try initialization with unified timeout
		initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer initCancel()

		_, err = cli.Initialize(initCtx, initReq)

		if err != nil {
			log.Printf("[MCP] Initialization failed for %s: %v", server.Name, err)
			// If initialization fails, we should not try to get tools
			log.Printf("[MCP] Skipping tool discovery for %s due to initialization failure", server.Name)

			// Store empty tools and mark as connected but with error
			server.Tools = []mcp.Tool{}
			server.LastError = fmt.Sprintf("initialization failed: %v", err)
			server.Client = cli
			server.Connected = true // Mark as connected but with initialization error

			m.mu.Lock()
			m.servers[server.Name] = &server
			m.mu.Unlock()

			// 存储连接到资源管理器
			if err := m.resourceManager.StoreConnection(server.Name, cli); err != nil {
				log.Printf("[MCP] Warning: Failed to store connection for %s in resource manager: %v", server.Name, err)
			}

			// 记录熔断器成功（虽然初始化失败，但连接已建立）
			cb.Execute(func() error { return nil })

			// Log connection with error
			log.Printf("[MCP] Connected to server %s with initialization error: %v", server.Name, err)
			return
		} else {
			log.Printf("[MCP] Initialization successful for %s", server.Name)
		}
	}

	// List tools with improved error handling
	// Only try to get tools if client is properly initialized
	log.Printf("[MCP] Getting tools for %s...\n", server.Name)

	var toolsResult *mcp.ListToolsResult
	var toolsErr error

	// Create context for listing tools with appropriate timeout
	// Playwright and context7 may need longer timeouts
	var toolsTimeout time.Duration
	if server.Name == "playwright" || server.Name == "context7" {
		toolsTimeout = 30 * time.Second
		log.Printf("[MCP] Using longer timeout (%v) for %s tool listing", toolsTimeout, server.Name)
	} else {
		toolsTimeout = 15 * time.Second
	}

	toolsCtx, toolsCancel := context.WithTimeout(context.Background(), toolsTimeout)
	defer toolsCancel()

	toolsResult, toolsErr = cli.ListTools(toolsCtx, mcp.ListToolsRequest{})

	if toolsErr != nil {
		// Tool discovery failure should not break the entire connection
		// Some servers may have tools but discovery fails due to timeout or other issues
		log.Printf("[MCP] Warning: Tool discovery failed for %s: %v", server.Name, toolsErr)
		log.Printf("[MCP] Continuing with connection despite tool discovery failure for %s", server.Name)

		// Store empty tools but keep the connection
		server.Tools = []mcp.Tool{}
		server.LastError = fmt.Sprintf("tool discovery failed: %v", toolsErr)
	} else {
		// Successfully got tools
		server.Tools = toolsResult.Tools
		log.Printf("[MCP] Successfully discovered %d tools for %s", len(toolsResult.Tools), server.Name)
	}

	// 存储连接到资源管理器
	if err := m.resourceManager.StoreConnection(server.Name, cli); err != nil {
		log.Printf("[MCP] Warning: Failed to store connection for %s in resource manager: %v", server.Name, err)
	}

	// Update server - mark as connected even if tool discovery failed
	server.Client = cli
	server.Connected = true
	m.mu.Lock()
	m.servers[server.Name] = &server
	m.mu.Unlock()

	// 记录连接变化
	m.monitor.RecordConnectionChange(1)

	// 记录熔断器成功
	cb.Execute(func() error { return nil })

	// Log tool names for debugging
	if server.Name == "context7" && toolsResult != nil && len(toolsResult.Tools) > 0 {
		log.Printf("[MCP] Context7 tools found:")
		for i, tool := range toolsResult.Tools {
			log.Printf("[MCP]   Tool %d: %s - %s", i+1, tool.Name, tool.Description)
		}
	}

	// Safely log connection success, handling nil toolsResult
	toolCount := 0
	if toolsResult != nil {
		toolCount = len(toolsResult.Tools)
	}
	log.Printf("[MCP] Successfully connected to server: %s with %d tools\n", server.Name, toolCount)

	// Note: Tool discovery (DiscoverTools) is now triggered on demand
	// rather than immediately after connection to avoid overloading the server
	// The initial tool listing during connection is sufficient for basic operation
}

// ListConnected returns all connected MCP servers
func (m *MCPManager) ListConnected() []config.MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connected []config.MCPServer
	for _, server := range m.servers {
		if server.Connected {
			connected = append(connected, *server)
		}
	}
	return connected
}

// ListAll returns all MCP servers including disconnected ones
func (m *MCPManager) ListAll() []config.MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []config.MCPServer
	for _, server := range m.servers {
		all = append(all, *server)
	}
	return all
}

// GetServer gets a specific MCP server
func (m *MCPManager) GetServer(name string) (*config.MCPServer, bool) {
	m.mu.RLock()
	server, ok := m.servers[name]
	m.mu.RUnlock()

	// 如果服务器未找到，尝试发现它
	if !ok {
		log.Printf("[MCP] Server %s not found, triggering discovery...", name)
		m.Discover()

		// 发现后再次尝试
		m.mu.RLock()
		server, ok = m.servers[name]
		m.mu.RUnlock()

		if ok {
			log.Printf("[MCP] Server %s found after discovery", name)
		} else {
			log.Printf("[MCP] Server %s still not found after discovery", name)
		}
	}

	return server, ok
}

// Close 关闭所有连接并清理资源
func (m *MCPManager) Close() error {
	log.Println("[MCP] Closing MCP manager and cleaning up resources...")

	// Stop health check goroutine
	if m.healthCheckCancel != nil {
		m.healthCheckCancel()
		m.healthCheckWg.Wait()
		log.Println("[MCP] Health check goroutine stopped")
	}

	// 关闭所有连接
	if err := m.resourceManager.CloseAll(); err != nil {
		log.Printf("[MCP] Error closing resource manager: %v", err)
	}

	// 清理服务器映射
	m.mu.Lock()
	for name, server := range m.servers {
		if server.Client != nil {
			server.Client.Close()
		}
		delete(m.servers, name)
	}
	m.mu.Unlock()

	// 记录连接变化
	m.monitor.RecordConnectionChange(-len(m.servers))

	log.Println("[MCP] MCP manager closed successfully")
	return nil
}

// GetResourceStats 获取资源统计信息
func (m *MCPManager) GetResourceStats() ResourceStats {
	return m.resourceManager.GetStats()
}

// GetMonitorStats 获取监控统计信息
func (m *MCPManager) GetMonitorStats() MonitorStats {
	return m.monitor.GetStats()
}

// GetCircuitBreakerState 获取熔断器状态
func (m *MCPManager) GetCircuitBreakerState(serverName string) CircuitState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cb, exists := m.circuitBreakers[serverName]; exists {
		return cb.GetState()
	}
	return StateClosed
}

// GetAllServers gets all MCP servers
func (m *MCPManager) GetAllServers() map[string]*config.MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid external modifications
	servers := make(map[string]*config.MCPServer)
	for name, server := range m.servers {
		servers[name] = server
	}
	return servers
}

// CloseServer closes a specific MCP server
func (m *MCPManager) CloseServer(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[serverName]
	if !ok {
		return fmt.Errorf("server %s not found", serverName)
	}

	if server.Client != nil {
		log.Printf("[MCP] Closing server: %s", serverName)
		server.Client.Close()
		server.Client = nil
	}

	server.Connected = false
	m.servers[serverName] = server
	log.Printf("[MCP] Server %s closed successfully", serverName)
	return nil
}

// CloseAllServers closes all MCP servers
func (m *MCPManager) CloseAllServers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[MCP] Closing all MCP servers...")
	for name, server := range m.servers {
		if server.Client != nil {
			log.Printf("[MCP] Closing server: %s", name)
			server.Client.Close()
			server.Client = nil
			server.Connected = false
			m.servers[name] = server
		}
	}
	log.Printf("[MCP] All MCP servers closed")
}

// GetAvailableTools returns a list of all available tools from connected MCP servers
func (m *MCPManager) GetAvailableTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []string

	for serverName, server := range m.servers {
		if server.Connected && server.Client != nil {
			// Check if we have documentation for this server
			if doc, exists := m.documentations[serverName]; exists {
				for _, tool := range doc.Tools {
					tools = append(tools, fmt.Sprintf("%s/%s", serverName, tool.Name))
				}
			} else {
				// Fallback to server tools if no documentation
				for _, tool := range server.Tools {
					tools = append(tools, fmt.Sprintf("%s/%s", serverName, tool.Name))
				}
			}
		}
	}

	return tools
}

// DiscoverTools discovers and documents tools for a specific MCP server
func (m *MCPManager) DiscoverTools(serverName string) (*config.MCPServerDocumentation, error) {
	m.mu.Lock()
	server, exists := m.servers[serverName]
	m.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	if !server.Connected || server.Client == nil {
		return nil, fmt.Errorf("server %s is not connected", serverName)
	}

	log.Printf("[MCP] Discovering tools for server: %s", serverName)

	// Use appropriate timeout for tool discovery
	// Playwright and context7 may need longer timeouts
	var timeout time.Duration
	if serverName == "playwright" || serverName == "context7" {
		timeout = 30 * time.Second // Longer timeout for complex servers
		log.Printf("[MCP] Using longer timeout (%v) for %s tool discovery", timeout, serverName)
	} else {
		timeout = 15 * time.Second // Standard timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// List tools from the server
	toolsResult, err := server.Client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		// Tool discovery failure should not break the server connection
		// Log the error but return empty documentation
		log.Printf("[MCP] Warning: Tool discovery failed for %s: %v", serverName, err)
		log.Printf("[MCP] Returning empty documentation for %s", serverName)

		// Return empty documentation instead of error
		return &config.MCPServerDocumentation{
			ServerName:     serverName,
			ServerVersion:  "1.0.0",
			Description:    server.Name,
			LastUpdated:    time.Now(),
			DiscoveryCount: 0,
			Tools:          []config.MCPTool{},
			Overview:       fmt.Sprintf("Tool discovery failed: %v", err),
			UseCases:       []string{},
			Summary:        "Tool discovery failed, server may still be functional",
		}, nil
	}

	// Create documentation
	doc := &config.MCPServerDocumentation{
		ServerName:     serverName,
		ServerVersion:  "1.0.0", // Default version
		Description:    server.Name,
		LastUpdated:    time.Now(),
		DiscoveryCount: 1,
	}

	// Process each tool
	for _, tool := range toolsResult.Tools {
		mcpTool := config.MCPTool{
			Name:           tool.Name,
			Description:    tool.Description,
			LastDiscovered: time.Now(),
		}

		// Extract input schema if available
		// Note: mcp.ToolInputSchema is an interface{}
		// Use reflection to check if the value is not nil
		inputSchemaValue := reflect.ValueOf(tool.InputSchema)
		if inputSchemaValue.IsValid() {
			// Check if the value is nil using a safer approach
			isNil := false
			switch inputSchemaValue.Kind() {
			case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
				isNil = inputSchemaValue.IsNil()
			default:
				// For non-nilable types, check if it's the zero value
				isNil = inputSchemaValue.IsZero()
			}

			if !isNil {
				// Try to marshal the input schema
				if schemaJSON, err := json.Marshal(tool.InputSchema); err == nil && string(schemaJSON) != "null" {
					var schemaMap map[string]interface{}
					if err := json.Unmarshal(schemaJSON, &schemaMap); err == nil {
						mcpTool.InputSchema = schemaMap
						mcpTool.InputParams = config.ExtractInputSchema(schemaMap)
					}
				}
			}
		}

		// Generate usage scenarios based on tool description
		mcpTool.UsageScenarios = m.generateUsageScenarios(tool.Name, tool.Description)

		doc.Tools = append(doc.Tools, mcpTool)
		log.Printf("[MCP] Discovered tool: %s - %s", tool.Name, tool.Description)
	}

	// Generate server overview and summary
	doc.Overview = m.generateServerOverview(serverName, doc.Tools)
	doc.UseCases = m.generateUseCases(doc.Tools)
	doc.Summary = doc.GenerateSummary()

	// Store documentation separately
	m.mu.Lock()
	m.documentations[serverName] = doc
	m.mu.Unlock()

	// Save documentation to file
	if err := m.saveDocumentation(serverName, doc); err != nil {
		log.Printf("[MCP] Warning: Failed to save documentation for %s: %v", serverName, err)
	}

	log.Printf("[MCP] Successfully discovered %d tools for %s", len(doc.Tools), serverName)
	return doc, nil
}

// generateUsageScenarios generates usage scenarios based on tool name and description
func (m *MCPManager) generateUsageScenarios(toolName, description string) []string {
	scenarios := []string{}

	// Common patterns
	lowerName := strings.ToLower(toolName)
	lowerDesc := strings.ToLower(description)

	if strings.Contains(lowerName, "query") || strings.Contains(lowerDesc, "query") {
		scenarios = append(scenarios, "When user needs to search or query information")
	}
	if strings.Contains(lowerName, "resolve") || strings.Contains(lowerDesc, "resolve") {
		scenarios = append(scenarios, "When user needs to find or identify something")
	}
	if strings.Contains(lowerName, "search") || strings.Contains(lowerDesc, "search") {
		scenarios = append(scenarios, "When user needs to search for information")
	}
	if strings.Contains(lowerName, "read") || strings.Contains(lowerDesc, "read") {
		scenarios = append(scenarios, "When user needs to read or view content")
	}
	if strings.Contains(lowerName, "write") || strings.Contains(lowerDesc, "write") {
		scenarios = append(scenarios, "When user needs to create or modify content")
	}
	if strings.Contains(lowerName, "execute") || strings.Contains(lowerDesc, "execute") {
		scenarios = append(scenarios, "When user needs to run a command or operation")
	}

	// Add generic scenario if none found
	if len(scenarios) == 0 {
		scenarios = append(scenarios, "When the task matches the tool's purpose")
	}

	return scenarios
}

// generateServerOverview generates an overview for the server
func (m *MCPManager) generateServerOverview(serverName string, tools []config.MCPTool) string {
	overview := fmt.Sprintf("The %s MCP server provides %d tools for various tasks.\n\n", serverName, len(tools))
	overview += "Available tools:\n"

	for i, tool := range tools {
		overview += fmt.Sprintf("%d. %s: %s\n", i+1, tool.Name, tool.Description)
	}

	overview += "\nThis server is designed to help with tasks related to "

	// Determine server type based on name
	switch serverName {
	case "context7":
		overview += "documentation lookup and library information retrieval."
	case "playwright":
		overview += "browser automation and web interaction."
	case "brave-search":
		overview += "web search and information retrieval."
	case "filesystem":
		overview += "file system operations and file management."
	case "terminal":
		overview += "command execution and system operations."
	default:
		overview += "various automation and integration tasks."
	}

	return overview
}

// generateUseCases generates use cases based on available tools
func (m *MCPManager) generateUseCases(tools []config.MCPTool) []string {
	useCases := []string{}

	for _, tool := range tools {
		lowerName := strings.ToLower(tool.Name)
		lowerDesc := strings.ToLower(tool.Description)

		if strings.Contains(lowerName, "search") || strings.Contains(lowerDesc, "search") {
			useCases = append(useCases, "Information retrieval and search tasks")
		}
		if strings.Contains(lowerName, "query") || strings.Contains(lowerDesc, "query") {
			useCases = append(useCases, "Data querying and information lookup")
		}
		if strings.Contains(lowerName, "read") || strings.Contains(lowerDesc, "read") {
			useCases = append(useCases, "Reading and viewing content")
		}
		if strings.Contains(lowerName, "write") || strings.Contains(lowerDesc, "write") {
			useCases = append(useCases, "Content creation and modification")
		}
		if strings.Contains(lowerName, "execute") || strings.Contains(lowerDesc, "execute") {
			useCases = append(useCases, "Command execution and automation")
		}
	}

	// Remove duplicates
	uniqueCases := make(map[string]bool)
	var result []string
	for _, uc := range useCases {
		if !uniqueCases[uc] {
			uniqueCases[uc] = true
			result = append(result, uc)
		}
	}

	return result
}

// saveDocumentation saves documentation to file
func (m *MCPManager) saveDocumentation(serverName string, doc *config.MCPServerDocumentation) error {
	// Ensure directory exists
	if err := os.MkdirAll(m.docsPath, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Convert to JSON
	docJSON, err := doc.ConvertToJSON()
	if err != nil {
		return err
	}

	// Save to file
	filename := filepath.Join(m.docsPath, fmt.Sprintf("%s_docs.json", serverName))
	if err := os.WriteFile(filename, []byte(docJSON), 0644); err != nil {
		return fmt.Errorf("failed to write documentation file: %w", err)
	}

	log.Printf("[MCP] Documentation saved to: %s", filename)
	return nil
}

// loadDocumentation loads documentation from file
func (m *MCPManager) loadDocumentation(serverName string) (*config.MCPServerDocumentation, error) {
	filename := filepath.Join(m.docsPath, fmt.Sprintf("%s_docs.json", serverName))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation file: %w", err)
	}

	var doc config.MCPServerDocumentation
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse documentation: %w", err)
	}

	return &doc, nil
}

// GetServerDocumentation returns documentation for a server
func (m *MCPManager) GetServerDocumentation(serverName string) (*config.MCPServerDocumentation, error) {
	m.mu.RLock()
	server, exists := m.servers[serverName]
	doc, docExists := m.documentations[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	// If documentation exists and server is connected, return it
	if docExists && server.Connected {
		return doc, nil
	}

	// Otherwise, discover tools
	return m.DiscoverTools(serverName)
}

// GetAllServerSummaries returns summaries for all servers
func (m *MCPManager) GetAllServerSummaries() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make(map[string]string)

	for serverName, server := range m.servers {
		if server.Connected {
			if doc, exists := m.documentations[serverName]; exists {
				summaries[serverName] = doc.Summary
			} else {
				// Create a basic summary from server info
				summary := fmt.Sprintf("MCP Server: %s\n", serverName)
				if server.Name != "" {
					summary += fmt.Sprintf("Description: %s\n", server.Name)
				}
				summary += fmt.Sprintf("Tools available: %d\n", len(server.Tools))
				for i, tool := range server.Tools {
					summary += fmt.Sprintf("  %d. %s: %s\n", i+1, tool.Name, tool.Description)
				}
				summaries[serverName] = summary
			}
		}
	}

	return summaries
}

// ConnectToServer connects to a specific MCP server if not already connected
func (m *MCPManager) ConnectToServer(serverName string) error {
	// Check if server is already connected
	server, ok := m.GetServer(serverName)
	if !ok {
		return fmt.Errorf("MCP server %s not found in configuration", serverName)
	}

	if server.Connected {
		return nil // Already connected
	}

	// 检查熔断器
	cb := m.getOrCreateCircuitBreaker(serverName)
	err := cb.Execute(func() error {
		return m.connectToServerInternal(serverName)
	})

	if err != nil {
		m.monitor.RecordRequest(serverName, "connect", 0, err)
		return err
	}

	return nil
}

// connectToServerInternal 内部连接服务器方法
func (m *MCPManager) connectToServerInternal(serverName string) error {
	start := time.Now()
	defer func() {
		m.monitor.RecordRequest(serverName, "connect", time.Since(start), nil)
	}()

	// Get the original configuration to reconnect
	cfg := config.GetMCPServersConfig()
	if cfg == nil {
		return fmt.Errorf("no MCP servers config found")
	}

	originalServer, exists := cfg.Servers[serverName]
	if !exists {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Connect to this specific server
	originalServer.Name = serverName
	m.connectServer(originalServer)

	// Wait for connection to establish - context7 needs about 2 seconds
	// Use longer timeout for context7
	var waitTime time.Duration
	if serverName == "context7" {
		waitTime = 5 * time.Second
		log.Printf("[MCP] Waiting %v for context7 connection...", waitTime)
	} else {
		waitTime = 2 * time.Second
	}
	time.Sleep(waitTime)

	// Check if connection was successful
	updatedServer, ok := m.GetServer(serverName)
	if !ok || !updatedServer.Connected {
		return fmt.Errorf("failed to connect to server %s after %v wait", serverName, waitTime)
	}

	return nil
}

// getOrCreateCircuitBreaker 获取或创建熔断器
func (m *MCPManager) getOrCreateCircuitBreaker(serverName string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.circuitBreakers[serverName]; exists {
		return cb
	}

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig.FailureThreshold, DefaultCircuitBreakerConfig.ResetTimeout)
	m.circuitBreakers[serverName] = cb
	return cb
}

// CallTool calls a tool on an MCP server
func (m *MCPManager) CallTool(serverName string, toolName string, args map[string]interface{}) (string, error) {
	server, ok := m.GetServer(serverName)
	if !ok {
		return "", fmt.Errorf("MCP server %s not found", serverName)
	}

	// Check connection health before calling tool
	// 对于Playwright，创建全新的连接（单次使用模式）
	if serverName == "playwright" {
		log.Printf("[MCP] For playwright, creating fresh connection for single-use...")

		// 1. 完全清理旧连接
		if server.Client != nil {
			server.Client.Close()
		}
		// 从管理器中移除
		m.mu.Lock()
		delete(m.servers, serverName)
		m.mu.Unlock()

		// 2. 从原始配置创建全新的连接
		cfg := config.GetMCPServersConfig()
		if cfg == nil {
			return "", fmt.Errorf("no MCP servers config found for playwright")
		}

		originalServer, exists := cfg.Servers[serverName]
		if !exists {
			return "", fmt.Errorf("playwright server not found in configuration")
		}

		// 3. 直接连接（绕过管理器缓存）
		originalServer.Name = serverName

		// 创建全新的客户端
		var cli *client.Client
		var err error

		// 使用MCPtest的方式：transport.NewStdio("playwright-mcp", nil)
		t := transport.NewStdio("playwright-mcp", nil)
		cli = client.NewClient(t)

		// 启动客户端
		startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer startCancel()

		if err := cli.Start(startCtx); err != nil {
			log.Printf("[MCP] Failed to start fresh playwright client: %v", err)
			return "", fmt.Errorf("failed to start fresh playwright client: %v", err)
		}

		// 4. 立即初始化
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = "2024-11-05"
		initReq.Params.ClientInfo = mcp.Implementation{
			Name:    "playwright-go-client",
			Version: "1.0.0",
		}
		initReq.Params.Capabilities = mcp.ClientCapabilities{}

		initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer initCancel()

		_, err = cli.Initialize(initCtx, initReq)
		if err != nil {
			log.Printf("[MCP] Fresh playwright initialization failed: %v", err)
			cli.Close()
			return "", fmt.Errorf("fresh playwright initialization failed: %v", err)
		}

		// 5. 立即使用工具（不等待）
		// 更新server变量用于工具调用
		server.Client = cli
		server.Connected = true

		log.Printf("[MCP] Fresh playwright connection created and initialized, ready for immediate tool call")
	} else if !m.isConnectionHealthy(serverName) {
		log.Printf("[MCP] Connection to %s appears unhealthy, attempting reconnection before tool call...", serverName)
		err := m.ConnectToServer(serverName)
		if err != nil {
			log.Printf("[MCP] Reconnection failed for server %s: %v", serverName, err)
			return "", fmt.Errorf("server %s connection unhealthy and reconnection failed: %v", serverName, err)
		}
		// Get updated server info after reconnection attempt
		server, ok = m.GetServer(serverName)
		if !ok {
			return "", fmt.Errorf("server %s not found after reconnection attempt", serverName)
		}
	}

	// Check if server is marked as needing reconnection
	if server.NeedsReconnect {
		// Check if we should attempt an immediate reconnection
		// Only attempt if last reconnect attempt was more than 30 seconds ago
		if time.Since(server.LastReconnectAttempt) > 30*time.Second {
			log.Printf("[MCP] Server %s needs reconnection, attempting immediate reconnection...", serverName)
			err := m.ConnectToServer(serverName)
			if err != nil {
				log.Printf("[MCP] Immediate reconnection failed for server %s: %v", serverName, err)
				return "", fmt.Errorf("server %s is reconnecting, please try again later (last error: %v)", serverName, server.LastError)
			}
			// Get updated server info after reconnection attempt
			server, ok = m.GetServer(serverName)
			if !ok {
				return "", fmt.Errorf("server %s not found after reconnection attempt", serverName)
			}
		} else {
			// Server is currently in reconnection process
			return "", fmt.Errorf("server %s is currently reconnecting, please try again later (last error: %v)", serverName, server.LastError)
		}
	}

	if !server.Connected {
		// Try to connect to this specific server
		err := m.ConnectToServer(serverName)
		if err != nil {
			return "", fmt.Errorf("MCP server %s not connected and failed to connect: %v", serverName, err)
		}
		// Get updated server info after connection attempt
		server, ok = m.GetServer(serverName)
		if !ok || !server.Connected {
			return "", fmt.Errorf("MCP server %s still not connected after connection attempt", serverName)
		}
	}

	// Handle built-in servers that don't have external MCP processes
	if server.Client == nil {
		return m.handleBuiltInTool(serverName, toolName, args)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Log the tool call details for debugging
	log.Printf("[MCP] Calling tool: server=%s, tool=%s, args=%v", serverName, toolName, args)

	// Special handling for context7 server - if tool is "call_tool", follow the correct tool sequence
	if serverName == "context7" && toolName == "call_tool" {
		log.Printf("[MCP] INFO: Detected call_tool for context7, following correct tool sequence")
		log.Printf("[MCP] Context7 has %d actual tools:", len(server.Tools))
		for i, tool := range server.Tools {
			log.Printf("[MCP]   Tool %d: %s - %s", i+1, tool.Name, tool.Description)
		}

		// First, try to use resolve-library-id to get a valid libraryId
		log.Printf("[MCP] First calling resolve-library-id to get valid libraryId")

		// Call resolve-library-id with the query
		resolveArgs := map[string]interface{}{
			"query":       "Next.js middleware JWT authentication",
			"libraryName": "next.js",
		}

		// Create a new context for the resolve call
		resolveCtx, resolveCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer resolveCancel()

		resolveResult, resolveErr := server.Client.CallTool(resolveCtx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "resolve-library-id",
				Arguments: resolveArgs,
			},
		})

		if resolveErr != nil {
			log.Printf("[MCP] resolve-library-id failed: %v", resolveErr)
			// Fall back to query-docs with default libraryId
			toolName = "query-docs"
			args = map[string]interface{}{
				"query":     "Next.js middleware JWT authentication",
				"libraryId": "/vercel/next.js", // Default Next.js library ID
			}
			log.Printf("[MCP] Using query-docs with default libraryId: %v", args)
		} else {
			// Extract libraryId from resolve result
			var libraryId string
			for _, content := range resolveResult.Content {
				if textContent, ok := content.(mcp.TextContent); ok {
					// Try to extract libraryId from the text
					// The result should contain something like: "/vercel/next.js"
					if strings.Contains(textContent.Text, "/vercel/next.js") {
						libraryId = "/vercel/next.js"
						break
					} else if strings.Contains(textContent.Text, "/org/") {
						// Try to extract any library ID
						lines := strings.Split(textContent.Text, "\n")
						for _, line := range lines {
							if strings.Contains(line, "/") && strings.Count(line, "/") >= 2 {
								libraryId = strings.TrimSpace(line)
								break
							}
						}
					}
				}
			}

			if libraryId == "" {
				libraryId = "/vercel/next.js" // Default fallback
			}

			// Now call query-docs with the obtained libraryId
			toolName = "query-docs"
			args = map[string]interface{}{
				"query":     "Next.js middleware JWT authentication",
				"libraryId": libraryId,
			}
			log.Printf("[MCP] Using query-docs with resolved libraryId: %s", libraryId)
		}
	}

	// 工具调用
	result, err := server.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})

	// 对于Playwright，工具调用后立即关闭连接（单次使用）
	if serverName == "playwright" && server.Client != nil {
		defer func() {
			log.Printf("[MCP] Closing fresh playwright connection after tool call...")
			server.Client.Close()
			server.Client = nil
			server.Connected = false
		}()
	}

	if err != nil {
		log.Printf("[MCP] Tool call failed: server=%s, tool=%s, error=%v", serverName, toolName, err)
		return "", err
	}

	// Extract text content
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		}
	}

	// Log detailed results for context7
	if serverName == "context7" {
		log.Printf("[MCP] Context7 raw result (length: %d):", len(output))
		// Log first 500 characters to see what's returned
		if len(output) > 0 {
			preview := output
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			log.Printf("[MCP] Context7 result preview: %s", preview)
		} else {
			log.Printf("[MCP] Context7 returned empty result")
		}
	} else {
		log.Printf("[MCP] Tool call successful: server=%s, tool=%s, result length=%d",
			serverName, toolName, len(output))
	}

	return output, nil
}

// asyncReconnectServer asynchronously attempts to reconnect to a server with exponential backoff
func (m *MCPManager) asyncReconnectServer(serverName, toolName string, args map[string]interface{}) {
	log.Printf("[MCP] Starting asynchronous reconnection for server: %s", serverName)

	// Exponential backoff parameters
	maxRetries := 5
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Calculate delay with exponential backoff
		delay := baseDelay * time.Duration(1<<(attempt-1)) // 1, 2, 4, 8, 16 seconds
		if delay > maxDelay {
			delay = maxDelay
		}

		log.Printf("[MCP] Reconnection attempt %d/%d for server %s, waiting %v before attempt...",
			attempt, maxRetries, serverName, delay)

		// Wait before attempting to reconnect
		time.Sleep(delay)

		// Get current server state
		server, ok := m.GetServer(serverName)
		if !ok {
			log.Printf("[MCP] Server %s not found during reconnection attempt %d", serverName, attempt)
			continue
		}

		// Update last reconnect attempt time
		server.LastReconnectAttempt = time.Now()
		m.mu.Lock()
		m.servers[serverName] = server
		m.mu.Unlock()

		log.Printf("[MCP] Attempting to reconnect to server %s (attempt %d/%d)...", serverName, attempt, maxRetries)

		// Try to reconnect
		err := m.ConnectToServer(serverName)
		if err != nil {
			log.Printf("[MCP] Failed to reconnect to server %s on attempt %d: %v", serverName, attempt, err)

			// Update error state
			server.LastError = err.Error()
			m.mu.Lock()
			m.servers[serverName] = server
			m.mu.Unlock()

			// Continue to next attempt
			continue
		}

		// Reconnection successful
		log.Printf("[MCP] Successfully reconnected to server %s on attempt %d", serverName, attempt)

		// Update server state
		server.NeedsReconnect = false
		server.LastError = ""
		m.mu.Lock()
		m.servers[serverName] = server
		m.mu.Unlock()

		// Log successful reconnection
		log.Printf("[MCP] Server %s reconnection completed successfully", serverName)
		return
	}

	// All retries failed
	log.Printf("[MCP] Failed to reconnect to server %s after %d attempts", serverName, maxRetries)

	// Mark server as permanently disconnected
	server, ok := m.GetServer(serverName)
	if ok {
		server.NeedsReconnect = false // Stop trying
		server.LastError = "Failed to reconnect after multiple attempts"
		m.mu.Lock()
		m.servers[serverName] = server
		m.mu.Unlock()
	}
}

// handleBuiltInTool handles calls to built-in MCP tools
func (m *MCPManager) handleBuiltInTool(serverName, toolName string, args map[string]interface{}) (string, error) {
	switch serverName {
	case "filesystem-local":
		return m.handleFilesystemTool(toolName, args)
	case "terminal":
		return m.handleTerminalTool(toolName, args)
	case "code-analysis":
		return m.handleCodeAnalysisTool(toolName, args)
	case "search":
		return m.handleSearchTool(toolName, args)
	default:
		return "", fmt.Errorf("unknown built-in server: %s", serverName)
	}
}

// handleFilesystemTool handles filesystem operations
func (m *MCPManager) handleFilesystemTool(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "read_file":
		path, _ := args["path"].(string)
		if path == "" {
			return "", fmt.Errorf("path is required")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(content), nil

	case "write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		if path == "" {
			return "", fmt.Errorf("path is required")
		}
		// Ensure directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil

	case "list_directory":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", fmt.Errorf("failed to list directory: %w", err)
		}
		var result strings.Builder
		for _, entry := range entries {
			if entry.IsDir() {
				result.WriteString(fmt.Sprintf("[DIR]  %s\n", entry.Name()))
			} else {
				info, _ := entry.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				result.WriteString(fmt.Sprintf("[FILE] %s (%d bytes)\n", entry.Name(), size))
			}
		}
		return result.String(), nil

	default:
		return "", fmt.Errorf("unknown filesystem tool: %s", toolName)
	}
}

// handleTerminalTool handles terminal/command execution
func (m *MCPManager) handleTerminalTool(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "execute_command":
		command, _ := args["command"].(string)
		workingDir, _ := args["working_dir"].(string)
		if command == "" {
			return "", fmt.Errorf("command is required")
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}

		if workingDir != "" {
			cmd.Dir = workingDir
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output), fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
		}
		return string(output), nil

	default:
		return "", fmt.Errorf("unknown terminal tool: %s", toolName)
	}
}

// handleCodeAnalysisTool handles code analysis operations
func (m *MCPManager) handleCodeAnalysisTool(toolName string, args map[string]interface{}) (string, error) {
	code, _ := args["code"].(string)
	language, _ := args["language"].(string)

	switch toolName {
	case "analyze_code":
		if code == "" {
			return "", fmt.Errorf("code is required")
		}
		// Basic static analysis
		var issues []string
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			// Check for common issues
			if strings.Contains(strings.ToLower(line), "todo") || strings.Contains(strings.ToLower(line), "fixme") {
				issues = append(issues, fmt.Sprintf("Line %d: TODO/FIXME comment found", i+1))
			}
			if len(line) > 120 {
				issues = append(issues, fmt.Sprintf("Line %d: Line exceeds 120 characters", i+1))
			}
		}
		if len(issues) == 0 {
			return fmt.Sprintf("No issues found in %s code (%d lines)", language, len(lines)), nil
		}
		return fmt.Sprintf("Found %d issues:\n%s", len(issues), strings.Join(issues, "\n")), nil

	case "suggest_improvements":
		if code == "" {
			return "", fmt.Errorf("code is required")
		}
		var suggestions []string
		lines := strings.Split(code, "\n")

		// Language-specific suggestions
		switch language {
		case "go", "golang":
			if !strings.Contains(code, "func main()") && !strings.Contains(code, "package ") {
				suggestions = append(suggestions, "Consider adding package declaration")
			}
			if strings.Contains(code, "fmt.Println") && !strings.Contains(code, "\"fmt\"") {
				suggestions = append(suggestions, "Missing fmt import")
			}
		case "python":
			if !strings.Contains(code, "if __name__") && strings.Contains(code, "def ") {
				suggestions = append(suggestions, "Consider adding if __name__ == '__main__' guard")
			}
		case "javascript", "typescript":
			if strings.Contains(code, "var ") {
				suggestions = append(suggestions, "Consider using 'let' or 'const' instead of 'var'")
			}
		}

		if len(suggestions) == 0 {
			return fmt.Sprintf("Code looks good! (%d lines)", len(lines)), nil
		}
		return fmt.Sprintf("Suggestions:\n%s", strings.Join(suggestions, "\n")), nil

	default:
		return "", fmt.Errorf("unknown code-analysis tool: %s", toolName)
	}
}

// handleSearchTool handles web search operations
func (m *MCPManager) handleSearchTool(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "web_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		// Return a message indicating search is not available without external MCP
		return fmt.Sprintf("Web search for '%s' requires an external search MCP server (e.g., brave-search MCP). Please configure one in your MCP settings.", query), nil

	default:
		return "", fmt.Errorf("unknown search tool: %s", toolName)
	}
}

// SearchCodeExamples searches for code examples using web search MCP
func (m *MCPManager) SearchCodeExamples(query string) ([]CodeExample, error) {
	server, ok := m.GetServer("web-search")
	if !ok || !server.Connected {
		return nil, fmt.Errorf("web-search MCP not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := server.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "brave_web_search",
			Arguments: map[string]interface{}{
				"query": query + " code example tutorial",
			},
		},
	})

	if err != nil {
		return nil, err
	}

	// Parse results into CodeExample structs
	var examples []CodeExample
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			// Extract code blocks from search results
			examples = append(examples, extractExamplesFromText(textContent.Text)...)
		}
	}

	return examples, nil
}

// SearchErrorSolutions searches for solutions to errors
func (m *MCPManager) SearchErrorSolutions(query string) ([]string, error) {
	server, ok := m.GetServer("web-search")
	if !ok || !server.Connected {
		return nil, fmt.Errorf("web-search MCP not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := server.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "brave_web_search",
			Arguments: map[string]interface{}{
				"query": query + " stackoverflow solution fix",
			},
		},
	})

	if err != nil {
		return nil, err
	}

	var solutions []string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			solutions = append(solutions, textContent.Text)
		}
	}

	return solutions, nil
}

// ReadFile reads a file using the filesystem MCP
func (m *MCPManager) ReadFile(path string) (string, error) {
	return m.CallTool("filesystem", "read_file", map[string]interface{}{
		"path": path,
	})
}

// WriteFile writes a file using the filesystem MCP
func (m *MCPManager) WriteFile(path string, content string) error {
	_, err := m.CallTool("filesystem", "write_file", map[string]interface{}{
		"path":    path,
		"content": content,
	})
	return err
}

// ListDirectory lists directory contents using the filesystem MCP
func (m *MCPManager) ListDirectory(path string) ([]string, error) {
	output, err := m.CallTool("filesystem", "list_directory", map[string]interface{}{
		"path": path,
	})
	if err != nil {
		return nil, err
	}

	// Parse output as JSON array
	var files []string
	if err := json.Unmarshal([]byte(output), &files); err != nil {
		// Fallback: split by newlines
		lines := make([]string, 0)
		for _, line := range splitLines(output) {
			if trimmed := trimSpace(line); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		return lines, nil
	}

	return files, nil
}

// SearchFiles searches for files using the filesystem MCP
func (m *MCPManager) SearchFiles(pattern string) ([]string, error) {
	output, err := m.CallTool("filesystem", "search_files", map[string]interface{}{
		"pattern": pattern,
	})
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range splitLines(output) {
		if trimmed := trimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
}

// GetGitStatus gets git repository status
func (m *MCPManager) GetGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GitCommit commits changes with a message
func (m *MCPManager) GitCommit(message string) error {
	// Add all changes
	addCmd := exec.Command("git", "add", "-A")
	if err := addCmd.Run(); err != nil {
		return err
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	return commitCmd.Run()
}

// Helper functions
func extractExamplesFromText(text string) []CodeExample {
	var examples []CodeExample

	// Simple extraction - look for code blocks
	lines := splitLines(text)
	var inCodeBlock bool
	var codeLines []string
	var language string

	for _, line := range lines {
		trimmed := trimSpace(line)

		if hasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Opening
				inCodeBlock = true
				language = trimSpace(trimPrefix(trimmed, "```"))
				codeLines = []string{}
			} else {
				// Closing
				if len(codeLines) > 0 {
					examples = append(examples, CodeExample{
						Source:   "web_search",
						Code:     joinLines(codeLines),
						Language: language,
					})
				}
				inCodeBlock = false
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	return examples
}

// String manipulation helpers (avoiding strings package conflicts)
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}

// GetConfigValidator returns the configuration validator
func (m *MCPManager) GetConfigValidator() *ConfigValidator {
	return m.configValidator
}

// GetMonitor returns the monitor
func (m *MCPManager) GetMonitor() *Monitor {
	return m.monitor
}

// shouldSkipServer 检查是否应该跳过某个服务器的连接
// 返回true表示应该跳过，false表示应该尝试连接
func (m *MCPManager) shouldSkipServer(serverName string) bool {
	// 检查熔断器状态
	cb := m.getOrCreateCircuitBreaker(serverName)
	state := cb.GetState()

	// 如果熔断器打开，跳过连接
	if state == StateOpen {
		log.Printf("[MCP] Skipping server %s due to open circuit breaker", serverName)
		return true
	}

	// 检查服务器是否已经连接
	m.mu.RLock()
	server, exists := m.servers[serverName]
	m.mu.RUnlock()

	if exists && server.Connected {
		log.Printf("[MCP] Server %s already connected, skipping", serverName)
		return true
	}

	return false
}

// isConnectionHealthy 检查MCP服务器连接是否健康
func (m *MCPManager) isConnectionHealthy(serverName string) bool {
	server, ok := m.GetServer(serverName)
	if !ok || !server.Connected || server.Client == nil {
		log.Printf("[MCP] Health check: server %s not connected or client is nil", serverName)
		return false
	}

	// 对于需要健康检查的服务器，进行更严格的检查
	if serverName == "playwright" || serverName == "context7" {
		return m.checkServerHealth(serverName, server)
	}

	// 对于其他服务器，默认认为连接健康
	// 可以通过发送ping或简单请求来验证，但为了性能暂时跳过
	return true
}

// checkServerHealth 检查服务器连接健康状态
func (m *MCPManager) checkServerHealth(serverName string, server *config.MCPServer) bool {
	// 确定使用哪个工具进行健康检查
	var toolName string
	var args map[string]interface{}

	switch serverName {
	case "playwright":
		toolName = "browser_list"
		args = map[string]interface{}{}
	case "context7":
		toolName = "resolve-library-id"
		args = map[string]interface{}{
			"query":       "health-check",
			"libraryName": "react",
		}
	default:
		// 对于其他服务器，默认认为健康
		return true
	}

	// 对于Playwright，使用简单的健康检查
	// 因为Playwright连接可能不稳定，复杂的健康检查可能反而干扰
	if serverName == "playwright" {
		// 只检查最基本的连接状态
		if server.Client == nil || !server.Connected {
			log.Printf("[MCP] %s health check failed (client nil or not connected)", serverName)
			return false
		}

		// 对于Playwright，我们返回true，让工具调用自己处理错误
		// 因为健康检查可能通过，但工具调用时连接可能已经断开
		log.Printf("[MCP] %s health check passed (basic check)", serverName)
		return true
	}

	// 对于其他服务器，尝试发送一个简单的请求检查连接是否有效
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试调用一个简单的工具检查连接
	_, err := server.Client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})

	if err != nil {
		// 检查是否是broken pipe或其他连接错误
		if strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "transport error") ||
			strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "EOF") {
			log.Printf("[MCP] %s health check failed (connection issue): %v", serverName, err)
			return false
		}
		// 其他错误可能只是工具不存在，但连接本身可能还是好的
		log.Printf("[MCP] %s health check warning (tool error): %v", serverName, err)
		return true // 连接可能还是好的，只是工具调用失败
	}

	log.Printf("[MCP] %s health check passed", serverName)
	return true
}

// markConnectionUnhealthy 标记连接为不健康
func (m *MCPManager) markConnectionUnhealthy(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if server, exists := m.servers[serverName]; exists {
		server.Connected = false
		server.NeedsReconnect = true
		server.LastReconnectAttempt = time.Now()
		log.Printf("[MCP] Marked server %s as unhealthy", serverName)
	}
}

// testServerProcess 测试服务器进程是否可启动（与setting模块相同的测试方法）
func (m *MCPManager) testServerProcess(server config.MCPServer) bool {
	// 检查基本配置
	if server.Type != "builtin" && server.Command == "" {
		log.Printf("[MCP] TEST: Command not configured for %s", server.Name)
		return false
	}

	// builtin 类型直接返回成功
	if server.Type == "builtin" {
		log.Printf("[MCP] TEST: Builtin server %s - always pass", server.Name)
		return true
	}

	// 测试命令是否存在
	cmdPath, err := exec.LookPath(server.Command)
	if err != nil {
		log.Printf("[MCP] TEST: Command '%s' not found: %v", server.Command, err)
		return false
	}

	// 尝试启动进程并检查是否可以初始化 MCP 连接
	// 对于context7，使用更长的超时时间（与Setting模块相同的30秒）
	var timeout time.Duration
	if server.Name == "context7" {
		timeout = 30 * time.Second
		log.Printf("[MCP] TEST: Using extended timeout (%v) for context7 process test", timeout)
	} else {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, server.Args...)

	// 设置环境变量 - 使用与Setting模块相同的格式
	cmd.Env = os.Environ() // 先继承当前环境
	for k, v := range server.Env {
		// Setting模块使用原始key（不转换为大写）
		envVar := fmt.Sprintf("%s=%s", k, v)
		cmd.Env = append(cmd.Env, envVar)
		log.Printf("[MCP] TEST: Setting env var: %s", k)
	}

	// 启动进程
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("[MCP] TEST: Cannot create stdin pipe: %v", err)
		return false
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[MCP] TEST: Cannot create stdout pipe: %v", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[MCP] TEST: Cannot start process: %v", err)
		return false
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
		log.Printf("[MCP] TEST: Failed to send initialization request: %v", err)
		return false
	}

	// 读取响应
	decoder := json.NewDecoder(stdout)
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		cmd.Process.Kill()
		cmd.Wait() // 等待进程结束以获取退出状态
		exitCode := cmd.ProcessState.ExitCode()
		log.Printf("[MCP] TEST: Failed to read response: %v (exit code: %d)", err, exitCode)

		// 对于context7，提供更详细的诊断信息
		if server.Name == "context7" {
			log.Printf("[MCP] CONTEXT7_DIAGNOSIS: Process test failed with EOF error")
			log.Printf("[MCP] CONTEXT7_DIAGNOSIS: Command: %s %v", cmdPath, server.Args)
			log.Printf("[MCP] CONTEXT7_DIAGNOSIS: Env vars set: %d", len(server.Env))
			for k, v := range server.Env {
				// 只显示key，不显示value（敏感信息）
				log.Printf("[MCP] CONTEXT7_DIAGNOSIS:   - %s (value set: %v)", k, v != "")
			}

			// 尝试获取stderr输出以了解具体错误
			stderr, _ := cmd.StderrPipe()
			if stderr != nil {
				stderrBytes, _ := io.ReadAll(stderr)
				if len(stderrBytes) > 0 {
					log.Printf("[MCP] CONTEXT7_DIAGNOSIS: Stderr output: %s", string(stderrBytes))
				}
			}

			// 检查进程状态
			if exitCode == -1 {
				log.Printf("[MCP] CONTEXT7_DIAGNOSIS: Process terminated by signal (exit code -1)")
				log.Printf("[MCP] CONTEXT7_DIAGNOSIS: This usually means the process crashed or was killed")
			}
		}
		return false
	}

	// 检查响应
	if result, ok := response["result"]; ok {
		log.Printf("[MCP] TEST: Server %s responded with result: %v", server.Name, result)
		// 对于Context7，不杀死进程，让它继续运行供MCP连接使用
		if server.Name == "context7" {
			log.Printf("[MCP] CONTEXT7_DEBUG: Process test successful, keeping process alive for MCP connection")
			// 不杀死进程，返回true让调用者知道进程正在运行
		} else {
			cmd.Process.Kill()
		}
		return true
	}

	if error, ok := response["error"]; ok {
		log.Printf("[MCP] TEST: Server %s responded with error: %v", server.Name, error)
		cmd.Process.Kill()
		return false
	}

	log.Printf("[MCP] TEST: Server %s responded with unknown format: %v", server.Name, response)
	cmd.Process.Kill()
	return false
}
