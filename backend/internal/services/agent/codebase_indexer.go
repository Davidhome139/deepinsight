package agent

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CodebaseIndexer provides intelligent codebase context and dependency analysis
type CodebaseIndexer struct {
	rootDir       string
	index         *CodebaseIndex
	mu            sync.RWMutex
	lastIndexTime time.Time
}

// CodebaseIndex represents the indexed codebase structure
type CodebaseIndex struct {
	Files        map[string]*FileInfo `json:"files"`
	Dependencies map[string][]string  `json:"dependencies"` // file -> dependencies
	Dependents   map[string][]string  `json:"dependents"`   // file -> files that depend on it
	Symbols      map[string][]*Symbol `json:"symbols"`      // file -> symbols defined
	SymbolIndex  map[string]string    `json:"symbol_index"` // symbol name -> file
	ImportGraph  map[string][]string  `json:"import_graph"` // file -> imported files
	ExportGraph  map[string][]string  `json:"export_graph"` // file -> exported symbols
	ProjectType  string               `json:"project_type"` // go, python, javascript, etc.
	TotalFiles   int                  `json:"total_files"`
	TotalLines   int                  `json:"total_lines"`
	IndexedAt    time.Time            `json:"indexed_at"`
}

// FileInfo represents indexed file information
type FileInfo struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Language     string    `json:"language"`
	Size         int64     `json:"size"`
	Lines        int       `json:"lines"`
	Hash         string    `json:"hash"`
	Imports      []string  `json:"imports"`
	Exports      []string  `json:"exports"`
	Functions    []string  `json:"functions"`
	Classes      []string  `json:"classes"`
	ModifiedAt   time.Time `json:"modified_at"`
}

// Symbol represents a code symbol (function, class, variable)
type Symbol struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // function, class, variable, interface, const
	File       string `json:"file"`
	Line       int    `json:"line"`
	Exported   bool   `json:"exported"`
	Signature  string `json:"signature,omitempty"`
	DocComment string `json:"doc_comment,omitempty"`
}

// DependencyChange represents a change that affects dependencies
type DependencyChange struct {
	File            string   `json:"file"`
	AffectedFiles   []string `json:"affected_files"`
	AffectedSymbols []string `json:"affected_symbols"`
	ChangeType      string   `json:"change_type"` // add, modify, delete
}

// NewCodebaseIndexer creates a new codebase indexer
func NewCodebaseIndexer(rootDir string) *CodebaseIndexer {
	return &CodebaseIndexer{
		rootDir: rootDir,
		index: &CodebaseIndex{
			Files:        make(map[string]*FileInfo),
			Dependencies: make(map[string][]string),
			Dependents:   make(map[string][]string),
			Symbols:      make(map[string][]*Symbol),
			SymbolIndex:  make(map[string]string),
			ImportGraph:  make(map[string][]string),
			ExportGraph:  make(map[string][]string),
		},
	}
}

// Index indexes the entire codebase
func (ci *CodebaseIndexer) Index() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// Reset index
	ci.index = &CodebaseIndex{
		Files:        make(map[string]*FileInfo),
		Dependencies: make(map[string][]string),
		Dependents:   make(map[string][]string),
		Symbols:      make(map[string][]*Symbol),
		SymbolIndex:  make(map[string]string),
		ImportGraph:  make(map[string][]string),
		ExportGraph:  make(map[string][]string),
		IndexedAt:    time.Now(),
	}

	// Detect project type
	ci.index.ProjectType = ci.detectProjectType()

	// Walk directory and index files
	err := filepath.Walk(ci.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" ||
				name == "vendor" || name == "dist" || name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Index source files
		if ci.isSourceFile(path) {
			ci.indexFile(path, info)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Build dependency graph
	ci.buildDependencyGraph()

	ci.lastIndexTime = time.Now()
	return nil
}

// indexFile indexes a single source file
func (ci *CodebaseIndexer) indexFile(path string, info os.FileInfo) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	relPath, _ := filepath.Rel(ci.rootDir, path)
	hash := md5.Sum(content)

	fileInfo := &FileInfo{
		Path:         path,
		RelativePath: relPath,
		Language:     ci.detectLanguage(path),
		Size:         info.Size(),
		Lines:        strings.Count(string(content), "\n") + 1,
		Hash:         hex.EncodeToString(hash[:]),
		ModifiedAt:   info.ModTime(),
	}

	// Extract imports and exports based on language
	fileInfo.Imports = ci.extractImports(string(content), fileInfo.Language)
	fileInfo.Exports = ci.extractExports(string(content), fileInfo.Language)
	fileInfo.Functions = ci.extractFunctions(string(content), fileInfo.Language)
	fileInfo.Classes = ci.extractClasses(string(content), fileInfo.Language)

	ci.index.Files[relPath] = fileInfo
	ci.index.TotalFiles++
	ci.index.TotalLines += fileInfo.Lines

	// Extract and index symbols
	symbols := ci.extractSymbols(string(content), fileInfo.Language, relPath)
	ci.index.Symbols[relPath] = symbols
	for _, sym := range symbols {
		ci.index.SymbolIndex[sym.Name] = relPath
	}
}

