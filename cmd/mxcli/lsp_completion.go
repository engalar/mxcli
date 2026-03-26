// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Completion handles textDocument/completion requests.
func (s *mdlServer) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	// Get current line text to provide context-aware completions
	docURI := uri.URI(params.TextDocument.URI)
	s.mu.Lock()
	text := s.docs[docURI]
	s.mu.Unlock()

	linePrefix := ""
	if text != "" {
		lines := strings.Split(text, "\n")
		line := int(params.Position.Line)
		if line < len(lines) {
			col := min(int(params.Position.Character), len(lines[line]))
			linePrefix = strings.TrimLeft(lines[line][:col], " \t")
		}
	}
	linePrefixUpper := strings.ToUpper(linePrefix)

	// Check if context calls for catalog-based element completion
	if types := inferCompletionTypes(linePrefixUpper); types != nil {
		items := s.catalogCompletionItems(ctx, linePrefix, types)
		return &protocol.CompletionList{
			IsIncomplete: false,
			Items:        items,
		}, nil
	}

	items := s.mdlCompletionItems(linePrefixUpper)
	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// CompletionResolve handles completionItem/resolve requests.
func (s *mdlServer) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return params, nil
}

// mdlCompletionItems returns completion items filtered by context.
func (s *mdlServer) mdlCompletionItems(linePrefixUpper string) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// After CREATE, suggest object types and CREATE snippets only
	if strings.HasPrefix(linePrefixUpper, "CREATE ") || linePrefixUpper == "CREATE" {
		items = append(items, mdlCreateSnippets...)
		items = append(items, mdlCreateContextKeywords...)
		return items
	}

	// After SHOW, suggest showable objects
	if strings.HasPrefix(linePrefixUpper, "SHOW ") || linePrefixUpper == "SHOW" {
		items = append(items, mdlShowContextKeywords...)
		return items
	}

	// General context: all generated keywords + snippets + widget types
	items = append(items, mdlGeneratedKeywords...)
	items = append(items, mdlStatementSnippets...)
	items = append(items, mdlCreateSnippets...)
	items = append(items, s.widgetRegistryCompletions()...)

	return items
}

// widgetRegistryCompletions returns completion items for registered widget types,
// including user-defined widgets from global (~/.mxcli/widgets/) and project-level
// (.mxcli/widgets/) directories.
// NOTE: Cached via sync.Once — new .def.json files added while the LSP server is
// running will not appear until the server is restarted.
func (s *mdlServer) widgetRegistryCompletions() []protocol.CompletionItem {
	s.widgetCompletionsOnce.Do(func() {
		registry, err := executor.NewWidgetRegistry()
		if err != nil {
			return
		}
		if err := registry.LoadUserDefinitions(s.mprPath); err != nil {
			// Non-fatal: user definitions are optional
			_ = err
		}
		for _, def := range registry.All() {
			s.widgetCompletionItems = append(s.widgetCompletionItems, protocol.CompletionItem{
				Label:  def.MDLName,
				Kind:   protocol.CompletionItemKindClass,
				Detail: "Pluggable widget: " + def.WidgetID,
			})
		}
	})
	return s.widgetCompletionItems
}

// mdlCreateContextKeywords are object types suggested after CREATE.
// These are hand-written because they require semantic knowledge of what can be created.
var mdlCreateContextKeywords = []protocol.CompletionItem{
	kw("ENTITY", "Entity type"),
	kw("PERSISTENT", "Persistent entity modifier"),
	kw("NON-PERSISTENT", "Non-persistent entity modifier"),
	kw("VIEW", "View entity modifier"),
	kw("EXTERNAL", "External entity modifier"),
	kw("MICROFLOW", "Microflow type"),
	kw("NANOFLOW", "Nanoflow type"),
	kw("PAGE", "Page type"),
	kw("SNIPPET", "Snippet type"),
	kw("LAYOUT", "Layout type"),
	kw("ENUMERATION", "Enumeration type"),
	kw("ASSOCIATION", "Association type"),
	kw("CONSTANT", "Constant type"),
	kw("MODULE", "Module type"),
	kw("JAVA ACTION", "Java action type"),
}

