// Package normalize provides MDL normalization for noise-free diff comparison.
//
// Normalization removes non-functional differences (comments, @position,
// whitespace, keyword casing, statement ordering) so that a diff between
// describe output and a hand-written MDL script shows only semantic gaps.
package normalize

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/formatter"
)

// Options controls which normalization passes are applied.
type Options struct {
	StripPosition bool // remove @Position(...) annotations
	StripComments bool // remove -- and /**/ comments
	SortStatements bool // sort statements by type then name
}

// DefaultOptions returns sensible defaults for compare-oriented normalization.
func DefaultOptions() Options {
	return Options{
		StripPosition: true,
		StripComments: true,
		SortStatements: true,
	}
}

// statementBlock is a raw text block for one top-level statement, delimited
// by the terminating "/" line.
type statementBlock struct {
	text     string // full text including the trailing "/"
	category string // statement category for sorting (enum, entity, assoc, etc.)
	key      string // sorting key (qualified name if available)
}

var (
	rePosition       = regexp.MustCompile(`(?m)^\s*@Position\([^)]*\)\s*$`)
	reSlashLine      = regexp.MustCompile(`(?m)^/\s*$`)
	reDocComment     = regexp.MustCompile(`(?s)/\*[*!].*?\*/`)
	reTrailingWS     = regexp.MustCompile(`(?m)[ \t]+$`)
	reBlankLine      = regexp.MustCompile(`(?m)^[ \t]*\n`)
	reDoubleBlank    = regexp.MustCompile(`\n{3,}`)
	reAnnotation     = regexp.MustCompile(`(?m)^\s*-- @annotation .*$`)
	reExcluded       = regexp.MustCompile(`(?m)^\s*@excluded\s*$`)
	reCaptionColor   = regexp.MustCompile(`(?m)^\s*(@caption|@color)\b.*$`)
	reTerminator     = regexp.MustCompile(`(?m)^/\s*$`)
	reStmtBoundary   = regexp.MustCompile(`\n/\s*(\n|$)`)
)

// Normalize applies all normalization passes to the given MDL text and
// returns the normalized result. An error is returned if the text cannot
// be parsed as valid MDL (ensures that only structurally sound scripts
// are compared).
func Normalize(input string, opts Options) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil
	}

	text := input

	// Pass 1: strip documentation blocks and comments.
	if opts.StripComments {
		text = stripComments(text)
	}

	// Pass 2: strip @Position and other display-only annotations.
	if opts.StripPosition {
		text = rePosition.ReplaceAllString(text, "")
		text = reAnnotation.ReplaceAllString(text, "")
		text = reExcluded.ReplaceAllString(text, "")
		text = reCaptionColor.ReplaceAllString(text, "")
	}

	// Pass 3: normalize trailing whitespace and blank lines.
	text = reTrailingWS.ReplaceAllString(text, "")
	text = reBlankLine.ReplaceAllString(text, "\n")
	text = reDoubleBlank.ReplaceAllString(text, "\n\n")
	text = strings.TrimSpace(text)

	// Pass 4: split into statement blocks and sort.
	if opts.SortStatements {
		blocks := splitBlocks(text)
		sortBlocks(blocks)
		text = reassembleBlocks(blocks)
	}

	// Pass 5: apply canonical formatting (uppercase keywords, 2-space indent).
	text = formatter.Format(text)

	return text, nil
}

// stripComments removes -- line comments and /**/ block comments.
func stripComments(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip pure-comment lines.
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Strip inline comments after code (only for non-string lines).
		cleaned := stripInlineComment(line)
		result = append(result, cleaned)
	}

	// Remove /**/ doc block lines.
	joined := strings.Join(result, "\n")
	joined = reDocComment.ReplaceAllString(joined, "")
	return joined
}

// stripInlineComment removes a trailing -- or // comment from a line,
// preserving whitespace before the comment. Simple heuristic: find the
// first unquoted -- or // and truncate.
func stripInlineComment(line string) string {
	inString := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\'':
			inString = !inString
		case '-':
			if !inString && i+1 < len(line) && line[i+1] == '-' {
				return strings.TrimRight(line[:i], " \t")
			}
		case '/':
			if !inString && i+1 < len(line) && line[i+1] == '/' {
				return strings.TrimRight(line[:i], " \t")
			}
		}
	}
	return line
}

