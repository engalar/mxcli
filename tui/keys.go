package tui

import "github.com/charmbracelet/bubbles/key"

// ListKeyMap defines key bindings for list navigation and filtering.
type ListKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Back    key.Binding
	Open    key.Binding
	First   key.Binding
	Last    key.Binding
	Filter  key.Binding
	ExitFlt key.Binding
}

// ShortHelp returns a minimal set of bindings for inline help.
func (k ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Back, k.Filter}
}

// FullHelp returns all bindings grouped for the full-help overlay.
func (k ListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.First, k.Last},
		{k.Open, k.Back},
		{k.Filter, k.ExitFlt},
	}
}

// OverlayKeyMap defines key bindings for the overlay (detail) view.
type OverlayKeyMap struct {
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Search     key.Binding
	NextMatch  key.Binding
	PrevMatch  key.Binding
	Copy       key.Binding
	ToggleFmt  key.Binding
	Close      key.Binding
}

// ShortHelp returns a minimal set of bindings for overlay inline help.
func (k OverlayKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.ScrollDown, k.Search, k.Copy, k.ToggleFmt, k.Close}
}

// FullHelp returns all overlay bindings grouped for the full-help view.
func (k OverlayKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ScrollUp, k.ScrollDown},
		{k.Search, k.NextMatch, k.PrevMatch},
		{k.Copy, k.ToggleFmt},
		{k.Close},
	}
}

// CompareKeyMap defines key bindings for the compare (side-by-side) view.
type CompareKeyMap struct {
	ModeNDSL    key.Binding
	ModeNDSLMDL key.Binding
	ModeMDL     key.Binding
	SyncScroll  key.Binding
	Search      key.Binding
	Copy        key.Binding
	Close       key.Binding
}

// ShortHelp returns a minimal set of compare bindings.
func (k CompareKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.ModeNDSL, k.ModeNDSLMDL, k.ModeMDL, k.SyncScroll, k.Close}
}

// FullHelp returns all compare bindings grouped for the full-help view.
func (k CompareKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ModeNDSL, k.ModeNDSLMDL, k.ModeMDL},
		{k.SyncScroll, k.Search, k.Copy},
		{k.Close},
	}
}

// TabKeyMap defines key bindings for tab management.
type TabKeyMap struct {
	Tab1      key.Binding
	Tab2      key.Binding
	Tab3      key.Binding
	Tab4      key.Binding
	Tab5      key.Binding
	Tab6      key.Binding
	Tab7      key.Binding
	Tab8      key.Binding
	Tab9      key.Binding
	NewTab    key.Binding
	NewCross  key.Binding
	CloseTab  key.Binding
	PrevTab   key.Binding
	NextTab   key.Binding
}

// ShortHelp returns a minimal set of tab bindings.
func (k TabKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NewTab, k.CloseTab, k.PrevTab, k.NextTab}
}

// FullHelp returns all tab bindings grouped for the full-help view.
func (k TabKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.NewTab, k.NewCross, k.CloseTab},
		{k.PrevTab, k.NextTab},
	}
}

// GlobalKeyMap defines key bindings available in all contexts.
type GlobalKeyMap struct {
	Zen     key.Binding
	Compare key.Binding
	Diagram key.Binding
	Copy    key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// ShortHelp returns a minimal set of global bindings.
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns all global bindings grouped for the full-help view.
func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Zen, k.Compare, k.Diagram},
		{k.Copy, k.Refresh},
		{k.Help, k.Quit},
	}
}

// DefaultListKeys returns the default list navigation key bindings.
func DefaultListKeys() ListKeyMap {
	return ListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Back: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "back"),
		),
		Open: key.NewBinding(
			key.WithKeys("l", "right", "enter"),
			key.WithHelp("l/→/⏎", "open"),
		),
		First: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "first"),
		),
		Last: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "last"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ExitFlt: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit filter"),
		),
	}
}

// DefaultOverlayKeys returns the default overlay key bindings.
func DefaultOverlayKeys() OverlayKeyMap {
	return OverlayKeyMap{
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "scroll down"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		ToggleFmt: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "mdl/ndsl"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc/q", "close"),
		),
	}
}

// DefaultCompareKeys returns the default compare view key bindings.
func DefaultCompareKeys() CompareKeyMap {
	return CompareKeyMap{
		ModeNDSL: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "ndsl|ndsl"),
		),
		ModeNDSLMDL: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "ndsl|mdl"),
		),
		ModeMDL: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "mdl|mdl"),
		),
		SyncScroll: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync scroll"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
	}
}

// DefaultTabKeys returns the default tab management key bindings.
func DefaultTabKeys() TabKeyMap {
	return TabKeyMap{
		Tab1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tab 1")),
		Tab2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "tab 2")),
		Tab3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "tab 3")),
		Tab4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "tab 4")),
		Tab5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "tab 5")),
		Tab6: key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "tab 6")),
		Tab7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "tab 7")),
		Tab8: key.NewBinding(key.WithKeys("8"), key.WithHelp("8", "tab 8")),
		Tab9: key.NewBinding(key.WithKeys("9"), key.WithHelp("9", "tab 9")),
		NewTab: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "new tab"),
		),
		NewCross: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "new cross-project"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "close tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
	}
}

// DefaultGlobalKeys returns the default global key bindings.
func DefaultGlobalKeys() GlobalKeyMap {
	return GlobalKeyMap{
		Zen: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "zen mode"),
		),
		Compare: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compare"),
		),
		Diagram: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "diagram"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
}
