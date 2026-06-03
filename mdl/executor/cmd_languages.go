// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// listLanguages lists all languages found in the project's translatable strings.
// Requires REFRESH CATALOG FULL to populate the strings table.
func listLanguages(ctx *ExecContext) error {
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

// --- Executor method wrapper for backward compatibility ---
