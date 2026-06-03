// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	// ── Languages ───────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "language",
		Summary: "Manage project languages — list registered/supported languages, add or drop a language",
		Keywords: []string{
			"language", "languages", "locale", "i18n", "internationalization",
			"add language", "drop language", "supported languages",
		},
		Syntax: `SHOW LANGUAGES;
SHOW SUPPORTED LANGUAGES;
ALTER SETTINGS LANGUAGE ADD '<code>';
ALTER SETTINGS LANGUAGE ADD '<code>' ( checkCompleteness: <bool>, dateFormat: '<fmt>', dateTimeFormat: '<fmt>', timeFormat: '<fmt>' );
ALTER SETTINGS LANGUAGE DROP '<code>';
ALTER SETTINGS LANGUAGE DefaultLanguageCode = '<code>';`,
		Example: `-- See which languages are registered in the project, and which codes are valid
SHOW LANGUAGES;
SHOW SUPPORTED LANGUAGES;

-- Register a new language (idempotent; rejects unknown codes)
ALTER SETTINGS LANGUAGE ADD 'nl_NL';
ALTER SETTINGS LANGUAGE ADD 'fr_FR' ( checkCompleteness: true, dateFormat: 'dd-MM-yyyy' );

-- Remove a language (cannot drop the default language)
ALTER SETTINGS LANGUAGE DROP 'de_DE';`,
		SeeAlso: []string{"translate", "translate.describe", "settings.alter"},
	})

	// ── Translation ─────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "translate",
		Summary: "Set per-language translations for page, snippet, and enumeration text",
		Keywords: []string{
			"translate", "translation", "i18n", "caption", "title",
			"multilingual", "language text",
		},
		Syntax: `TRANSLATE PAGE <Module.Name> IN <code> SET <path> = '<text>', ...;
TRANSLATE SNIPPET <Module.Name> IN <code> SET <path> = '<text>', ...;
TRANSLATE ENUMERATION <Module.Name> IN <code> SET <ValueName>.caption = '<text>', ...;

-- Page/snippet paths:  title  |  <WidgetName>.<property>   (caption, placeholder, tooltip, label, content)
-- Enumeration paths:   <ValueName>.caption`,
		Example: `-- The target language must be registered first (ALTER SETTINGS LANGUAGE ADD).
TRANSLATE PAGE MyModule.Home IN nl_NL SET
  title = 'Welkom',
  SubmitButton.caption = 'Verzenden';

TRANSLATE SNIPPET MyModule.Header IN nl_NL SET
  LogoutButton.caption = 'Afmelden';

TRANSLATE ENUMERATION MyModule.Status IN nl_NL SET
  ACTIVE.caption = 'Actief',
  CLOSED.caption = 'Gesloten';`,
		SeeAlso: []string{"language", "translate.describe"},
	})

	Register(SyntaxFeature{
		Path:    "translate.describe",
		Summary: "Show the translatable text of a document with its per-language translations",
		Keywords: []string{
			"describe translations", "show translations", "translation status",
			"missing translations", "translate template",
		},
		Syntax: `DESCRIBE TRANSLATIONS <Module.Name>;
DESCRIBE TRANSLATIONS <Module.Name> IN <code>;`,
		Example: `-- All translatable nodes with every language's text
DESCRIBE TRANSLATIONS MyModule.Home;

-- Focus on one language (blanks reveal what still needs translating)
DESCRIBE TRANSLATIONS MyModule.Home IN nl_NL;`,
		SeeAlso: []string{"translate", "language"},
	})
}
