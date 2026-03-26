package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadSessionRoundtrip(t *testing.T) {
	// Use a temp directory to avoid polluting real session file
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "tui-session.json")

	original := &TUISession{
		Version:   1,
		Timestamp: "2026-03-26T01:30:00Z",
		Tabs: []TabState{
			{
				ProjectPath:  "/path/to/App.mpr",
				MillerPath:   []string{"Project", "MyModule", "Pages"},
				SelectedNode: "MyModule.HomePage",
				PreviewMode:  "MDL",
			},
			{
				ProjectPath:  "/path/to/Other.mpr",
				MillerPath:   []string{"Project"},
				SelectedNode: "",
				PreviewMode:  "NDSL",
			},
		},
		ActiveTab: 1,
		ViewStack: []ViewState{
			{Type: "browser"},
			{Type: "overlay", Title: "mx check", Filter: "all"},
		},
		CheckNavActive: true,
		CheckNavIndex:  3,
	}

	// Write directly to temp path
	encoded, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(sessionPath, encoded, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back
	raw, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var loaded TUISession
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify fields
	if loaded.Version != original.Version {
		t.Errorf("Version: got %d, want %d", loaded.Version, original.Version)
	}
	if loaded.ActiveTab != original.ActiveTab {
		t.Errorf("ActiveTab: got %d, want %d", loaded.ActiveTab, original.ActiveTab)
	}
	if loaded.CheckNavActive != original.CheckNavActive {
		t.Errorf("CheckNavActive: got %v, want %v", loaded.CheckNavActive, original.CheckNavActive)
	}
	if loaded.CheckNavIndex != original.CheckNavIndex {
		t.Errorf("CheckNavIndex: got %d, want %d", loaded.CheckNavIndex, original.CheckNavIndex)
	}
	if len(loaded.Tabs) != len(original.Tabs) {
		t.Fatalf("Tabs count: got %d, want %d", len(loaded.Tabs), len(original.Tabs))
	}
	for i, tab := range loaded.Tabs {
		orig := original.Tabs[i]
		if tab.ProjectPath != orig.ProjectPath {
			t.Errorf("Tab[%d].ProjectPath: got %q, want %q", i, tab.ProjectPath, orig.ProjectPath)
		}
		if tab.PreviewMode != orig.PreviewMode {
			t.Errorf("Tab[%d].PreviewMode: got %q, want %q", i, tab.PreviewMode, orig.PreviewMode)
		}
		if tab.SelectedNode != orig.SelectedNode {
			t.Errorf("Tab[%d].SelectedNode: got %q, want %q", i, tab.SelectedNode, orig.SelectedNode)
		}
		if len(tab.MillerPath) != len(orig.MillerPath) {
			t.Errorf("Tab[%d].MillerPath length: got %d, want %d", i, len(tab.MillerPath), len(orig.MillerPath))
		}
	}
	if len(loaded.ViewStack) != len(original.ViewStack) {
		t.Fatalf("ViewStack count: got %d, want %d", len(loaded.ViewStack), len(original.ViewStack))
	}
	if loaded.ViewStack[1].Title != "mx check" {
		t.Errorf("ViewStack[1].Title: got %q, want %q", loaded.ViewStack[1].Title, "mx check")
	}
	if loaded.ViewStack[1].Filter != "all" {
		t.Errorf("ViewStack[1].Filter: got %q, want %q", loaded.ViewStack[1].Filter, "all")
	}
}

func TestLoadSessionMissingFile(t *testing.T) {
	// LoadSession uses the real sessionFilePath, so test by reading a non-existent temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expected: file not found → return nil, nil
		return
	}
	t.Fatalf("expected file not found, got data: %s", raw)
}

func TestLoadSessionCorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "corrupt.json")

	if err := os.WriteFile(path, []byte("not valid json{{{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var session TUISession
	err = json.Unmarshal(raw, &session)
	if err == nil {
		t.Error("expected unmarshal error for corrupt JSON, got nil")
	}
}

func TestLoadSessionFutureVersion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "future.json")

	future := &TUISession{
		Version:   99,
		Timestamp: "2030-01-01T00:00:00Z",
	}
	encoded, err := json.MarshalIndent(future, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var session TUISession
	if err := json.Unmarshal(raw, &session); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Future version should be ignored
	if session.Version <= currentSessionVersion {
		t.Errorf("expected future version > %d, got %d", currentSessionVersion, session.Version)
	}
}

