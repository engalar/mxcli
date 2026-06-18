package graphcatalog

// ThemeVariableNode 对应 label="ThemeVariable" 的节点。
type ThemeVariableNode struct {
	Name         string
	Value        string
	VariableType string // "sass" | "css-custom-property"
	IsDefault    bool
	IsActive     bool
	Source       string // "project-custom-variables" | "project-main" | "module:xxx" | "atlas-core"
	Module       string
	Category     string
	FilePath     string
	LineNumber   int
}

// WidgetInstanceNode 对应 label="WidgetInstance" 的节点。
type WidgetInstanceNode struct {
	ID               string
	Name             string
	WidgetType       string
	Class            string
	Style            string
	DesignProperties map[string]string
	Page             string
}

// DesignPropertyNode 对应 label="DesignProperty" 的节点。
type DesignPropertyNode struct {
	WidgetType     string
	Name           string
	Type           string
	Category       string
	Description    string
	Options        []string
	ReferencedVars []string
	SourceModule   string
}

// ThemeVarFilter 是 ThemeVariables 查询的过滤条件。
type ThemeVarFilter struct {
	ActiveOnly bool
	Like       string
	Source     string
}