// mdlShowContextKeywords are items suggested after SHOW.
var mdlShowContextKeywords = []protocol.CompletionItem{
	kw("ENTITIES", "List entities"),
	kw("MICROFLOWS", "List microflows"),
	kw("NANOFLOWS", "List nanoflows"),
	kw("PAGES", "List pages"),
	kw("SNIPPETS", "List snippets"),
	kw("LAYOUTS", "List layouts"),
	kw("ENUMERATIONS", "List enumerations"),
	kw("MODULES", "List modules"),
	kw("ASSOCIATIONS", "List associations"),
	kw("CONSTANTS", "List constants"),
	kw("WIDGETS", "List widgets"),
	kw("CALLERS", "Show callers of element"),
	kw("CALLEES", "Show callees of element"),
	kw("REFERENCES", "Show references to element"),
	kw("IMPACT", "Show impact analysis"),
	kw("CONTEXT", "Show context of element"),
	kw("CATALOG", "Show catalog info"),
	kw("DATABASE CONNECTIONS", "List database connections"),
}

func kw(label string, detail string) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:  label,
		Kind:   protocol.CompletionItemKindKeyword,
		Detail: detail,
	}
}

func snippet(label, insertText, detail string) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:            label,
		Kind:             protocol.CompletionItemKindSnippet,
		Detail:           detail,
		InsertText:       insertText,
		InsertTextFormat: protocol.InsertTextFormatSnippet,
	}
}

var mdlCreateSnippets = []protocol.CompletionItem{
	snippet("CREATE ENTITY", "CREATE ENTITY ${1:Module}.${2:EntityName}\n(\n\t${3:AttributeName} : ${4:String}\n);", "Create a new entity"),
	snippet("CREATE PERSISTENT ENTITY", "CREATE PERSISTENT ENTITY ${1:Module}.${2:EntityName}\n(\n\t${3:AttributeName} : ${4:String}\n);", "Create a persistent entity"),
	snippet("CREATE NON_PERSISTENT ENTITY", "CREATE NON_PERSISTENT ENTITY ${1:Module}.${2:EntityName}\n(\n\t${3:AttributeName} : ${4:String}\n);", "Create a non-persistent entity"),
	snippet("CREATE MICROFLOW", "CREATE MICROFLOW ${1:Module}.${2:MicroflowName}\nBEGIN\n\t$0\nEND;", "Create a new microflow"),
	snippet("CREATE MICROFLOW (with params)", "CREATE MICROFLOW ${1:Module}.${2:MicroflowName}\n(\n\t$$${3:Param}: ${4:Module.Entity}\n)\nRETURNS ${5:Boolean} AS $$${6:Result}\nBEGIN\n\t$0\nEND;", "Create microflow with parameters"),
	snippet("CREATE NANOFLOW", "CREATE NANOFLOW ${1:Module}.${2:NanoflowName}\nBEGIN\n\t$0\nEND;", "Create a new nanoflow"),
	snippet("CREATE ENUMERATION", "CREATE ENUMERATION ${1:Module}.${2:EnumName}\n(\n\t'${3:Value1}' '${4:Caption1}',\n\t'${5:Value2}' '${6:Caption2}'\n);", "Create a new enumeration"),
	snippet("CREATE CONSTANT", "CREATE CONSTANT ${1:Module}.${2:ConstantName}\nTYPE ${3|String,Integer,Long,Decimal,Boolean,DateTime|}\nDEFAULT ${4:'value'};", "Create a new constant"),
	snippet("CREATE PAGE", "CREATE PAGE ${1:Module}.${2:PageName}\n(\n\tTitle: '${3:Page Title}',\n\tLayout: ${4:Atlas_Core.Atlas_Default}\n)\n{\n\t$0\n}", "Create a new page"),
	snippet("CREATE SNIPPET", "CREATE SNIPPET ${1:Module}.${2:SnippetName}\n{\n\t$0\n}", "Create a new snippet"),
	snippet("CREATE ASSOCIATION", "CREATE ASSOCIATION ${1:Module}.${2:AssocName}\nFROM ${1:Module}.${3:ChildEntity}\nTO ${1:Module}.${4:ParentEntity}\nTYPE ${5|Reference,ReferenceSet|};", "Create a new association"),
	snippet("CREATE MODULE", "CREATE MODULE ${1:ModuleName};", "Create a new module"),
}

