package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"backend/internal/config"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPManager manages MCP server connections
type MCPManager struct {
	servers    map[string]*MCPServer
	mu         sync.RWMutex
	discovered bool
	discoverMu sync.RWMutex
}

// MCPServer represents a connected MCP server
type MCPServer struct {
	Name      string         `json:"name"`
	Command   string         `json:"command"`
	Args      []string       `json:"args"`
	Client    *client.Client `json:"-"`
	Connected bool           `json:"connected"`
	Tools     []mcp.Tool     `json:"tools"`
	LastError string         `json:"last_error,omitempty"`
}

// NewMCPManager creates a new MCP manager
func NewMCPManager() *MCPManager {
	return &MCPManager{
		servers: make(map[string]*MCPServer),
	}
}

// Discover automatically discovers and connects to MCP servers using multiple methods
// Only runs once - subsequent calls will return cached results unless force=true
func (m *MCPManager) Discover(force ...bool) {
	// Check if already discovered and not forcing refresh
	m.discoverMu.RLock()
	alreadyDiscovered := m.discovered
	m.discoverMu.RUnlock()

	if alreadyDiscovered && len(force) == 0 {
		return
	}

	fmt.Println("[MCP] Starting multi-method discovery...")

	// Method 1: Built-in MCPs (always available, no external dependencies)
	fmt.Println("[MCP] Method 1: Loading built-in MCPs...")
	m.addMockFilesystemMCP()
	m.AddBuiltInMCPs()

	// Method 2: External MCPs via npx (requires Node.js)
	fmt.Println("[MCP] Method 2: Discovering external MCPs via npx...")
	m.discoverExternalMCPs()

	// Method 3: MCPs from environment variables
	fmt.Println("[MCP] Method 3: Loading MCPs from environment...")
	m.discoverFromEnvironment()

	// Method 4: MCPs from configuration file
	fmt.Println("[MCP] Method 4: Loading MCPs from config file...")
	m.discoverFromConfig()

	// Method 5: MCPs from Docker socket (if running in Docker)
	fmt.Println("[MCP] Method 5: Checking for Docker-based MCPs...")
	m.discoverFromDocker()

	// Mark as discovered
	m.discoverMu.Lock()
	m.discovered = true
	m.discoverMu.Unlock()

	fmt.Println("[MCP] Discovery completed and cached.")
}

// Refresh forces a re-discovery of MCP servers
func (m *MCPManager) Refresh() {
	fmt.Println("[MCP] Refreshing MCP discovery...")
	m.Discover(true)
}

// discoverExternalMCPs discovers MCP servers via npx
func (m *MCPManager) discoverExternalMCPs() {
	commonServers := []MCPServer{
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

			server := MCPServer{
				Name:    name,
				Command: configParts[0],
				Args:    configParts[1:],
			}

			fmt.Printf("[MCP] Found environment MCP: %s\n", name)
			go m.connectServer(server)
		}
	}
}

// discoverFromConfig discovers MCPs from configuration files
func (m *MCPManager) discoverFromConfig() {
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
			var servers []MCPServer
			if err := json.Unmarshal(data, &servers); err == nil {
				fmt.Printf("[MCP] Loaded %d MCPs from %s\n", len(servers), path)
				for _, server := range servers {
					go m.connectServer(server)
				}
				return
			}

			// Try parsing as object with "servers" field
			var config struct {
				Servers []MCPServer `json:"servers"`
			}
			if err := json.Unmarshal(data, &config); err == nil {
				fmt.Printf("[MCP] Loaded %d MCPs from %s\n", len(config.Servers), path)
				for _, server := range config.Servers {
					go m.connectServer(server)
				}
				return
			}

			fmt.Printf("[MCP] Failed to parse config %s: invalid format\n", path)
		}
	}
}

