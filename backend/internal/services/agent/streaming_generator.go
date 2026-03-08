package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// StreamingCodeGenerator generates code with real-time streaming updates
type StreamingCodeGenerator struct {
	eventCallback   func(event CodeStreamEvent)
	buffer          strings.Builder
	currentFile     string
	currentLanguage string
	mu              sync.Mutex
}

// CodeStreamEvent represents a streaming code generation event
type CodeStreamEvent struct {
	Type      string `json:"type"` // code_chunk, file_start, file_complete, generation_complete
	FilePath  string `json:"file_path,omitempty"`
	Language  string `json:"language,omitempty"`
	Content   string `json:"content,omitempty"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// NewStreamingCodeGenerator creates a new streaming code generator
func NewStreamingCodeGenerator(callback func(event CodeStreamEvent)) *StreamingCodeGenerator {
	return &StreamingCodeGenerator{
		eventCallback: callback,
	}
}

// ProcessChunk processes a streaming chunk and emits events
func (s *StreamingCodeGenerator) ProcessChunk(chunk string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer.WriteString(chunk)
	content := s.buffer.String()

	// Look for file markers
	filePattern := regexp.MustCompile(`### File:\s*(.+?)\n`)
	codeStartPattern := regexp.MustCompile("```(\\w+)?\\n")
	codeEndPattern := regexp.MustCompile("```\\s*\\n?")

	// Check for file start
	if fileMatch := filePattern.FindStringSubmatch(content); len(fileMatch) > 1 {
		newFile := strings.TrimSpace(fileMatch[1])
		if newFile != s.currentFile {
			// Emit file start event
			s.currentFile = newFile
			s.emitEvent(CodeStreamEvent{
				Type:     "file_start",
				FilePath: newFile,
				Language: detectLanguageFromPath(newFile),
			})
		}
	}

	// Check for code block start
	if codeMatch := codeStartPattern.FindStringSubmatch(content); len(codeMatch) > 0 {
		if len(codeMatch) > 1 && codeMatch[1] != "" {
			s.currentLanguage = codeMatch[1]
		}
	}

	// Extract code content and emit chunks
	codeStart := codeStartPattern.FindStringIndex(content)
	codeEnd := codeEndPattern.FindStringIndex(content)

	if codeStart != nil && codeEnd != nil && codeEnd[0] > codeStart[1] {
		// Complete code block found
		codeContent := content[codeStart[1]:codeEnd[0]]
		s.emitEvent(CodeStreamEvent{
			Type:     "code_chunk",
			FilePath: s.currentFile,
			Language: s.currentLanguage,
			Content:  codeContent,
		})

		s.emitEvent(CodeStreamEvent{
			Type:     "file_complete",
			FilePath: s.currentFile,
			Language: s.currentLanguage,
			Content:  codeContent,
		})

		// Clear buffer after code block
		s.buffer.Reset()
		s.buffer.WriteString(content[codeEnd[1]:])
	} else if codeStart != nil {
		// Partial code block - emit incremental chunks
		partialContent := content[codeStart[1]:]
		lines := strings.Split(partialContent, "\n")
		if len(lines) > 1 {
			// Emit all complete lines
			completeLines := strings.Join(lines[:len(lines)-1], "\n")
			s.emitEvent(CodeStreamEvent{
				Type:     "code_chunk",
				FilePath: s.currentFile,
				Language: s.currentLanguage,
				Content:  completeLines,
			})
		}
	}
}

// emitEvent emits a code stream event
func (s *StreamingCodeGenerator) emitEvent(event CodeStreamEvent) {
	event.Timestamp = time.Now().UnixMilli()
	if s.eventCallback != nil {
		s.eventCallback(event)
	}
}

// Complete signals that generation is complete
func (s *StreamingCodeGenerator) Complete() {
	s.emitEvent(CodeStreamEvent{
		Type: "generation_complete",
	})
}

func detectLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "jsx",
		".tsx":  "tsx",
		".java": "java",
		".rs":   "rust",
		".vue":  "vue",
		".html": "html",
		".css":  "css",
		".sql":  "sql",
		".sh":   "bash",
		".yaml": "yaml",
		".json": "json",
	}
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "text"
}

