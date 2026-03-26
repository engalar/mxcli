package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// currentSessionVersion is the schema version for session files.
const currentSessionVersion = 1

// TUISession represents the serializable state of the TUI application.
type TUISession struct {
	Version   int         `json:"version"`
	Timestamp string      `json:"timestamp"`
	Tabs      []TabState  `json:"tabs"`
	ActiveTab int         `json:"activeTab"`
	ViewStack []ViewState `json:"viewStack"`

	CheckNavActive bool `json:"checkNavActive"`
	CheckNavIndex  int  `json:"checkNavIndex"`
}

// TabState captures the navigable state of a single tab.
type TabState struct {
	ProjectPath  string   `json:"projectPath"`
	MillerPath   []string `json:"millerPath"`
	SelectedNode string   `json:"selectedNode"`
	PreviewMode  string   `json:"previewMode"`
}

// ViewState captures one entry in the view stack.
type ViewState struct {
	Type   string `json:"type"`
	Title  string `json:"title,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// sessionFilePath returns the path to the TUI session file.
func sessionFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mxcli", "tui-session.json")
}

// SaveSession serializes the given session to disk.
func SaveSession(session *TUISession) error {
	path := sessionFilePath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

// LoadSession reads and parses the session file.
// Returns (nil, nil) if the file does not exist.
func LoadSession() (*TUISession, error) {
	path := sessionFilePath()
	if path == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var session TUISession
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}

	// Ignore future versions we don't understand
	if session.Version > currentSessionVersion {
		return nil, nil
	}

	return &session, nil
}

// ExtractSession captures the current TUI state into a serializable session.
func ExtractSession(app *App) *TUISession {
	session := &TUISession{
		Version:        currentSessionVersion,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		ActiveTab:      app.activeTab,
		CheckNavActive: app.checkNavActive,
		CheckNavIndex:  app.checkNavIndex,
	}

	// Extract tab states
	for i := range app.tabs {
		tab := &app.tabs[i]
		ts := TabState{
			ProjectPath: tab.ProjectPath,
			MillerPath:  tab.Miller.Breadcrumb(),
		}

		// Capture selected node name
		if node := tab.Miller.SelectedNode(); node != nil {
			ts.SelectedNode = node.QualifiedName
			if ts.SelectedNode == "" {
				ts.SelectedNode = node.Label
			}
		}

		// Capture preview mode
		if tab.Miller.preview.mode == PreviewNDSL {
			ts.PreviewMode = "NDSL"
		} else {
			ts.PreviewMode = "MDL"
		}

		session.Tabs = append(session.Tabs, ts)
	}

	// Extract view stack
	session.ViewStack = extractViewStack(&app.views)

	return session
}

// extractViewStack converts the ViewStack into serializable ViewState entries.
func extractViewStack(vs *ViewStack) []ViewState {
	states := []ViewState{viewToState(vs.Base())}
	for _, v := range vs.stack {
		states = append(states, viewToState(v))
	}
	return states
}

// viewToState converts a View into a ViewState based on its mode.
func viewToState(v View) ViewState {
	switch v.Mode() {
	case ModeBrowser:
		return ViewState{Type: "browser"}
	case ModeOverlay:
		vs := ViewState{Type: "overlay"}
		if ov, ok := v.(OverlayView); ok {
			vs.Title = ov.overlay.title
			if ov.refreshable {
				vs.Filter = ov.checkFilter
			}
		}
		return vs
	case ModeCompare:
		return ViewState{Type: "compare"}
	case ModeDiff:
		return ViewState{Type: "diff"}
	case ModeExec:
		return ViewState{Type: "exec"}
	default:
		return ViewState{Type: v.Mode().String()}
	}
}