// discoverFromMCPServersConfig loads enabled MCP servers from mcpservers.yaml
func (m *MCPManager) discoverFromMCPServersConfig() {
	cfg := config.GetMCPServersConfig()
	if cfg == nil || cfg.Servers == nil {
		fmt.Println("[MCP] No MCP servers config loaded from mcpservers.yaml")
		return
	}

	enabledCount := 0
	for name, server := range cfg.Servers {
		if !server.Enabled {
			continue
		}

		// Skip builtin servers (they are handled separately)
		if server.Type == "builtin" {
			continue
		}

		mcpServer := MCPServer{
			Name:    name,
			Command: server.Command,
			Args:    server.Args,
		}

		fmt.Printf("[MCP] Loading enabled MCP server from config: %s (type: %s)\n", name, server.Type)
		go m.connectServer(mcpServer)
		enabledCount++
	}

	if enabledCount > 0 {
		fmt.Printf("[MCP] Loaded %d enabled MCP servers from mcpservers.yaml\n", enabledCount)
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
	fmt.Println("[MCP] Docker socket available, but Docker discovery not implemented")
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
	m.servers["filesystem-local"] = &MCPServer{
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

	fmt.Println("[MCP] Added built-in filesystem-local server")
}

// AddBuiltInMCPs adds built-in MCP servers that don't require external processes
func (m *MCPManager) AddBuiltInMCPs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Terminal MCP - executes shell commands
	if _, exists := m.servers["terminal"]; !exists {
		m.servers["terminal"] = &MCPServer{
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
		fmt.Println("[MCP] Added built-in terminal server")
	}

	// Code Analysis MCP
	if _, exists := m.servers["code-analysis"]; !exists {
		m.servers["code-analysis"] = &MCPServer{
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
		fmt.Println("[MCP] Added built-in code-analysis server")
	}

	// Search MCP
	if _, exists := m.servers["search"]; !exists {
		m.servers["search"] = &MCPServer{
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
		fmt.Println("[MCP] Added built-in search server")
	}
}

// connectServer attempts to connect to an MCP server
func (m *MCPManager) connectServer(server MCPServer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if existing, ok := m.servers[server.Name]; ok && existing.Connected {
		return
	}

	// Create client
	cli, err := client.NewStdioMCPClient(server.Command, server.Args)
	if err != nil {
		server.LastError = err.Error()
		m.servers[server.Name] = &server
		return
	}

	// Initialize
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "ai-programming-agent",
		Version: "1.0.0",
	}

	_, err = cli.Initialize(ctx, initReq)
	if err != nil {
		server.LastError = err.Error()
		cli.Close()
		m.servers[server.Name] = &server
		return
	}

	// List tools
	toolsResult, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		server.LastError = err.Error()
		cli.Close()
		m.servers[server.Name] = &server
		return
	}

	server.Client = cli
	server.Connected = true
	server.Tools = toolsResult.Tools
	m.servers[server.Name] = &server
}

// ListConnected returns all connected MCP servers
func (m *MCPManager) ListConnected() []MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connected []MCPServer
	for _, server := range m.servers {
		if server.Connected {
			connected = append(connected, *server)
		}
	}
	return connected
}

// ListAll returns all MCP servers including disconnected ones
func (m *MCPManager) ListAll() []MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []MCPServer
	for _, server := range m.servers {
		all = append(all, *server)
	}
	return all
}

// GetServer gets a specific MCP server
func (m *MCPManager) GetServer(name string) (*MCPServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[name]
	return server, ok
}

// GetAllServers gets all MCP servers
func (m *MCPManager) GetAllServers() map[string]*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid external modifications
	servers := make(map[string]*MCPServer)
	for name, server := range m.servers {
		servers[name] = server
	}
	return servers
}

// CallTool calls a tool on an MCP server
func (m *MCPManager) CallTool(serverName string, toolName string, args map[string]interface{}) (string, error) {
	server, ok := m.GetServer(serverName)
	if !ok {
		return "", fmt.Errorf("MCP server %s not found", serverName)
	}

	if !server.Connected {
		return "", fmt.Errorf("MCP server %s not connected", serverName)
	}

	// Handle built-in servers that don't have external MCP processes
	if server.Client == nil {
		return m.handleBuiltInTool(serverName, toolName, args)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := server.Client.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})

	if err != nil {
		return "", err
	}

	// Extract text content
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		}
	}

	return output, nil
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

// Close closes all MCP connections
func (m *MCPManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, server := range m.servers {
		if server.Client != nil {
			server.Client.Close()
		}
	}
}
