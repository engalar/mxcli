// SPDX-License-Identifier: Apache-2.0

package ast

// AlterLanguageOp represents the operation for ALTER SETTINGS LANGUAGE ADD/DROP.
type AlterLanguageOp int

const (
	AlterLanguageAdd  AlterLanguageOp = iota
	AlterLanguageDrop
)

// AlterLanguageStmt represents ALTER SETTINGS LANGUAGE ADD 'code' [...] or DROP 'code'.
type AlterLanguageStmt struct {
	Op                AlterLanguageOp
	Code              string
	CheckCompleteness *bool
	DateFormat        string
	DateTimeFormat    string
	TimeFormat        string
}

func (s *AlterLanguageStmt) isStatement() {}
func (s *AlterLanguageStmt) String() string {
	if s.Op == AlterLanguageAdd {
		return "ALTER SETTINGS LANGUAGE ADD " + s.Code
	}
	return "ALTER SETTINGS LANGUAGE DROP " + s.Code
}

// TranslateSetOp is a single SET path = text operation inside TRANSLATE.
type TranslateSetOp struct {
	Path string
	Text string
}

// TranslateStmt represents TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW Mod.Name IN lang SET ...
type TranslateStmt struct {
	DocType string
	QName   QualifiedName
	Lang    string
	Ops     []TranslateSetOp
}

func (s *TranslateStmt) isStatement() {}
func (s *TranslateStmt) String() string {
	return "TRANSLATE " + s.DocType + " " + s.QName.String() + " IN " + s.Lang
}

// DescribeTranslationsStmt represents DESCRIBE TRANSLATIONS Mod.Name [IN lang].
type DescribeTranslationsStmt struct {
	QName QualifiedName
	Lang  string
}

func (s *DescribeTranslationsStmt) isStatement() {}
func (s *DescribeTranslationsStmt) String() string {
	return "DESCRIBE TRANSLATIONS " + s.QName.String()
}