// detectProjectType detects the project type from config files
func (ci *CodebaseIndexer) detectProjectType() string {
	if _, err := os.Stat(filepath.Join(ci.rootDir, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(ci.rootDir, "package.json")); err == nil {
		return "javascript"
	}
	if _, err := os.Stat(filepath.Join(ci.rootDir, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(ci.rootDir, "Cargo.toml")); err == nil {
		return "rust"
	}
	if _, err := os.Stat(filepath.Join(ci.rootDir, "pom.xml")); err == nil {
		return "java"
	}
	return "unknown"
}

// detectLanguage detects the programming language from file extension
func (ci *CodebaseIndexer) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".vue":
		return "vue"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}

// isSourceFile checks if a file is a source file
func (ci *CodebaseIndexer) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	sourceExts := []string{".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".rs", ".c", ".h", ".cpp", ".hpp", ".vue", ".rb", ".php"}
	for _, e := range sourceExts {
		if ext == e {
			return true
		}
	}
	return false
}

// extractImports extracts import statements from source code
func (ci *CodebaseIndexer) extractImports(content, lang string) []string {
	var imports []string
	var patterns []*regexp.Regexp

	switch lang {
	case "go":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`import\s+"([^"]+)"`),
			regexp.MustCompile(`"([^"]+)"`), // Within import block
		}
	case "python":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`import\s+(\w+)`),
			regexp.MustCompile(`from\s+(\S+)\s+import`),
		}
	case "javascript", "typescript", "jsx", "tsx":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`import\s+.*from\s+['"]([^'"]+)['"]`),
			regexp.MustCompile(`require\(['"]([^'"]+)['"]\)`),
		}
	case "java":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`import\s+([^;]+);`),
		}
	case "rust":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`use\s+([^;]+);`),
		}
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				imports = append(imports, match[1])
			}
		}
	}

	return imports
}

// extractExports extracts exported symbols from source code
func (ci *CodebaseIndexer) extractExports(content, lang string) []string {
	var exports []string
	var patterns []*regexp.Regexp

	switch lang {
	case "go":
		// Exported Go symbols start with uppercase
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`func\s+([A-Z]\w*)\s*\(`),
			regexp.MustCompile(`type\s+([A-Z]\w*)\s+`),
			regexp.MustCompile(`var\s+([A-Z]\w*)\s+`),
			regexp.MustCompile(`const\s+([A-Z]\w*)\s+`),
		}
	case "javascript", "typescript", "jsx", "tsx":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`export\s+(?:default\s+)?(?:function|class|const|let|var)\s+(\w+)`),
			regexp.MustCompile(`export\s+\{\s*([^}]+)\s*\}`),
		}
	case "python":
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`^def\s+(\w+)\s*\(`),
			regexp.MustCompile(`^class\s+(\w+)`),
		}
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				exports = append(exports, match[1])
			}
		}
	}

	return exports
}

