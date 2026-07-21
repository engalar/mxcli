package kernel

type ViewMode int

const (
	ModeBrowser ViewMode = iota
	ModeOverlay
	ModeCompare
	ModeDiff
	ModePicker
	ModeJumper
	ModeExec
	ModeCommandPalette
	ModeInput
	ModeConfirm
)

func (m ViewMode) String() string {
	switch m {
	case ModeBrowser:
		return "Browse"
	case ModeOverlay:
		return "Overlay"
	case ModeCompare:
		return "Compare"
	case ModeDiff:
		return "Diff"
	case ModePicker:
		return "Picker"
	case ModeJumper:
		return "Jump"
	case ModeExec:
		return "Exec"
	case ModeCommandPalette:
		return "Palette"
	case ModeInput:
		return "Input"
	case ModeConfirm:
		return "Confirm"
	default:
		return "Unknown"
	}
}

type StatusInfo struct {
	Breadcrumb []string
	Position   string
	Mode       string
	Extra      string
}

type Hint struct {
	Key   string
	Label string
}

type TreeNode struct {
	Label         string      `json:"label"`
	Type          string      `json:"type"`
	QualifiedName string      `json:"qualifiedName,omitempty"`
	Children      []*TreeNode `json:"children,omitempty"`
}
