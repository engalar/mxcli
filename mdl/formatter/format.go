// SPDX-License-Identifier: Apache-2.0

// Package formatter provides heuristic MDL code formatting.
package formatter

import (
	"strings"
)

// mdlKeywords are MDL/SQL keywords that should be uppercased.
var mdlKeywords = []string{
	"CREATE", "OR", "MODIFY", "REPLACE", "DROP", "ALTER", "SHOW", "DESCRIBE",
	"ENTITY", "ASSOCIATION", "ENUMERATION", "CONSTANT", "MODULE", "MICROFLOW",
	"NANOFLOW", "PAGE", "SNIPPET", "LAYOUT", "WORKFLOW", "INDEX",
	"ATTRIBUTE", "FROM", "TO", "TYPE", "DEFAULT", "OWNER",
	"PERSISTENT", "NON_PERSISTENT", "SYSTEM_MEMBER", "STORED_VALUE", "CALCULATED",
	"REQUIRED", "UNIQUE", "INDEXED",
	"END", "IF", "THEN", "ELSE", "LOOP", "IN", "RETURN",
	"RETRIEVE", "WHERE", "LIMIT", "FIRST", "LIST", "OF",
	"CHANGE", "DELETE", "COMMIT", "ROLLBACK", "DOWNLOAD", "BROWSER",
	"SHOW_PAGE", "CLOSE_PAGE", "SHOW_MESSAGE",
	"PARAMETER", "PARAMETERS", "VARIABLE", "DECLARE",
	"JAVA_ACTION", "CALL", "CALL_MICROFLOW", "CALL_NANOFLOW",
	"LOG_MESSAGE", "LEVEL", "MESSAGE", "NODE",
	"ONE_TO_MANY", "MANY_TO_MANY", "ONE_TO_ONE", "REFERENCE", "REFERENCE_SET",
	"EXPOSED", "CLIENT", "COMMENT", "FOLDER",
	"ON_DELETE", "PREVENT", "CASCADE",
	"GRANT", "REVOKE", "ACCESS", "ALLOW", "DENY",
	"NOT", "AND", "NULL", "EMPTY", "TRUE", "FALSE",
	"SET", "INSERT", "BEFORE", "AFTER", "REPLACE",
	"BOOLEAN", "INTEGER", "LONG", "DECIMAL", "STRING", "DATETIME", "BINARY",
	"AUTO_NUMBER", "HASHED_STRING",
	"WIDGET", "COLUMN", "ROW", "CONTAINER", "DATAVIEW", "LISTVIEW",
	"BUTTON", "TEXT", "LABEL", "TITLE", "INPUT", "DROPDOWN", "CHECKBOX",
	"ENUMERATION_SELECTOR", "REFERENCE_SELECTOR", "DATE_PICKER",
	"DATA_SOURCE", "DIRECT", "XPATH",
	"NAVIGATION", "HOME", "MENU", "ITEM",
	"ROLE", "ROLES", "USER", "SECURITY", "PASSWORD",
	"REFRESH", "CATALOG",
	"SELECT", "AS", "TABLE", "TABLES",
	"SQL", "CONNECT", "QUERY", "IMPORT", "INTO", "MAP",
	"MOVE", "IMAGE", "COLLECTION",
}

// keywordSet for O(1) lookup.
var keywordSet map[string]string

func init() {
	keywordSet = make(map[string]string, len(mdlKeywords))
	for _, kw := range mdlKeywords {
		keywordSet[strings.ToUpper(kw)] = kw
	}
}

// Format applies heuristic formatting to MDL source code:
// - Uppercase MDL keywords
// - Normalize indentation to 2 spaces
// - Remove trailing whitespace
// - Normalize blank lines (max 1 consecutive)
func Format(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")

		// Collapse multiple blank lines
		if trimmed == "" {
			if !prevBlank {
				result = append(result, "")
			}
			prevBlank = true
			continue
		}
		prevBlank = false

		// Normalize indentation: count leading spaces, normalize to 2-space units
		stripped := strings.TrimLeft(trimmed, " \t")
		// Count effective indent (tabs = 2 spaces)
		indent := 0
		for _, ch := range trimmed {
			switch ch {
			case ' ':
				indent++
			case '\t':
				indent += 2
			default:
				goto indentDone
			}
		}
	indentDone:
		// Round to nearest 2-space unit
		indentLevel := indent / 2
		normalizedIndent := strings.Repeat("  ", indentLevel)

		// Uppercase keywords (but not inside quoted strings)
		formatted := uppercaseKeywords(stripped)
		result = append(result, normalizedIndent+formatted)
	}

	// Remove trailing blank line
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n") + "\n"
}

// uppercaseKeywords uppercases MDL keywords while preserving quoted strings.
func uppercaseKeywords(line string) string {
	// Track whether we're inside a string literal
	var result strings.Builder
	inString := false
	i := 0
	runes := []rune(line)

	for i < len(runes) {
		ch := runes[i]

		if ch == '\'' {
			// Toggle string mode, handle escaped quotes ('')
			result.WriteRune(ch)
			i++
			inString = !inString
			continue
		}

		if inString {
			result.WriteRune(ch)
			i++
			continue
		}

		// Check for line comment (--)
		if ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			// Rest of line is comment, write as-is
			for i < len(runes) {
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Check for block comment start
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			// Write rest as-is (simplification: assume comment ends on same line or later)
			for i < len(runes) {
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Try to match a word
		if isWordStart(ch) {
			wordStart := i
			for i < len(runes) && isWordChar(runes[i]) {
				i++
			}
			word := string(runes[wordStart:i])
			if _, ok := keywordSet[strings.ToUpper(word)]; ok {
				result.WriteString(strings.ToUpper(word))
			} else {
				result.WriteString(word)
			}
			continue
		}

		result.WriteRune(ch)
		i++
	}

	return result.String()
}

func isWordStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isWordChar(ch rune) bool {
	return isWordStart(ch) || (ch >= '0' && ch <= '9')
}