// extractFunctions extracts function names from source code
func (ci *CodebaseIndexer) extractFunctions(content, lang string) []string {
	var functions []string
	var pattern *regexp.Regexp

	switch lang {
	case "go":
		pattern = regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	case "python":
		pattern = regexp.MustCompile(`def\s+(\w+)\s*\(`)
	case "javascript", "typescript", "jsx", "tsx":
		pattern = regexp.MustCompile(`(?:function\s+(\w+)|(\w+)\s*[:=]\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))`)
	case "java":
		pattern = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static)?\s*\w+\s+(\w+)\s*\(`)
	}

	if pattern != nil {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					functions = append(functions, match[i])
					break
				}
			}
		}
	}

	return functions
}

// extractClasses extracts class names from source code
func (ci *CodebaseIndexer) extractClasses(content, lang string) []string {
	var classes []string
	var pattern *regexp.Regexp

	switch lang {
	case "go":
		pattern = regexp.MustCompile(`type\s+(\w+)\s+struct`)
	case "python":
		pattern = regexp.MustCompile(`class\s+(\w+)`)
	case "javascript", "typescript", "jsx", "tsx":
		pattern = regexp.MustCompile(`class\s+(\w+)`)
	case "java":
		pattern = regexp.MustCompile(`class\s+(\w+)`)
	}

	if pattern != nil {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				classes = append(classes, match[1])
			}
		}
	}

	return classes
}

// extractSymbols extracts all symbols from source code
func (ci *CodebaseIndexer) extractSymbols(content, lang, filePath string) []*Symbol {
	var symbols []*Symbol
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Extract function symbols
		funcPattern := ci.getFunctionPattern(lang)
		if funcPattern != nil {
			if match := funcPattern.FindStringSubmatch(line); len(match) > 1 {
				symbols = append(symbols, &Symbol{
					Name:     match[1],
					Type:     "function",
					File:     filePath,
					Line:     lineNum + 1,
					Exported: ci.isExported(match[1], lang),
				})
			}
		}

		// Extract class/struct symbols
		classPattern := ci.getClassPattern(lang)
		if classPattern != nil {
			if match := classPattern.FindStringSubmatch(line); len(match) > 1 {
				symbols = append(symbols, &Symbol{
					Name:     match[1],
					Type:     "class",
					File:     filePath,
					Line:     lineNum + 1,
					Exported: ci.isExported(match[1], lang),
				})
			}
		}
	}

	return symbols
}

func (ci *CodebaseIndexer) getFunctionPattern(lang string) *regexp.Regexp {
	switch lang {
	case "go":
		return regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	case "python":
		return regexp.MustCompile(`def\s+(\w+)\s*\(`)
	case "javascript", "typescript":
		return regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?function)`)
	default:
		return nil
	}
}

func (ci *CodebaseIndexer) getClassPattern(lang string) *regexp.Regexp {
	switch lang {
	case "go":
		return regexp.MustCompile(`type\s+(\w+)\s+struct`)
	case "python", "javascript", "typescript", "java":
		return regexp.MustCompile(`class\s+(\w+)`)
	default:
		return nil
	}
}

func (ci *CodebaseIndexer) isExported(name, lang string) bool {
	switch lang {
	case "go":
		return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
	case "python":
		return !strings.HasPrefix(name, "_")
	default:
		return true
	}
}

// buildDependencyGraph builds the dependency graph from imports
func (ci *CodebaseIndexer) buildDependencyGraph() {
	for filePath, fileInfo := range ci.index.Files {
		ci.index.ImportGraph[filePath] = fileInfo.Imports
		ci.index.ExportGraph[filePath] = fileInfo.Exports

		// Resolve imports to local files
		for _, imp := range fileInfo.Imports {
			// Try to find the imported file in the index
			for otherPath := range ci.index.Files {
				if strings.Contains(otherPath, imp) || strings.HasSuffix(otherPath, imp+"."+fileInfo.Language) {
					ci.index.Dependencies[filePath] = append(ci.index.Dependencies[filePath], otherPath)
					ci.index.Dependents[otherPath] = append(ci.index.Dependents[otherPath], filePath)
				}
			}
		}
	}
}

