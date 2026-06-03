// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// listLanguages lists the project's languages. It prefers the registered
// languages from project settings (Settings$LanguageSettings); when settings
// are unavailable or define no languages, it falls back to the per-string
// language counts in the catalog (requires REFRESH CATALOG FULL).
func listLanguages(ctx *ExecContext) error {
	if handled, err := listLanguagesFromSettings(ctx); err != nil {
		return err
	} else if handled {
		return nil
	}

	if ctx.Catalog == nil {
		return mdlerrors.NewValidation("no catalog available — run refresh catalog full first")
	}

	result, err := ctx.Catalog.Query(`
		select Language, count(*) as StringCount
		from strings
		where Language != ''
		GROUP by Language
		ORDER by StringCount desc
	`)
	if err != nil {
		return mdlerrors.NewBackend("query languages", err)
	}

	if len(result.Rows) == 0 && ctx.Format != FormatJSON {
		fmt.Fprintln(ctx.Output, "No translatable strings found. Run refresh catalog full to populate the strings table.")
		return nil
	}

	tr := &TableResult{
		Columns: []string{"Language", "Strings"},
		Summary: fmt.Sprintf("(%d languages)", len(result.Rows)),
	}
	for _, row := range result.Rows {
		lang := ""
		count := ""
		if len(row) > 0 {
			lang = fmt.Sprintf("%v", row[0])
		}
		if len(row) > 1 {
			count = fmt.Sprintf("%v", row[1])
		}
		tr.Rows = append(tr.Rows, []any{lang, count})
	}
	return writeResult(ctx, tr)
}

// listLanguagesFromSettings lists the languages registered in project settings.
// It returns handled=true when it produced output, and handled=false (with nil
// error) when settings are unavailable or define no languages, signalling the
// caller to fall back to the catalog.
func listLanguagesFromSettings(ctx *ExecContext) (bool, error) {
	if !ctx.Connected() {
		return false, nil
	}
	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		// Settings unavailable — let the caller fall back to the catalog.
		return false, nil
	}
	if ps == nil || ps.Language == nil || len(ps.Language.Languages) == 0 {
		return false, nil
	}

	defaultCode := ps.Language.DefaultLanguageCode
	tr := &TableResult{
		Columns: []string{"Code", "Language", "Default", "CheckCompleteness"},
		Summary: fmt.Sprintf("(%d languages)", len(ps.Language.Languages)),
	}
	for _, l := range ps.Language.Languages {
		name := supportedLanguages[l.Code]
		if name == "" {
			name = "(unknown)"
		}
		def := ""
		if l.Code == defaultCode {
			def = "yes"
		}
		cc := ""
		if l.CheckCompleteness {
			cc = "yes"
		}
		tr.Rows = append(tr.Rows, []any{l.Code, name, def, cc})
	}
	return true, writeResult(ctx, tr)
}

// supportedLanguages is the built-in list of valid Mendix language codes.
var supportedLanguages = map[string]string{
	"ar_SA": "Arabic",
	"bg_BG": "Bulgarian",
	"ca_ES": "Catalan",
	"cs_CZ": "Czech",
	"da_DK": "Danish",
	"de_DE": "German",
	"el_GR": "Greek",
	"en_GB": "English (UK)",
	"en_US": "English (US)",
	"es_ES": "Spanish",
	"es_MX": "Spanish (Mexico)",
	"fi_FI": "Finnish",
	"fr_BE": "French (Belgium)",
	"fr_FR": "French",
	"hr_HR": "Croatian",
	"hu_HU": "Hungarian",
	"id_ID": "Indonesian",
	"it_IT": "Italian",
	"ja_JP": "Japanese",
	"ko_KR": "Korean",
	"nb_NO": "Norwegian",
	"nl_BE": "Dutch (Belgium)",
	"nl_NL": "Dutch",
	"pl_PL": "Polish",
	"pt_BR": "Portuguese (Brazil)",
	"pt_PT": "Portuguese (Portugal)",
	"ro_RO": "Romanian",
	"ru_RU": "Russian",
	"sk_SK": "Slovak",
	"sl_SI": "Slovenian",
	"sr_CS": "Serbian",
	"sv_SE": "Swedish",
	"th_TH": "Thai",
	"tr_TR": "Turkish",
	"uk_UA": "Ukrainian",
	"vi_VN": "Vietnamese",
	"zh_CN": "Chinese (Simplified)",
	"zh_TW": "Chinese (Traditional)",
}

