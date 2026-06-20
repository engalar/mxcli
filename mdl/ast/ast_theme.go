package ast

// ShowThemeVariablesStmt 表示: SHOW THEME VARIABLES [LIKE pattern] [DEFAULT] [IN MODULE name]
type ShowThemeVariablesStmt struct {
	LikePattern  string
	ShowDefaults bool
	InModule     string
}

func (s *ShowThemeVariablesStmt) isStatement()     {}
func (s *ShowThemeVariablesStmt) TypeName() string { return "ShowThemeVariables" }
