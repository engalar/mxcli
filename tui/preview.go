package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// buildDescribeCmd constructs the correct DESCRIBE command for a node type and qualified name.
// Returns "" for virtual container nodes that have no valid DESCRIBE syntax.
// Multi-word security types require special syntax that differs from the generic pattern.
func buildDescribeCmd(nodeType, qualifiedName string) string {
	switch strings.ToLower(nodeType) {
	case "systemoverview":
		return "SHOW STRUCTURE DEPTH 2"
	case "security", "category", "domainmodel", "navigation", "projectsecurity", "navprofile",
		"nanoflow": // DESCRIBE NANOFLOW not implemented in executor
		return ""
	case "javaaction":
		return "DESCRIBE JAVA ACTION " + qualifiedName
	case "imagecollection":
		return "DESCRIBE IMAGE COLLECTION " + qualifiedName
	case "modulerole":
		return "DESCRIBE MODULE ROLE " + qualifiedName
	case "userrole":
		return fmt.Sprintf("DESCRIBE USER ROLE '%s'", qualifiedName)
	case "demouser":
		return fmt.Sprintf("DESCRIBE DEMO USER '%s'", qualifiedName)
	default:
		return fmt.Sprintf("DESCRIBE %s %s", strings.ToUpper(nodeType), qualifiedName)
	}
}

// PreviewMode selects what format to request from mxcli.
type PreviewMode int

const (
	PreviewMDL PreviewMode = iota
	PreviewNDSL
)

// PreviewResult holds cached preview content.
type PreviewResult struct {
	Content       string
	ImagePaths    []string // image file paths for lazy rendering (excluded from yank)
	HighlightType string   // "mdl" / "ndsl" / "plain"
}

// PreviewReadyMsg is sent when async preview content is available.
type PreviewReadyMsg struct {
	Content       string
	ImagePaths    []string // image file paths for lazy rendering (excluded from yank)
	HighlightType string
	NodeKey       string
}

// PreviewLoadingMsg is sent when a preview fetch starts.
type PreviewLoadingMsg struct {
	NodeKey string
}

// PreviewEngine manages async preview fetching with caching and cancellation.
type PreviewEngine struct {
	mu          sync.Mutex
	cache       map[string]PreviewResult
	cancelFunc  context.CancelFunc
	mxcliPath   string
	projectPath string
}

// NewPreviewEngine creates a PreviewEngine for the given mxcli binary and project.
func NewPreviewEngine(mxcliPath, projectPath string) *PreviewEngine {
	return &PreviewEngine{
		cache:       make(map[string]PreviewResult),
		mxcliPath:   mxcliPath,
		projectPath: projectPath,
	}
}

// cacheKey builds a deterministic cache key from type, name, and mode.
func cacheKey(nodeType, qualifiedName string, mode PreviewMode) string {
	return fmt.Sprintf("%s:%s:%d", nodeType, qualifiedName, mode)
}