// isValidLanguageCode returns true if code is a known Mendix language code.
func isValidLanguageCode(code string) bool {
	_, ok := supportedLanguages[code]
	return ok
}

// listSupportedLanguages outputs all valid Mendix language codes.
func listSupportedLanguages(ctx *ExecContext) error {
	tr := &TableResult{
		Columns: []string{"Code", "Language"},
		Summary: fmt.Sprintf("(%d supported languages)", len(supportedLanguages)),
	}
	codes := make([]string, 0, len(supportedLanguages))
	for code := range supportedLanguages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		tr.Rows = append(tr.Rows, []any{code, supportedLanguages[code]})
	}
	return writeResult(ctx, tr)
}

// alterLanguage handles ALTER SETTINGS LANGUAGE ADD/DROP.
func alterLanguage(ctx *ExecContext, stmt *ast.AlterLanguageStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if !isValidLanguageCode(stmt.Code) {
		return mdlerrors.NewValidation(fmt.Sprintf(
			"'%s' is not a valid Mendix language code. Run SHOW SUPPORTED LANGUAGES to see valid codes.",
			stmt.Code,
		))
	}

	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	if ps.Language == nil {
		ps.Language = &model.LanguageSettings{DefaultLanguageCode: "en_US"}
	}

	switch stmt.Op {
	case ast.AlterLanguageAdd:
		return alterLanguageAdd(ctx, ps, stmt)
	case ast.AlterLanguageDrop:
		return alterLanguageDrop(ctx, ps, stmt)
	}
	return nil
}

func alterLanguageAdd(ctx *ExecContext, ps *model.ProjectSettings, stmt *ast.AlterLanguageStmt) error {
	for _, l := range ps.Language.Languages {
		if l.Code == stmt.Code {
			fmt.Fprintf(ctx.Output, "LANGUAGE %s already registered\n", stmt.Code)
			return nil
		}
	}
	lang := model.Language{Code: stmt.Code}
	if stmt.CheckCompleteness != nil {
		lang.CheckCompleteness = *stmt.CheckCompleteness
	}
	if stmt.DateFormat != "" {
		lang.CustomDateFormat = stmt.DateFormat
	}
	if stmt.DateTimeFormat != "" {
		lang.CustomDateTimeFormat = stmt.DateTimeFormat
	}
	if stmt.TimeFormat != "" {
		lang.CustomTimeFormat = stmt.TimeFormat
	}
	ps.Language.Languages = append(ps.Language.Languages, lang)
	if err := ctx.Backend.UpdateProjectSettings(ps); err != nil {
		return mdlerrors.NewBackend("update project settings", err)
	}
	fmt.Fprintf(ctx.Output, "LANGUAGE %s added\n", stmt.Code)
	return nil
}

func alterLanguageDrop(ctx *ExecContext, ps *model.ProjectSettings, stmt *ast.AlterLanguageStmt) error {
	if ps.Language.DefaultLanguageCode == stmt.Code {
		return mdlerrors.NewValidation(fmt.Sprintf(
			"cannot drop the default language '%s'. Change DefaultLanguageCode first.",
			stmt.Code,
		))
	}
	original := len(ps.Language.Languages)
	filtered := ps.Language.Languages[:0]
	for _, l := range ps.Language.Languages {
		if l.Code != stmt.Code {
			filtered = append(filtered, l)
		}
	}
	ps.Language.Languages = filtered
	if len(ps.Language.Languages) == original {
		fmt.Fprintf(ctx.Output, "LANGUAGE %s not registered\n", stmt.Code)
		return nil
	}
	if err := ctx.Backend.UpdateProjectSettings(ps); err != nil {
		return mdlerrors.NewBackend("update project settings", err)
	}
	fmt.Fprintf(ctx.Output, "LANGUAGE %s dropped\n", stmt.Code)
	return nil
}

// --- Executor method wrapper for backward compatibility ---