// MultiFileCoordinator coordinates changes across multiple files
type MultiFileCoordinator struct {
	changes      map[string]*FileChange
	dependencies map[string][]string // file -> files that depend on it
	changeOrder  []string            // Order to apply changes
	indexer      *CodebaseIndexer
	mu           sync.RWMutex
}

// FileChange represents a proposed change to a file
type FileChange struct {
	FilePath     string         `json:"file_path"`
	ChangeType   string         `json:"change_type"` // create, modify, delete
	OldContent   string         `json:"old_content,omitempty"`
	NewContent   string         `json:"new_content"`
	Hunks        []DiffHunk     `json:"hunks,omitempty"`
	Dependencies []string       `json:"dependencies"`
	Dependents   []string       `json:"dependents"`
	Status       string         `json:"status"` // pending, approved, applied, rejected
	Metadata     ChangeMetadata `json:"metadata"`
}

// DiffHunk represents a single diff hunk
type DiffHunk struct {
	OldStart int    `json:"old_start"`
	OldLines int    `json:"old_lines"`
	NewStart int    `json:"new_start"`
	NewLines int    `json:"new_lines"`
	Content  string `json:"content"`
	Context  string `json:"context"`
}

// ChangeMetadata contains metadata about a change
type ChangeMetadata struct {
	Author       string   `json:"author"`
	Reason       string   `json:"reason"`
	AffectedFns  []string `json:"affected_functions"`
	AffectedClss []string `json:"affected_classes"`
	RiskLevel    string   `json:"risk_level"` // low, medium, high
	TestRequired bool     `json:"test_required"`
}

// NewMultiFileCoordinator creates a new multi-file coordinator
func NewMultiFileCoordinator(indexer *CodebaseIndexer) *MultiFileCoordinator {
	return &MultiFileCoordinator{
		changes:      make(map[string]*FileChange),
		dependencies: make(map[string][]string),
		changeOrder:  make([]string, 0),
		indexer:      indexer,
	}
}

// ProposeChange proposes a change to a file
func (m *MultiFileCoordinator) ProposeChange(change *FileChange) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Analyze dependencies if indexer is available
	if m.indexer != nil {
		affected := m.indexer.GetAffectedFiles(change.FilePath)
		change.Dependents = affected
	}

	// Calculate diff hunks
	if change.OldContent != "" && change.NewContent != "" {
		change.Hunks = m.calculateDiffHunks(change.OldContent, change.NewContent)
	}

	// Assess risk level
	change.Metadata.RiskLevel = m.assessRiskLevel(change)

	// Add to changes
	m.changes[change.FilePath] = change

	// Update change order based on dependencies
	m.updateChangeOrder()

	return nil
}

// calculateDiffHunks calculates diff hunks between old and new content
func (m *MultiFileCoordinator) calculateDiffHunks(oldContent, newContent string) []DiffHunk {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	hunks := make([]DiffHunk, 0)

	// Simple line-by-line diff (for production, use a proper diff algorithm)
	var currentHunk *DiffHunk
	hunkContent := strings.Builder{}

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if currentHunk == nil {
				currentHunk = &DiffHunk{
					OldStart: i + 1,
					NewStart: i + 1,
				}
				hunkContent.Reset()
			}

			if oldLine != "" {
				hunkContent.WriteString("-" + oldLine + "\n")
				currentHunk.OldLines++
			}
			if newLine != "" {
				hunkContent.WriteString("+" + newLine + "\n")
				currentHunk.NewLines++
			}
		} else if currentHunk != nil {
			// End of hunk
			currentHunk.Content = hunkContent.String()
			hunks = append(hunks, *currentHunk)
			currentHunk = nil
		}
	}

	// Finalize last hunk
	if currentHunk != nil {
		currentHunk.Content = hunkContent.String()
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// assessRiskLevel assesses the risk level of a change
func (m *MultiFileCoordinator) assessRiskLevel(change *FileChange) string {
	// High risk: core files, many dependents, large changes
	if len(change.Dependents) > 10 {
		return "high"
	}

	// Check for critical patterns
	criticalPatterns := []string{"main.", "config", "auth", "database", "security"}
	for _, pattern := range criticalPatterns {
		if strings.Contains(strings.ToLower(change.FilePath), pattern) {
			return "high"
		}
	}

	// Medium risk: some dependents, moderate changes
	if len(change.Dependents) > 3 || len(change.Hunks) > 5 {
		return "medium"
	}

	return "low"
}

// updateChangeOrder updates the order in which changes should be applied
func (m *MultiFileCoordinator) updateChangeOrder() {
	// Topological sort based on dependencies
	visited := make(map[string]bool)
	order := make([]string, 0)

	var visit func(file string)
	visit = func(file string) {
		if visited[file] {
			return
		}
		visited[file] = true

		// Visit dependencies first
		for _, dep := range m.changes[file].Dependencies {
			if _, ok := m.changes[dep]; ok {
				visit(dep)
			}
		}

		order = append(order, file)
	}

	for file := range m.changes {
		visit(file)
	}

	m.changeOrder = order
}

// GetChangePlan returns the planned changes in order
func (m *MultiFileCoordinator) GetChangePlan() []*FileChange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plan := make([]*FileChange, 0, len(m.changeOrder))
	for _, file := range m.changeOrder {
		if change, ok := m.changes[file]; ok {
			plan = append(plan, change)
		}
	}
	return plan
}

