// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitAlterSettingsClause handles ALTER SETTINGS ... clauses.
func (b *Builder) ExitAlterSettingsClause(ctx *parser.AlterSettingsClauseContext) {
	// ALTER SETTINGS LANGUAGE ADD/DROP 'code' is a distinct statement.
	if ctx.LANGUAGE() != nil && (ctx.ADD() != nil || ctx.DROP() != nil) {
		b.buildAlterLanguage(ctx)
		return
	}

	stmt := &ast.AlterSettingsStmt{
		Properties: make(map[string]any),
	}

	if ctx.DROP() != nil && ctx.CONSTANT() != nil {
		// ALTER SETTINGS DROP CONSTANT 'name' [IN CONFIGURATION 'cfg']
		stmt.Section = "constant"
		stmt.DropConstant = true
		allStrings := ctx.AllSTRING_LITERAL()
		if len(allStrings) > 0 {
			stmt.ConstantId = unquoteString(allStrings[0].GetText())
		}
		if ctx.IN() != nil && ctx.CONFIGURATION() != nil && len(allStrings) > 1 {
			stmt.ConfigName = unquoteString(allStrings[1].GetText())
		}
	} else if ctx.CONSTANT() != nil {
		// ALTER SETTINGS CONSTANT 'name' (VALUE 'value' | DROP) [IN CONFIGURATION 'cfg']
		stmt.Section = "constant"
		allStrings := ctx.AllSTRING_LITERAL()
		if len(allStrings) > 0 {
			stmt.ConstantId = unquoteString(allStrings[0].GetText())
		}
		if ctx.DROP() != nil {
			stmt.DropConstant = true
		} else if ctx.SettingsValue() != nil {
			stmt.Value = settingsValueText(ctx.SettingsValue().(*parser.SettingsValueContext))
		}
		// Check for IN CONFIGURATION 'name'
		if ctx.IN() != nil && ctx.CONFIGURATION() != nil && len(allStrings) > 1 {
			stmt.ConfigName = unquoteString(allStrings[1].GetText())
		}
	} else if ctx.CONFIGURATION() != nil {
		// ALTER SETTINGS CONFIGURATION 'name' Key = Value, ...
		stmt.Section = "configuration"
		allStrings := ctx.AllSTRING_LITERAL()
		if len(allStrings) > 0 {
			stmt.ConfigName = unquoteString(allStrings[0].GetText())
		}
		for _, assignCtx := range ctx.AllSettingsAssignment() {
			assign, ok := assignCtx.(*parser.SettingsAssignmentContext)
			if !ok || assign == nil {
				continue
			}
			if assign.IDENTIFIER() == nil || assign.SettingsValue() == nil {
				continue
			}
			key := assign.IDENTIFIER().GetText()
			svCtx, ok := assign.SettingsValue().(*parser.SettingsValueContext)
			if !ok || svCtx == nil {
				continue
			}
			val := settingsValueText(svCtx)
			stmt.Properties[key] = val
		}
	} else if ctx.SettingsSection() != nil {
		// ALTER SETTINGS MODEL|LANGUAGE|WORKFLOWS Key = Value, ...
		stmt.Section = ctx.SettingsSection().GetText()
		for _, assignCtx := range ctx.AllSettingsAssignment() {
			assign, ok := assignCtx.(*parser.SettingsAssignmentContext)
			if !ok || assign == nil {
				continue
			}
			if assign.IDENTIFIER() == nil || assign.SettingsValue() == nil {
				continue
			}
			key := assign.IDENTIFIER().GetText()
			svCtx, ok := assign.SettingsValue().(*parser.SettingsValueContext)
			if !ok || svCtx == nil {
				continue
			}
			val := settingsValueToInterface(svCtx)
			stmt.Properties[key] = val
		}
	}

	b.statements = append(b.statements, stmt)
}

