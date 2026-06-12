// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

func describeTranslations(ctx *ExecContext, stmt *ast.DescribeTranslationsStmt) error {
	ps, err := ctx.SettingsReader.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	if ps == nil {
		ps = &model.ProjectSettings{}
	}

	var langs []string
	if stmt.Lang != "" {
		langs = []string{stmt.Lang}
	} else if ps.Language != nil {
		for _, l := range ps.Language.Languages {
			langs = append(langs, l.Code)
		}
	}
	sort.Strings(langs)

	docQN := stmt.QName.String()
	nodes, err := ctx.SettingsWriter.ListTranslationNodes(docQN, "")
	if err != nil {
		return mdlerrors.NewBackend("list translation nodes", err)
	}

	srcLang := "en_US"
	if ps.Language != nil && len(ps.Language.Languages) > 0 {
		srcLang = ps.Language.Languages[0].Code
	}

	if ctx.Format == FormatJSON {
		return describeTranslationsJSON(ctx, docQN, stmt.Lang, srcLang, nodes, ps)
	}
	return describeTranslationsText(ctx, docQN, stmt.Lang, langs, nodes)
}

func describeTranslationsText(ctx *ExecContext, docQN, targetLang string, langs []string, nodes []model.TranslationNode) error {
	columns := []string{"Path", "Property"}
	columns = append(columns, langs...)

	var missingPaths []string
	tr := &TableResult{Columns: columns}
	for _, node := range nodes {
		row := []any{node.Path, node.Property}
		for _, lang := range langs {
			if text, ok := node.Texts[lang]; ok {
				row = append(row, text)
			} else {
				row = append(row, "(missing)")
				if lang == targetLang || targetLang == "" {
					missingPaths = append(missingPaths, node.Path)
				}
			}
		}
		tr.Rows = append(tr.Rows, row)
	}
	missingCount := len(missingPaths)
	tr.Summary = fmt.Sprintf("(%d translatable fields, %d missing in %s)", len(nodes), missingCount, targetLang)

	if err := writeResult(ctx, tr); err != nil {
		return err
	}

	if missingCount > 0 && targetLang != "" {
		docType := strings.ToLower(guessDocType(nodes))
		fmt.Fprintf(ctx.Output, "\n-- Ready-to-execute template (replace '?' with translations):\n")
		fmt.Fprintf(ctx.Output, "-- translate %s %s in %s\n", docType, docQN, targetLang)
		for _, path := range missingPaths {
			fmt.Fprintf(ctx.Output, "--   set %s = '?'\n", path)
		}
		fmt.Fprintln(ctx.Output, "--   ;")
	}
	return nil
}

func describeTranslationsJSON(ctx *ExecContext, docQN, targetLang, srcLang string, nodes []model.TranslationNode, ps *model.ProjectSettings) error {
	type missingEntry struct {
		Path     string `json:"path"`
		Property string `json:"property"`
		SrcLang  string `json:"source_lang"`
		Source   string `json:"source"`
	}
	type translatedEntry struct {
		Path     string `json:"path"`
		Property string `json:"property"`
		SrcLang  string `json:"source_lang"`
		Source   string `json:"source"`
		Text     string `json:"text"`
	}

	missing := []missingEntry{}
	translated := []translatedEntry{}

	for _, node := range nodes {
		src := node.Texts[srcLang]
		if targetLang == "" {
			continue
		}
		if text, ok := node.Texts[targetLang]; ok {
			translated = append(translated, translatedEntry{
				Path: node.Path, Property: node.Property,
				SrcLang: srcLang, Source: src, Text: text,
			})
		} else {
			missing = append(missing, missingEntry{
				Path: node.Path, Property: node.Property,
				SrcLang: srcLang, Source: src,
			})
		}
	}

	docType := "page"
	if len(nodes) > 0 {
		docType = strings.ToLower(guessDocType(nodes))
	}
	templateLines := []string{fmt.Sprintf("translate %s %s in %s", docType, docQN, targetLang)}
	for _, m := range missing {
		templateLines = append(templateLines, fmt.Sprintf("  set %s = '?'", m.Path))
	}
	tmpl := strings.Join(templateLines, "\n") + ";"

	var allLangs []string
	if ps.Language != nil {
		for _, l := range ps.Language.Languages {
			allLangs = append(allLangs, l.Code)
		}
	}

	out := map[string]any{
		"document":           docQN,
		"document_type":      strings.ToUpper(docType),
		"target_language":    targetLang,
		"project_languages":  allLangs,
		"missing":            missing,
		"translated":         translated,
		"translate_template": tmpl,
	}
	enc := json.NewEncoder(ctx.Output)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func guessDocType(nodes []model.TranslationNode) string {
	if len(nodes) > 0 && nodes[0].DocType != "" {
		return nodes[0].DocType
	}
	return "PAGE"
}
