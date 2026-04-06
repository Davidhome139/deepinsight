package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GitOperationsManager handles Git-aware operations
type GitOperationsManager struct {
	workDir     string
	currentTask string
	mu          sync.Mutex
}

// GitBranch represents a Git branch
type GitBranch struct {
	Name      string    `json:"name"`
	IsCurrent bool      `json:"is_current"`
	TaskID    string    `json:"task_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// GitCommit represents a Git commit
type GitCommit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	TaskID    string    `json:"task_id,omitempty"`
}

// GitStatus represents the Git repository status
type GitStatus struct {
	Branch        string   `json:"branch"`
	Staged        []string `json:"staged"`
	Modified      []string `json:"modified"`
	Untracked     []string `json:"untracked"`
	Ahead         int      `json:"ahead"`
	Behind        int      `json:"behind"`
	IsClean       bool     `json:"is_clean"`
	HasConflicts  bool     `json:"has_conflicts"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
}

// PRInfo represents Pull Request information
type PRInfo struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	BaseBranch string   `json:"base_branch"`
	HeadBranch string   `json:"head_branch"`
	Labels     []string `json:"labels,omitempty"`
	Reviewers  []string `json:"reviewers,omitempty"`
	Draft      bool     `json:"draft"`
	Files      []string `json:"files"`
	Commits    int      `json:"commits"`
}

// NewGitOperationsManager creates a new Git operations manager
func NewGitOperationsManager(workDir string) *GitOperationsManager {
	return &GitOperationsManager{
		workDir: workDir,
	}
}

// IsGitRepo checks if the working directory is a Git repository
func (g *GitOperationsManager) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = g.workDir
	err := cmd.Run()
	return err == nil
}

// InitRepo initializes a new Git repository
func (g *GitOperationsManager) InitRepo() error {
	cmd := exec.Command("git", "init")
	cmd.Dir = g.workDir
	return cmd.Run()
}

// GetStatus returns the current Git status
func (g *GitOperationsManager) GetStatus() (*GitStatus, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	status := &GitStatus{}

	// Get current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = g.workDir
	branchOut, err := branchCmd.Output()
	if err == nil {
		status.Branch = strings.TrimSpace(string(branchOut))
	}

	// Get status
	statusCmd := exec.Command("git", "status", "--porcelain", "-b")
	statusCmd.Dir = g.workDir
	statusOut, err := statusCmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(statusOut), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// Parse branch info (first line with ##)
		if strings.HasPrefix(line, "##") {
			// Parse ahead/behind
			if strings.Contains(line, "ahead") {
				fmt.Sscanf(line, "%*s [ahead %d", &status.Ahead)
			}
			if strings.Contains(line, "behind") {
				fmt.Sscanf(line, "%*s [behind %d", &status.Behind)
			}
			continue
		}

		index := line[0]
		worktree := line[1]
		filename := strings.TrimSpace(line[3:])

		// Staged changes
		if index != ' ' && index != '?' {
			status.Staged = append(status.Staged, filename)
		}

		// Modified in worktree
		if worktree == 'M' || worktree == 'D' {
			status.Modified = append(status.Modified, filename)
		}

		// Untracked
		if index == '?' && worktree == '?' {
			status.Untracked = append(status.Untracked, filename)
		}

		// Conflicts
		if index == 'U' || worktree == 'U' {
			status.HasConflicts = true
			status.ConflictFiles = append(status.ConflictFiles, filename)
		}
	}

	status.IsClean = len(status.Staged) == 0 && len(status.Modified) == 0 && len(status.Untracked) == 0

	return status, nil
}

// CreateTaskBranch creates a new branch for a task
func (g *GitOperationsManager) CreateTaskBranch(taskID, description string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Generate branch name from task description
	branchName := g.generateBranchName(taskID, description)

	// Create and checkout new branch
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = g.workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}

	g.currentTask = taskID
	return nil
}

// generateBranchName generates a clean branch name
func (g *GitOperationsManager) generateBranchName(taskID, description string) string {
	// Clean description for branch name
	clean := strings.ToLower(description)
	clean = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, clean)

	// Remove multiple dashes
	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}
	clean = strings.Trim(clean, "-")

	// Truncate if too long
	if len(clean) > 30 {
		clean = clean[:30]
	}

	return fmt.Sprintf("ai-task/%s/%s", taskID[:8], clean)
}

// StageFiles stages specified files
func (g *GitOperationsManager) StageFiles(files []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// StageAll stages all changes
func (g *GitOperationsManager) StageAll() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = g.workDir
	return cmd.Run()
}

// Commit creates a commit with the specified message
func (g *GitOperationsManager) Commit(message string, taskID string) (*GitCommit, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Format commit message with task ID
	fullMessage := message
	if taskID != "" {
		fullMessage = fmt.Sprintf("[%s] %s", taskID, message)
	}

	cmd := exec.Command("git", "commit", "-m", fullMessage)
	cmd.Dir = g.workDir
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Get the commit info
	return g.getLastCommit()
}