var mdlStatementSnippets = []protocol.CompletionItem{
	snippet("IF ... END IF", "IF ${1:condition} THEN\n\t$0\nEND IF;", "If-then block"),
	snippet("IF ... ELSE ... END IF", "IF ${1:condition} THEN\n\t${2}\nELSE\n\t$0\nEND IF;", "If-then-else block"),
	snippet("LOOP ... END LOOP", "LOOP $$${1:Item} IN $$${2:List}\nBEGIN\n\t$0\nEND LOOP;", "Loop over a list"),
	snippet("WHILE ... END WHILE", "WHILE ${1:condition}\nBEGIN\n\t$0\nEND WHILE;", "While loop with condition"),
	snippet("DECLARE variable", "DECLARE $$${1:Var} ${2:String} = ${3:''};", "Declare a variable"),
	snippet("RETRIEVE ... FROM", "RETRIEVE $$${1:Var} FROM ${2:Module.Entity} WHERE ${3:condition};", "Retrieve from database"),
	snippet("RETRIEVE ... FROM $Var/Assoc", "RETRIEVE $$${1:List} FROM $$${2:Parent}/${3:Module.AssociationName};", "Retrieve by association"),
	snippet("DATAVIEW", "DATAVIEW ${1:dvName} (DataSource: $$${2:Var}) {\n\t$0\n}", "Data view widget"),
	snippet("INDEX", "INDEX (${1:AttributeName});", "Entity index"),
}

// inferCompletionTypes examines the line prefix and returns the ObjectType
// values to filter catalog elements on, or nil if no catalog completion applies.
func inferCompletionTypes(linePrefixUpper string) []string {
	// Patterns ordered from most specific to least specific.
	// Each entry: prefix to match → element types to suggest.
	patterns := []struct {
		prefix string
		types  []string
	}{
		// Microflow/nanoflow calls
		{"CALL MICROFLOW ", []string{"MICROFLOW"}},
		{"CALL NANOFLOW ", []string{"NANOFLOW"}},
		{"CALL JAVA ACTION ", []string{"JAVA_ACTION"}},

		// Page actions
		{"SHOW PAGE ", []string{"PAGE"}},

		// Retrieve
		{"RETRIEVE ", []string{"ENTITY"}}, // matches "RETRIEVE $x FROM Module." too

		// Widget datasource patterns
		{"DATASOURCE: DATABASE ", []string{"ENTITY"}},
		{"DATASOURCE: MICROFLOW ", []string{"MICROFLOW"}},
		{"DATASOURCE: NANOFLOW ", []string{"NANOFLOW"}},

		// Widget action patterns
		{"ACTION: SHOW_PAGE ", []string{"PAGE"}},
		{"ACTION: MICROFLOW ", []string{"MICROFLOW"}},
		{"ACTION: NANOFLOW ", []string{"NANOFLOW"}},
		{"ACTION: CREATE_OBJECT ", []string{"ENTITY"}},

		// Page properties
		{"LAYOUT: ", []string{"LAYOUT"}},
		{"SNIPPET: ", []string{"SNIPPET"}},

		// Snippet call widget
		{"SNIPPETCALL ", []string{"SNIPPET"}},
	}

	for _, p := range patterns {
		if strings.HasPrefix(linePrefixUpper, p.prefix) {
			return p.types
		}
		// Also check if the pattern appears after other content on the line
		// e.g. "  DataSource: DATABASE " preceded by whitespace
		if idx := strings.LastIndex(linePrefixUpper, p.prefix); idx >= 0 {
			return p.types
		}
	}

	// RETRIEVE ... FROM pattern: "RETRIEVE $x FROM "
	if strings.Contains(linePrefixUpper, " FROM ") && strings.Contains(linePrefixUpper, "RETRIEVE ") {
		return []string{"ENTITY"}
	}

	// EXTENDS in entity context
	if strings.Contains(linePrefixUpper, "EXTENDS ") {
		return []string{"ENTITY"}
	}

	// Association FROM/TO patterns (but not RETRIEVE FROM which is handled above)
	if (strings.HasPrefix(linePrefixUpper, "FROM ") || strings.HasPrefix(linePrefixUpper, "TO ")) &&
		!strings.Contains(linePrefixUpper, "RETRIEVE") {
		return []string{"ENTITY"}
	}

	return nil
}