// GetContext returns relevant context for a task
func (ci *CodebaseIndexer) GetContext(task string, maxFiles int) *CodebaseContext {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	ctx := &CodebaseContext{
		ProjectType:    ci.index.ProjectType,
		TotalFiles:     ci.index.TotalFiles,
		TotalLines:     ci.index.TotalLines,
		RelevantFiles:  make([]*FileInfo, 0),
		RelatedSymbols: make([]*Symbol, 0),
	}

	// Find relevant files based on task keywords
	keywords := ci.extractKeywords(task)
	scores := make(map[string]int)

	for filePath, fileInfo := range ci.index.Files {
		score := ci.calculateRelevanceScore(fileInfo, keywords)
		if score > 0 {
			scores[filePath] = score
		}
	}

	// Sort by relevance and take top files
	sortedFiles := ci.sortByScore(scores, maxFiles)
	for _, filePath := range sortedFiles {
		ctx.RelevantFiles = append(ctx.RelevantFiles, ci.index.Files[filePath])
		// Add symbols from relevant files
		ctx.RelatedSymbols = append(ctx.RelatedSymbols, ci.index.Symbols[filePath]...)
	}

	return ctx
}

// CodebaseContext represents contextual information for a task
type CodebaseContext struct {
	ProjectType    string      `json:"project_type"`
	TotalFiles     int         `json:"total_files"`
	TotalLines     int         `json:"total_lines"`
	RelevantFiles  []*FileInfo `json:"relevant_files"`
	RelatedSymbols []*Symbol   `json:"related_symbols"`
}

// GetAffectedFiles returns files that would be affected by changing a symbol
func (ci *CodebaseIndexer) GetAffectedFiles(symbolName string) []string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	affected := make(map[string]bool)

	// Find the file containing the symbol
	if filePath, ok := ci.index.SymbolIndex[symbolName]; ok {
		affected[filePath] = true

		// Find all files that depend on this file
		ci.findDependentsRecursive(filePath, affected, 3) // Max 3 levels deep
	}

	result := make([]string, 0, len(affected))
	for path := range affected {
		result = append(result, path)
	}
	return result
}

func (ci *CodebaseIndexer) findDependentsRecursive(filePath string, affected map[string]bool, depth int) {
	if depth <= 0 {
		return
	}

	for _, dependent := range ci.index.Dependents[filePath] {
		if !affected[dependent] {
			affected[dependent] = true
			ci.findDependentsRecursive(dependent, affected, depth-1)
		}
	}
}

func (ci *CodebaseIndexer) extractKeywords(task string) []string {
	// Simple keyword extraction - split by spaces and filter
	words := strings.Fields(strings.ToLower(task))
	keywords := make([]string, 0)
	stopWords := map[string]bool{"the": true, "a": true, "an": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true, "to": true, "for": true}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

func (ci *CodebaseIndexer) calculateRelevanceScore(fileInfo *FileInfo, keywords []string) int {
	score := 0
	pathLower := strings.ToLower(fileInfo.RelativePath)

	for _, keyword := range keywords {
		if strings.Contains(pathLower, keyword) {
			score += 10
		}
		for _, fn := range fileInfo.Functions {
			if strings.Contains(strings.ToLower(fn), keyword) {
				score += 5
			}
		}
		for _, cls := range fileInfo.Classes {
			if strings.Contains(strings.ToLower(cls), keyword) {
				score += 5
			}
		}
	}

	return score
}

func (ci *CodebaseIndexer) sortByScore(scores map[string]int, max int) []string {
	type fileScore struct {
		path  string
		score int
	}

	sorted := make([]fileScore, 0, len(scores))
	for path, score := range scores {
		sorted = append(sorted, fileScore{path, score})
	}

	// Simple bubble sort (sufficient for small sets)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := make([]string, 0, max)
	for i := 0; i < len(sorted) && i < max; i++ {
		result = append(result, sorted[i].path)
	}
	return result
}

// GetIndexJSON returns the index as JSON for debugging/inspection
func (ci *CodebaseIndexer) GetIndexJSON() ([]byte, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return json.MarshalIndent(ci.index, "", "  ")
}

// GetFileContent returns the content of a file with optional line range
func (ci *CodebaseIndexer) GetFileContent(filePath string, startLine, endLine int) (string, error) {
	fullPath := filepath.Join(ci.rootDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	if startLine <= 0 && endLine <= 0 {
		return string(content), nil
	}

	lines := strings.Split(string(content), "\n")
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 || endLine > len(lines) {
		endLine = len(lines)
	}

	return strings.Join(lines[startLine-1:endLine], "\n"), nil
}