// getLastCommit returns the last commit info
func (g *GitOperationsManager) getLastCommit() (*GitCommit, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%H|%s|%an|%ai")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid commit format")
	}

	timestamp, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])

	return &GitCommit{
		Hash:      parts[0],
		Message:   parts[1],
		Author:    parts[2],
		Timestamp: timestamp,
	}, nil
}

// GetCommitHistory returns the commit history
func (g *GitOperationsManager) GetCommitHistory(limit int) ([]*GitCommit, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", limit), "--format=%H|%s|%an|%ai")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	commits := make([]*GitCommit, 0)
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		timestamp, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])
		commits = append(commits, &GitCommit{
			Hash:      parts[0],
			Message:   parts[1],
			Author:    parts[2],
			Timestamp: timestamp,
		})
	}

	return commits, nil
}

// ListBranches lists all branches
func (g *GitOperationsManager) ListBranches() ([]*GitBranch, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)|%(HEAD)")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := make([]*GitBranch, 0)
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		branches = append(branches, &GitBranch{
			Name:      parts[0],
			IsCurrent: len(parts) > 1 && parts[1] == "*",
		})
	}

	return branches, nil
}

// SwitchBranch switches to a different branch
func (g *GitOperationsManager) SwitchBranch(branchName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// MergeBranch merges a branch into the current branch
func (g *GitOperationsManager) MergeBranch(branchName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "merge", branchName, "--no-ff")
	cmd.Dir = g.workDir
	return cmd.Run()
}

// Push pushes changes to remote
func (g *GitOperationsManager) Push(remote, branch string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "push", "-u", remote, branch)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// PreparePRInfo prepares information for creating a PR
func (g *GitOperationsManager) PreparePRInfo(taskID, description string) (*PRInfo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	pr := &PRInfo{
		Title:      description,
		Body:       g.generatePRBody(taskID),
		BaseBranch: "main",
		Labels:     []string{"ai-generated"},
	}

	// Get current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = g.workDir
	branchOut, err := branchCmd.Output()
	if err == nil {
		pr.HeadBranch = strings.TrimSpace(string(branchOut))
	}

	// Get changed files
	diffCmd := exec.Command("git", "diff", "--name-only", "main...HEAD")
	diffCmd.Dir = g.workDir
	diffOut, _ := diffCmd.Output()
	pr.Files = strings.Split(strings.TrimSpace(string(diffOut)), "\n")

	// Get commit count
	countCmd := exec.Command("git", "rev-list", "--count", "main...HEAD")
	countCmd.Dir = g.workDir
	countOut, _ := countCmd.Output()
	fmt.Sscanf(string(countOut), "%d", &pr.Commits)

	return pr, nil
}

// generatePRBody generates a PR body from task information
func (g *GitOperationsManager) generatePRBody(taskID string) string {
	return fmt.Sprintf(`## AI-Generated Changes

**Task ID:** %s

### Summary
This PR contains changes generated by the AI Programming Agent.

### Changes
- See the diff for detailed changes

### Testing
- [ ] Unit tests added/updated
- [ ] Integration tests passed
- [ ] Manual testing completed

### Checklist
- [ ] Code follows project conventions
- [ ] Documentation updated
- [ ] No sensitive data exposed
`, taskID)
}