// RequestPreview returns a tea.Cmd that fetches preview content.
// On cache hit it returns immediately; on miss it cancels any in-flight
// request and spawns a new goroutine.
func (e *PreviewEngine) RequestPreview(nodeType, qualifiedName string, mode PreviewMode) tea.Cmd {
	key := cacheKey(nodeType, qualifiedName, mode)
	Trace("preview: request type=%q name=%q mode=%d key=%q", nodeType, qualifiedName, mode, key)

	e.mu.Lock()
	if cached, ok := e.cache[key]; ok {
		e.mu.Unlock()
		Trace("preview: cache hit key=%q", key)
		return func() tea.Msg {
			return PreviewReadyMsg{
				Content:       cached.Content,
				HighlightType: cached.HighlightType,
				NodeKey:       key,
			}
		}
	}

	// Cancel previous in-flight request.
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFunc = cancel
	e.mu.Unlock()

	return tea.Batch(
		func() tea.Msg { return PreviewLoadingMsg{NodeKey: key} },
		func() tea.Msg {
			content, highlightType := e.fetch(ctx, nodeType, qualifiedName, mode)

			// If cancelled, discard result silently.
			if ctx.Err() != nil {
				return nil
			}

			// Apply syntax highlighting before caching.
			highlighted := content
			switch highlightType {
			case "ndsl":
				highlighted = HighlightNDSL(content)
			case "mdl":
				highlighted = DetectAndHighlight(content)
			}

			// Extract image paths for lazy rendering (size-aware, excluded from yank).
			var imagePaths []string
			if strings.ToLower(nodeType) == "imagecollection" && mode == PreviewMDL {
				imagePaths = extractImagePaths(content)
			}

			result := PreviewResult{Content: highlighted, ImagePaths: imagePaths, HighlightType: highlightType}
			e.mu.Lock()
			e.cache[key] = result
			e.mu.Unlock()

			return PreviewReadyMsg{
				Content:       highlighted,
				ImagePaths:    imagePaths,
				HighlightType: highlightType,
				NodeKey:       key,
			}
		},
	)
}

// stripJavaCodeBlock removes the "AS $$...$$" section from a javaaction DESCRIBE
// output so the TUI preview shows only the MDL signature, not the Java implementation.
func stripJavaCodeBlock(s string) string {
	const open = "\nAS $$"
	start := strings.Index(s, open)
	if start == -1 {
		return s
	}
	const close = "\n$$"
	end := strings.Index(s[start+len(open):], close)
	if end == -1 {
		return s[:start] + ";"
	}
	after := s[start+len(open)+end+len(close):]
	return s[:start] + after
}

// fetch runs the mxcli subprocess and returns raw content + highlight type.
func (e *PreviewEngine) fetch(ctx context.Context, nodeType, qualifiedName string, mode PreviewMode) (string, string) {
	var args []string
	var highlightType string

	switch mode {
	case PreviewNDSL:
		bsonType := inferBsonType(nodeType)
		if bsonType == "" {
			return fmt.Sprintf("Type %q not supported for BSON dump", nodeType), "plain"
		}
		args = []string{"bson", "dump", "-p", e.projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qualifiedName}
		highlightType = "ndsl"
	default: // PreviewMDL
		mdlCmd := buildDescribeCmd(nodeType, qualifiedName)
		if mdlCmd == "" {
			return "Select a document to preview", "plain"
		}
		args = []string{"-p", e.projectPath, "-c", mdlCmd}
		highlightType = "mdl"
	}

	cmd := exec.CommandContext(ctx, e.mxcliPath, args...)
	out, err := cmd.CombinedOutput()
	content := StripBanner(string(out))

	if err != nil {
		if ctx.Err() != nil {
			return "", "plain" // cancelled — caller checks ctx.Err()
		}
		return "Error: " + strings.TrimSpace(content), "plain"
	}

	// For javaaction MDL preview, strip the AS $$...$$ Java code block so the
	// preview shows only the signature, not the full implementation.
	if strings.ToLower(nodeType) == "javaaction" && mode == PreviewMDL {
		content = stripJavaCodeBlock(content)
	}

	return content, highlightType
}

// ClearCache discards all cached preview results.
func (e *PreviewEngine) ClearCache() {
	e.mu.Lock()
	e.cache = make(map[string]PreviewResult)
	e.mu.Unlock()
}

// SetProjectPath updates the project path and clears the cache.
func (e *PreviewEngine) SetProjectPath(path string) {
	e.mu.Lock()
	e.projectPath = path
	e.cache = make(map[string]PreviewResult)
	e.mu.Unlock()
}

// Cancel stops any in-flight preview request.
func (e *PreviewEngine) Cancel() {
	e.mu.Lock()
	if e.cancelFunc != nil {
		e.cancelFunc()
		e.cancelFunc = nil
	}
	e.mu.Unlock()
}