// catalogCompletionItems returns completion items from the catalog filtered by types.
func (s *mdlServer) catalogCompletionItems(ctx context.Context, linePrefix string, types []string) []protocol.CompletionItem {
	elems := s.getProjectElements(ctx)
	if len(elems) == 0 {
		return nil
	}

	// Build a set of allowed types
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	// Extract the partial text the user has typed after the trigger context.
	// e.g., "CALL MICROFLOW MyMod" → partial = "MyMod"
	partial := ""
	parts := strings.Fields(linePrefix)
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		// If the line ends with a space, no partial filter
		if !strings.HasSuffix(linePrefix, " ") {
			partial = last
		}
	}
	partialUpper := strings.ToUpper(partial)

	// Collect unique module names for module-level completion
	moduleSet := make(map[string]bool)

	var items []protocol.CompletionItem
	for _, elem := range elems {
		if !typeSet[elem.ObjectType] {
			continue
		}

		// Extract module name
		dotIdx := strings.Index(elem.QualifiedName, ".")
		if dotIdx > 0 {
			moduleSet[elem.QualifiedName[:dotIdx]] = true
		}

		// Filter by partial text
		if partialUpper != "" {
			nameUpper := strings.ToUpper(elem.QualifiedName)
			if !strings.Contains(nameUpper, partialUpper) {
				continue
			}
		}

		kind, detail := objectTypeToCompletionKind(elem.ObjectType)
		items = append(items, protocol.CompletionItem{
			Label:  elem.QualifiedName,
			Kind:   kind,
			Detail: detail,
		})
	}

	// If the partial has no dot, also suggest module names
	if !strings.Contains(partial, ".") {
		for mod := range moduleSet {
			modUpper := strings.ToUpper(mod)
			if partialUpper == "" || strings.Contains(modUpper, partialUpper) {
				items = append(items, protocol.CompletionItem{
					Label:  mod,
					Kind:   protocol.CompletionItemKindModule,
					Detail: "module",
				})
			}
		}
	}

	return items
}

// objectTypeToCompletionKind maps catalog ObjectType to LSP CompletionItemKind and detail text.
func objectTypeToCompletionKind(objectType string) (protocol.CompletionItemKind, string) {
	switch objectType {
	case "ENTITY":
		return protocol.CompletionItemKindClass, "entity"
	case "MICROFLOW":
		return protocol.CompletionItemKindMethod, "microflow"
	case "NANOFLOW":
		return protocol.CompletionItemKindMethod, "nanoflow"
	case "PAGE":
		return protocol.CompletionItemKindFile, "page"
	case "SNIPPET":
		return protocol.CompletionItemKindFile, "snippet"
	case "LAYOUT":
		return protocol.CompletionItemKindFile, "layout"
	case "ENUMERATION":
		return protocol.CompletionItemKindEnum, "enumeration"
	case "JAVA_ACTION":
		return protocol.CompletionItemKindMethod, "java action"
	case "WORKFLOW":
		return protocol.CompletionItemKindEvent, "workflow"
	case "MODULE":
		return protocol.CompletionItemKindModule, "module"
	default:
		return protocol.CompletionItemKindValue, objectType
	}
}