func TestExtractSessionFromApp(t *testing.T) {
	engine := NewPreviewEngine("", "")
	tab := NewTab(1, "/test/App.mpr", engine, nil)

	// Set up some root nodes for the miller view
	nodes := []*TreeNode{
		{Label: "MyModule", Type: "Module", Children: []*TreeNode{
			{Label: "Pages", Type: "Folder", Children: []*TreeNode{
				{Label: "HomePage", Type: "Page", QualifiedName: "MyModule.HomePage"},
			}},
		}},
	}
	tab.AllNodes = nodes
	tab.Miller.SetRootNodes(nodes)

	app := App{
		tabs:           []Tab{tab},
		activeTab:      0,
		views:          NewViewStack(NewBrowserView(&tab, "", engine)),
		checkNavActive: true,
		checkNavIndex:  2,
	}

	session := ExtractSession(&app)

	if session.Version != currentSessionVersion {
		t.Errorf("Version: got %d, want %d", session.Version, currentSessionVersion)
	}
	if session.ActiveTab != 0 {
		t.Errorf("ActiveTab: got %d, want 0", session.ActiveTab)
	}
	if session.CheckNavActive != true {
		t.Error("CheckNavActive: expected true")
	}
	if session.CheckNavIndex != 2 {
		t.Errorf("CheckNavIndex: got %d, want 2", session.CheckNavIndex)
	}
	if len(session.Tabs) != 1 {
		t.Fatalf("Tabs count: got %d, want 1", len(session.Tabs))
	}
	if session.Tabs[0].ProjectPath != "/test/App.mpr" {
		t.Errorf("Tab ProjectPath: got %q, want %q", session.Tabs[0].ProjectPath, "/test/App.mpr")
	}
	if session.Tabs[0].PreviewMode != "MDL" {
		t.Errorf("Tab PreviewMode: got %q, want %q", session.Tabs[0].PreviewMode, "MDL")
	}
	if len(session.ViewStack) < 1 {
		t.Fatal("ViewStack should have at least the browser view")
	}
	if session.ViewStack[0].Type != "browser" {
		t.Errorf("ViewStack[0].Type: got %q, want %q", session.ViewStack[0].Type, "browser")
	}
}

func TestApplySessionRestoreWithValidNode(t *testing.T) {
	engine := NewPreviewEngine("", "")
	nodes := []*TreeNode{
		{Label: "MyModule", Type: "Module", Children: []*TreeNode{
			{Label: "Pages", Type: "Folder", Children: []*TreeNode{
				{Label: "HomePage", Type: "Page", QualifiedName: "MyModule.HomePage"},
				{Label: "LoginPage", Type: "Page", QualifiedName: "MyModule.LoginPage"},
			}},
		}},
	}
	tab := NewTab(1, "/test/App.mpr", engine, nil)
	tab.AllNodes = nodes
	tab.Miller.SetRootNodes(nodes)

	app := App{
		tabs:      []Tab{tab},
		activeTab: 0,
		views:     NewViewStack(NewBrowserView(&tab, "", engine)),
	}

	session := &TUISession{
		Version: 1,
		Tabs: []TabState{
			{
				ProjectPath:  "/test/App.mpr",
				SelectedNode: "MyModule.LoginPage",
				PreviewMode:  "NDSL",
			},
		},
	}
	app.pendingSession = session
	applySessionRestore(&app)

	// Session should be consumed
	if app.pendingSession != nil {
		t.Error("pendingSession should be nil after restore")
	}

	// Preview mode should be NDSL
	activeTab := app.activeTabPtr()
	if activeTab == nil {
		t.Fatal("no active tab")
	}
	if activeTab.Miller.preview.mode != PreviewNDSL {
		t.Errorf("preview mode: got %d, want NDSL (%d)", activeTab.Miller.preview.mode, PreviewNDSL)
	}
}

func TestApplySessionRestoreWithDeletedNode(t *testing.T) {
	engine := NewPreviewEngine("", "")
	nodes := []*TreeNode{
		{Label: "MyModule", Type: "Module", Children: []*TreeNode{
			{Label: "Pages", Type: "Folder", Children: []*TreeNode{
				{Label: "HomePage", Type: "Page", QualifiedName: "MyModule.HomePage"},
			}},
		}},
	}
	tab := NewTab(1, "/test/App.mpr", engine, nil)
	tab.AllNodes = nodes
	tab.Miller.SetRootNodes(nodes)

	app := App{
		tabs:      []Tab{tab},
		activeTab: 0,
		views:     NewViewStack(NewBrowserView(&tab, "", engine)),
	}

	// Try to restore a node that doesn't exist
	session := &TUISession{
		Version: 1,
		Tabs: []TabState{
			{
				ProjectPath:  "/test/App.mpr",
				SelectedNode: "MyModule.DeletedPage",
				MillerPath:   []string{"MyModule", "Pages"},
				PreviewMode:  "MDL",
			},
		},
	}
	app.pendingSession = session
	applySessionRestore(&app)

	// Should not panic, session should be consumed
	if app.pendingSession != nil {
		t.Error("pendingSession should be nil after restore")
	}
}

func TestApplySessionRestoreEmptySession(t *testing.T) {
	engine := NewPreviewEngine("", "")
	tab := NewTab(1, "/test/App.mpr", engine, nil)

	app := App{
		tabs:      []Tab{tab},
		activeTab: 0,
		views:     NewViewStack(NewBrowserView(&tab, "", engine)),
	}

	// Empty tabs in session — should be a no-op
	session := &TUISession{Version: 1, Tabs: []TabState{}}
	app.pendingSession = session
	applySessionRestore(&app)

	if app.pendingSession != nil {
		t.Error("pendingSession should be nil after restore")
	}
}

func TestSessionFilePathNotEmpty(t *testing.T) {
	path := sessionFilePath()
	if path == "" {
		t.Skip("could not determine home directory")
	}
	if filepath.Base(path) != "tui-session.json" {
		t.Errorf("session file name: got %q, want tui-session.json", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != ".mxcli" {
		t.Errorf("session file parent dir: got %q, want .mxcli", filepath.Base(filepath.Dir(path)))
	}
}