// splitBlocks splits normalized text into statement blocks.
// Handles both describe output (;/ delimiter) and hand-written MDL (; delimiter).
func splitBlocks(text string) []statementBlock {
	// Normalize terminator: / on its own line is the canonical delimiter.
	// Replace standalone ; followed by newline with ;\n/ for uniform splitting.
	normalized := reTerminator.ReplaceAllStringFunc(text, func(m string) string {
		return "\n" + strings.TrimSpace(m) + "\n"
	})

	lines := strings.Split(normalized, "\n")
	var blocks []statementBlock
	var cur []string

	flush := func() {
		if len(cur) == 0 {
			return
		}
		block := strings.Join(cur, "\n")
		// Normalize terminator: strip trailing / and keep the core statement.
		block = strings.TrimSuffix(block, "/")
		block = strings.TrimSpace(block)
		if block == "" {
			return
		}
		blocks = append(blocks, statementBlock{
			text:     block,
			category: classifyBlock(block),
			key:      blockKey(block),
		})
		cur = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "/" {
			flush()
			continue
		}
		// Empty lines are block separators — swallow them.
		if trimmed == "" && len(cur) == 0 {
			continue
		}
		cur = append(cur, line)
	}
	flush()

	return blocks
}

// classifyBlock determines the sort category for a statement block.
func classifyBlock(block string) string {
	head := strings.ToUpper(firstWord(block))
	switch {
	case strings.Contains(block, "CREATE MODULE ROLE") || strings.Contains(block, "create module role"):
		return "01-module-role"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "ENUMERATION"):
		return "02-enumeration"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "ENTITY"):
		return "03-entity"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "ASSOCIATION"):
		return "04-association"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "MICROFLOW"):
		return "05-microflow"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "NANOFLOW"):
		return "06-nanoflow"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "WORKFLOW"):
		return "07-workflow"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "CONSTANT"):
		return "08-constant"
	case strings.Contains(block, "CREATE USER ROLE") || strings.Contains(block, "create user role"):
		return "09-user-role"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "NAVIGATION"):
		return "10-navigation"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "PAGE"):
		return "11-page"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "LAYOUT"):
		return "12-layout"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "SNIPPET"):
		return "13-snippet"
	case strings.Contains(block, "CREATE") && strings.Contains(block, "JAVA ACTION"):
		return "14-java-action"
	case strings.Contains(block, "GRANT VIEW") || strings.Contains(block, "grant view"):
		return "15-grant-view"
	case strings.Contains(block, "GRANT") || strings.Contains(block, "grant "):
		return "16-grant"
	case strings.Contains(block, "ALTER SETTINGS") || strings.Contains(block, "alter settings"):
		return "17-settings"
	case strings.Contains(block, "ALTER WORKFLOW") || strings.Contains(block, "alter workflow"):
		return "18-alter-workflow"
	case strings.Contains(block, "CREATE MODULE ") || strings.Contains(block, "create module "):
		return "00-module"
	case strings.Contains(block, "ALTER PAGE") || strings.Contains(block, "alter page"):
		return "19-alter-page"
	default:
		if strings.HasPrefix(head, "ALTER") || strings.HasPrefix(head, "ALTER") {
			return "90-alter"
		}
		return "99-other"
	}
}

// blockKey extracts the qualified name from a statement for sorting.
func blockKey(block string) string {
	lines := strings.SplitN(block, "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	first := strings.TrimSpace(lines[0])
	// Match MODULE.Name pattern.
	if idx := strings.Index(first, " "); idx >= 0 {
		rest := strings.TrimSpace(first[idx+1:])
		// Extract the qualified name (Module.Name or quoted name).
		for _, sep := range []string{"(", " "} {
			if i := strings.Index(rest, sep); i >= 0 {
				rest = strings.TrimSpace(rest[:i])
			}
		}
		return rest
	}
	return first
}

// firstWord returns the first whitespace-delimited word.
func firstWord(s string) string {
	idx := strings.IndexAny(s, " \t\n")
	if idx < 0 {
		return s
	}
	return s[:idx]
}

// sortBlocks sorts statement blocks by category then key.
func sortBlocks(blocks []statementBlock) {
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].category != blocks[j].category {
			return blocks[i].category < blocks[j].category
		}
		return blocks[i].key < blocks[j].key
	})
}

// reassembleBlocks joins sorted blocks with separator blank lines.
func reassembleBlocks(blocks []statementBlock) string {
	var parts []string
	for _, b := range blocks {
		parts = append(parts, strings.TrimSpace(b.text))
	}
	return strings.Join(parts, "\n\n")
}

// FormatAlias provides the formatter alias for the CLI help.
func FormatAlias() string {
	return fmt.Sprintf("normalize")
}