// buildAlterLanguage builds an AlterLanguageStmt from an ALTER SETTINGS LANGUAGE
// ADD/DROP clause and appends it to the statement list.
func (b *Builder) buildAlterLanguage(ctx *parser.AlterSettingsClauseContext) {
	stmt := &ast.AlterLanguageStmt{}
	if strs := ctx.AllSTRING_LITERAL(); len(strs) > 0 {
		stmt.Code = unquoteString(strs[0].GetText())
	}
	if ctx.ADD() != nil {
		stmt.Op = ast.AlterLanguageAdd
	} else {
		stmt.Op = ast.AlterLanguageDrop
	}

	if opts := ctx.LanguageOptions(); opts != nil {
		if oc, ok := opts.(*parser.LanguageOptionsContext); ok {
			for _, optCtx := range oc.AllLanguageOption() {
				opt, ok := optCtx.(*parser.LanguageOptionContext)
				if !ok || opt == nil || opt.IdentifierOrKeyword() == nil || opt.SettingsValue() == nil {
					continue
				}
				key := strings.ToLower(identifierOrKeywordText(opt.IdentifierOrKeyword()))
				svCtx, ok := opt.SettingsValue().(*parser.SettingsValueContext)
				if !ok || svCtx == nil {
					continue
				}
				val := settingsValueText(svCtx)
				switch key {
				case "checkcompleteness":
					b := strings.EqualFold(val, "true")
					stmt.CheckCompleteness = &b
				case "dateformat":
					stmt.DateFormat = val
				case "datetimeformat":
					stmt.DateTimeFormat = val
				case "timeformat":
					stmt.TimeFormat = val
				}
			}
		}
	}

	b.statements = append(b.statements, stmt)
}

// ExitCreateConfigurationStatement handles CREATE CONFIGURATION 'name' [Key = Value, ...].
func (b *Builder) ExitCreateConfigurationStatement(ctx *parser.CreateConfigurationStatementContext) {
	stmt := &ast.CreateConfigurationStmt{
		Properties: make(map[string]any),
	}

	if sl := ctx.STRING_LITERAL(); sl != nil {
		stmt.Name = unquoteString(sl.GetText())
	}

	for _, assignCtx := range ctx.AllSettingsAssignment() {
		assign, ok := assignCtx.(*parser.SettingsAssignmentContext)
		if !ok || assign == nil {
			continue
		}
		if assign.IDENTIFIER() == nil || assign.SettingsValue() == nil {
			continue
		}
		key := assign.IDENTIFIER().GetText()
		svCtx, ok := assign.SettingsValue().(*parser.SettingsValueContext)
		if !ok || svCtx == nil {
			continue
		}
		stmt.Properties[key] = settingsValueText(svCtx)
	}

	b.statements = append(b.statements, stmt)
}

// settingsValueText extracts the string value from a SettingsValue context.
func settingsValueText(ctx *parser.SettingsValueContext) string {
	if sl := ctx.STRING_LITERAL(); sl != nil {
		return unquoteString(sl.GetText())
	}
	if nl := ctx.NUMBER_LITERAL(); nl != nil {
		return nl.GetText()
	}
	if bl := ctx.BooleanLiteral(); bl != nil {
		return bl.GetText()
	}
	if qn := ctx.QualifiedName(); qn != nil {
		return getQualifiedNameText(qn)
	}
	return ctx.GetText()
}

// settingsValueToInterface extracts a typed value from a SettingsValue context.
func settingsValueToInterface(ctx *parser.SettingsValueContext) any {
	if sl := ctx.STRING_LITERAL(); sl != nil {
		return unquoteString(sl.GetText())
	}
	if nl := ctx.NUMBER_LITERAL(); nl != nil {
		if v, err := strconv.ParseInt(nl.GetText(), 10, 64); err == nil {
			return v
		}
		return nl.GetText()
	}
	if bl := ctx.BooleanLiteral(); bl != nil {
		text := bl.GetText()
		return text == "true" || text == "TRUE" || text == "True"
	}
	if qn := ctx.QualifiedName(); qn != nil {
		return getQualifiedNameText(qn)
	}
	return ctx.GetText()
}
