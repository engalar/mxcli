package tui

import "github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"

type Hint = kernel.Hint

var (
	BrowserHints = []kernel.Hint{
		{Key: "SPC", Label: "menu"},
		{Key: "h", Label: "back"},
		{Key: "l", Label: "open"},
		{Key: "/", Label: "filter"},
		{Key: ":", Label: "commands"},
		{Key: "?", Label: "help"},
	}
	FilterActiveHints = []kernel.Hint{
		{Key: "Enter", Label: "confirm"},
		{Key: "Esc", Label: "cancel"},
	}
	OverlayHints = []kernel.Hint{
		{Key: "j/k", Label: "scroll"},
		{Key: "/", Label: "search"},
		{Key: "y", Label: "copy"},
		{Key: "e", Label: "edit mdl"},
		{Key: "Tab", Label: "mdl/ndsl"},
		{Key: "q", Label: "close"},
	}
	CompareHints = []kernel.Hint{
		{Key: "h/l", Label: "navigate"},
		{Key: "/", Label: "search"},
		{Key: "s", Label: "sync scroll"},
		{Key: "1/2/3", Label: "mode"},
		{Key: "D", Label: "diff"},
		{Key: "q", Label: "close"},
	}
	ExecViewHints = []kernel.Hint{
		{Key: "Ctrl+E", Label: "execute"},
		{Key: "Ctrl+F", Label: "format"},
		{Key: "Ctrl+O", Label: "open file"},
		{Key: "Ctrl+P", Label: "preview"},
		{Key: "Esc", Label: "close"},
	}
	DiffViewHints = []kernel.Hint{
		{Key: "j/k", Label: "scroll"},
		{Key: "Tab", Label: "mode"},
		{Key: "]c/[c", Label: "hunk"},
		{Key: "/", Label: "search"},
		{Key: "y", Label: "yank"},
		{Key: "q", Label: "close"},
	}
)