// ApproveChange approves a pending change
func (m *MultiFileCoordinator) ApproveChange(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	change, ok := m.changes[filePath]
	if !ok {
		return fmt.Errorf("change for %s not found", filePath)
	}

	change.Status = "approved"
	return nil
}

// ApproveAll approves all pending changes
func (m *MultiFileCoordinator) ApproveAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, change := range m.changes {
		if change.Status == "pending" {
			change.Status = "approved"
		}
	}
}

// RejectChange rejects a pending change
func (m *MultiFileCoordinator) RejectChange(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	change, ok := m.changes[filePath]
	if !ok {
		return fmt.Errorf("change for %s not found", filePath)
	}

	change.Status = "rejected"
	return nil
}

// GetChangesSummary returns a summary of all proposed changes
func (m *MultiFileCoordinator) GetChangesSummary() *ChangesSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := &ChangesSummary{
		TotalFiles: len(m.changes),
		Changes:    make([]ChangeInfo, 0, len(m.changes)),
	}

	for _, change := range m.changes {
		info := ChangeInfo{
			FilePath:   change.FilePath,
			ChangeType: change.ChangeType,
			RiskLevel:  change.Metadata.RiskLevel,
			Status:     change.Status,
			HunkCount:  len(change.Hunks),
		}

		// Calculate line changes
		for _, hunk := range change.Hunks {
			info.LinesAdded += hunk.NewLines
			info.LinesRemoved += hunk.OldLines
		}

		summary.Changes = append(summary.Changes, info)

		// Update totals
		switch change.ChangeType {
		case "create":
			summary.FilesCreated++
		case "modify":
			summary.FilesModified++
		case "delete":
			summary.FilesDeleted++
		}
		summary.TotalLinesAdded += info.LinesAdded
		summary.TotalLinesRemoved += info.LinesRemoved
	}

	return summary
}

// ChangesSummary provides a summary of all proposed changes
type ChangesSummary struct {
	TotalFiles        int          `json:"total_files"`
	FilesCreated      int          `json:"files_created"`
	FilesModified     int          `json:"files_modified"`
	FilesDeleted      int          `json:"files_deleted"`
	TotalLinesAdded   int          `json:"total_lines_added"`
	TotalLinesRemoved int          `json:"total_lines_removed"`
	Changes           []ChangeInfo `json:"changes"`
}

// ChangeInfo provides info about a single change
type ChangeInfo struct {
	FilePath     string `json:"file_path"`
	ChangeType   string `json:"change_type"`
	RiskLevel    string `json:"risk_level"`
	Status       string `json:"status"`
	HunkCount    int    `json:"hunk_count"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
}

// GetUnifiedDiff returns a unified diff for all changes
func (m *MultiFileCoordinator) GetUnifiedDiff() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	for _, filePath := range m.changeOrder {
		change, ok := m.changes[filePath]
		if !ok {
			continue
		}

		sb.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

		for _, hunk := range change.Hunks {
			sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
				hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines))
			sb.WriteString(hunk.Content)
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateJSON returns changes as JSON for API responses
func (m *MultiFileCoordinator) GenerateJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.MarshalIndent(m.GetChangePlan(), "", "  ")
}