// GetDiff returns the diff between two references
func (g *GitOperationsManager) GetDiff(base, head string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	cmd := exec.Command("git", "diff", base+"..."+head)
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// LSPManager provides Language Server Protocol integration for syntax checking
type LSPManager struct {
	workDir string
	servers map[string]*LSPServer
	mu      sync.RWMutex
}

// LSPServer represents a language server
type LSPServer struct {
	Language string
	Command  string
	Args     []string
	Running  bool
	Process  *os.Process
}

// Diagnostic represents a code diagnostic (error, warning, etc.)
type Diagnostic struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	EndLine    int    `json:"end_line,omitempty"`
	EndColumn  int    `json:"end_column,omitempty"`
	Severity   string `json:"severity"` // error, warning, info, hint
	Message    string `json:"message"`
	Source     string `json:"source"`
	Code       string `json:"code,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// CodeAction represents a suggested code action/fix
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind"` // quickfix, refactor, source
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// WorkspaceEdit represents changes to workspace files
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

// TextEdit represents a text edit
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"new_text"`
}

// Range represents a range in a document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Command represents a command to execute
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// NewLSPManager creates a new LSP manager
func NewLSPManager(workDir string) *LSPManager {
	return &LSPManager{
		workDir: workDir,
		servers: make(map[string]*LSPServer),
	}
}

// RegisterServer registers a language server
func (l *LSPManager) RegisterServer(language, command string, args ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.servers[language] = &LSPServer{
		Language: language,
		Command:  command,
		Args:     args,
	}
}

// CheckSyntax checks the syntax of a file using external tools
// Note: Full LSP integration requires a more complex implementation
// This provides basic syntax checking using language-specific tools
func (l *LSPManager) CheckSyntax(filePath string) ([]Diagnostic, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(filePath))
	diagnostics := make([]Diagnostic, 0)

	switch ext {
	case ".go":
		return l.checkGoSyntax(filePath)
	case ".py":
		return l.checkPythonSyntax(filePath)
	case ".js", ".ts", ".jsx", ".tsx":
		return l.checkJSSyntax(filePath)
	case ".java":
		return l.checkJavaSyntax(filePath)
	default:
		return diagnostics, nil
	}
}

// checkGoSyntax checks Go syntax using go vet and go build
func (l *LSPManager) checkGoSyntax(filePath string) ([]Diagnostic, error) {
	diagnostics := make([]Diagnostic, 0)

	// Run go vet
	vetCmd := exec.Command("go", "vet", filePath)
	vetCmd.Dir = l.workDir
	vetOut, _ := vetCmd.CombinedOutput()

	// Parse vet output
	for _, line := range strings.Split(string(vetOut), "\n") {
		if d := l.parseGoDiagnostic(line); d != nil {
			diagnostics = append(diagnostics, *d)
		}
	}

	// Run go build (dry run)
	buildCmd := exec.Command("go", "build", "-n", filePath)
	buildCmd.Dir = l.workDir
	buildOut, _ := buildCmd.CombinedOutput()

	for _, line := range strings.Split(string(buildOut), "\n") {
		if d := l.parseGoDiagnostic(line); d != nil {
			diagnostics = append(diagnostics, *d)
		}
	}

	return diagnostics, nil
}

// parseGoDiagnostic parses a Go diagnostic line
func (l *LSPManager) parseGoDiagnostic(line string) *Diagnostic {
	// Format: file:line:column: message
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return nil
	}

	var lineNum, col int
	fmt.Sscanf(parts[1], "%d", &lineNum)
	fmt.Sscanf(parts[2], "%d", &col)

	return &Diagnostic{
		File:     parts[0],
		Line:     lineNum,
		Column:   col,
		Severity: "error",
		Message:  strings.TrimSpace(parts[3]),
		Source:   "go",
	}
}

// checkPythonSyntax checks Python syntax using py_compile
func (l *LSPManager) checkPythonSyntax(filePath string) ([]Diagnostic, error) {
	diagnostics := make([]Diagnostic, 0)

	cmd := exec.Command("python3", "-m", "py_compile", filePath)
	cmd.Dir = l.workDir
	out, err := cmd.CombinedOutput()

	if err != nil {
		// Parse error output
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "SyntaxError") || strings.Contains(line, "Error") {
				diagnostics = append(diagnostics, Diagnostic{
					File:     filePath,
					Severity: "error",
					Message:  line,
					Source:   "python",
				})
			}
		}
	}

	return diagnostics, nil
}

// checkJSSyntax checks JavaScript/TypeScript syntax using tsc or node
func (l *LSPManager) checkJSSyntax(filePath string) ([]Diagnostic, error) {
	diagnostics := make([]Diagnostic, 0)
	ext := filepath.Ext(filePath)

	var cmd *exec.Cmd
	if ext == ".ts" || ext == ".tsx" {
		cmd = exec.Command("npx", "tsc", "--noEmit", filePath)
	} else {
		cmd = exec.Command("node", "--check", filePath)
	}

	cmd.Dir = l.workDir
	out, err := cmd.CombinedOutput()

	if err != nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "error") || strings.Contains(line, "Error") {
				diagnostics = append(diagnostics, Diagnostic{
					File:     filePath,
					Severity: "error",
					Message:  line,
					Source:   "typescript",
				})
			}
		}
	}

	return diagnostics, nil
}

// checkJavaSyntax checks Java syntax using javac
func (l *LSPManager) checkJavaSyntax(filePath string) ([]Diagnostic, error) {
	diagnostics := make([]Diagnostic, 0)

	cmd := exec.Command("javac", "-Xlint:all", "-Werror", filePath)
	cmd.Dir = l.workDir
	out, _ := cmd.CombinedOutput()

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "error:") || strings.Contains(line, "warning:") {
			severity := "error"
			if strings.Contains(line, "warning:") {
				severity = "warning"
			}
			diagnostics = append(diagnostics, Diagnostic{
				File:     filePath,
				Severity: severity,
				Message:  line,
				Source:   "javac",
			})
		}
	}

	return diagnostics, nil
}

// CheckAllFiles checks syntax for all source files in the workspace
func (l *LSPManager) CheckAllFiles() (map[string][]Diagnostic, error) {
	results := make(map[string][]Diagnostic)

	err := filepath.Walk(l.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip common non-source directories
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") ||
			strings.Contains(path, "vendor") || strings.Contains(path, "__pycache__") {
			return nil
		}

		diagnostics, err := l.CheckSyntax(path)
		if err == nil && len(diagnostics) > 0 {
			relPath, _ := filepath.Rel(l.workDir, path)
			results[relPath] = diagnostics
		}

		return nil
	})

	return results, err
}

// DiagnosticsToJSON converts diagnostics to JSON
func DiagnosticsToJSON(diagnostics []Diagnostic) ([]byte, error) {
	return json.MarshalIndent(diagnostics, "", "  ")
}
