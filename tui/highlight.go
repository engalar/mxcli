package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var ndslLexer = chroma.MustNewLexer(
	&chroma.Config{
		Name:      "NDSL",
		Aliases:   []string{"ndsl"},
		MimeTypes: []string{"text/x-ndsl"},
	},
	func() chroma.Rules {
		return chroma.Rules{
			"root": {
				{Pattern: `\$Type\b`, Type: chroma.KeywordType},
				{Pattern: `<uuid>`, Type: chroma.CommentPreproc},
				{Pattern: `int(?:32|64)\(\d+\)`, Type: chroma.LiteralNumberInteger},
				{Pattern: `[A-Z][A-Za-z]*(?:\[\d+\])?(?:\.[A-Z][A-Za-z]*(?:\[\d+\])?)*`, Type: chroma.NameAttribute},
				{Pattern: `"[^"]*"`, Type: chroma.LiteralString},
				{Pattern: `'[^']*'`, Type: chroma.LiteralString},
				{Pattern: `\b\d+(?:\.\d+)?\b`, Type: chroma.LiteralNumber},
				{Pattern: `\b(?:true|false|null)\b`, Type: chroma.KeywordConstant},
				{Pattern: `\s+`, Type: chroma.TextWhitespace},
				{Pattern: `[=:,\[\]{}()]`, Type: chroma.Punctuation},
				{Pattern: `.`, Type: chroma.Text},
			},
		}
	},
)

func highlight(content string, lexer chroma.Lexer) string {
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return content
	}
	style := styles.Get("monokai")
	if style == nil {
		return content
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}
	return buf.String()
}

// HighlightMDL highlights MDL content using the SQL lexer as a base.
func HighlightMDL(content string) string {
	lexer := lexers.Get("sql")
	if lexer == nil {
		return content
	}
	return highlight(content, lexer)
}

// HighlightSQL highlights standard SQL content.
func HighlightSQL(content string) string {
	lexer := lexers.Get("sql")
	if lexer == nil {
		return content
	}
	return highlight(content, lexer)
}

// HighlightNDSL highlights NDSL format content with custom lexer rules.
func HighlightNDSL(content string) string {
	return highlight(content, ndslLexer)
}

// StripBanner removes leading banner lines (WARNING:, Connected to:, blank lines)
// from mxcli output so only the actual content remains.
func StripBanner(content string) string {
	lines := strings.Split(content, "\n")
	start := 0
	for start < len(lines) {
		trimmed := strings.TrimSpace(lines[start])
		if trimmed == "" ||
			strings.HasPrefix(trimmed, "WARNING:") ||
			strings.HasPrefix(trimmed, "Connected to:") {
			start++
			continue
		}
		break
	}
	if start == 0 {
		return content
	}
	return strings.Join(lines[start:], "\n")
}

// DetectAndHighlight strips mxcli banner, auto-detects content type, and applies highlighting.
func DetectAndHighlight(content string) string {
	content = StripBanner(content)
	if content == "" {
		return content
	}

	// NDSL detection: $Type or field path patterns
	if strings.Contains(content, "$Type") {
		return HighlightNDSL(content)
	}

	// Scan first non-blank lines for type detection
	for _, line := range strings.SplitN(content, "\n", 20) {
		trimmed := strings.ToUpper(strings.TrimSpace(line))
		if trimmed == "" {
			continue
		}

		// MDL keywords
		for _, kw := range []string{"CREATE ", "ALTER ", "SHOW ", "DESCRIBE ", "DROP ",
			"GRANT ", "REVOKE ", "WORKFLOW ", "MICROFLOW ", "ENTITY ", "PAGE ", "NANOFLOW "} {
			if strings.HasPrefix(trimmed, kw) {
				return HighlightMDL(content)
			}
		}

		// SQL keywords
		for _, kw := range []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "WITH "} {
			if strings.HasPrefix(trimmed, kw) {
				return HighlightSQL(content)
			}
		}

		// Comment lines (DESCRIBE output)
		if strings.HasPrefix(trimmed, "-- ") {
			return HighlightMDL(content)
		}
	}

	return content
}
