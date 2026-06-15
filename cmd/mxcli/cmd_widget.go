// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/modelsdk/widgets"
	"github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"
	"github.com/spf13/cobra"
)

var widgetCmd = &cobra.Command{
	Use:   "widget",
	Short: "Widget management commands",
}

var widgetExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract widget definition from an .mpk file",
	Long: `Extract a pluggable widget definition from a Mendix .mpk package file
and generate a skeleton .def.json for use with the pluggable widget engine.

The command parses the widget XML inside the .mpk to discover properties,
infers the appropriate operation for each property based on its type,
and writes the result to the project's .mxcli/widgets/ directory.

Examples:
  mxcli widget extract --mpk widgets/MyWidget.mpk
  mxcli widget extract --mpk widgets/MyWidget.mpk --output .mxcli/widgets/
  mxcli widget extract --mpk widgets/MyWidget.mpk --mdl-name MYWIDGET`,
	RunE: runWidgetExtract,
}

var widgetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered widget definitions",
	Long:  `List all widget definitions available in the pluggable widget engine registry.`,
	RunE:  runWidgetList,
}

var widgetInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Dump widget definitions for inspection or customization",
	Long: `Scan the project's widgets/ directory and write .def.json files to
.mxcli/widgets/ for each .mpk. Also writes markdown reference docs to
.claude/skills/widgets/.

Note: mxcli widget init is no longer required for CREATE PAGE to work.
Widget definitions are derived automatically at runtime from the project's
widgets/*.mpk files. Run this command only when you need to inspect or
hand-edit a widget's property mappings.

Existing .def.json files are skipped unless --force is given.
Built-in widget definitions (e.g. GALLERY, COMBOBOX) are always skipped;
they are not derived from project .mpk files.

Note: --force will overwrite any hand-edits you have made to .def.json files.

Requires --project (-p) to locate the project's widgets/ directory.`,
	Example: `  mxcli widget init -p /path/to/app.mpr
  mxcli widget init -p app.mpr --force`,
	RunE: runWidgetInit,
}

var widgetDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate widget skill documentation",
	Long:  `Generate per-widget markdown documentation in .claude/skills/widgets/ from .mpk definitions.`,
	RunE:  runWidgetDocs,
}

var widgetNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new pluggable widget project",
	Long: `Create a new Mendix pluggable widget project with a parameterized XML definition,
JSX stub, editorConfig, editorPreview, and placeholder icons.

Examples:
  mxcli widget new MySlider
  mxcli widget new MySlider --property "value:attribute:Decimal" --property "label:string" --property "onChange:action"
  mxcli widget new MySlider --id com.acme.widget.MySlider --offline
  mxcli widget new CrusherWidgets --package`,
	RunE: runWidgetNew,
}

var widgetAddWidgetCmd = &cobra.Command{
	Use:   "add-widget <name>",
	Short: "Add a widget to an existing package project",
	Long: `Add a new widget to a multi-widget package project.
Run inside the package directory (where package.xml lives).

Examples:
  mxcli widget add-widget CrusherSlider
  mxcli widget add-widget CrusherSlider --property "value:attribute:Decimal" --property "label:string"`,
	RunE: runWidgetAddWidget,
}

var widgetBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build widget project into an .mpk package",
	Long: `Build a Mendix pluggable widget project into an .mpk file.

Discovers all widgets in src/*.xml, validates their definitions, invokes esbuild
(via bun or npm) to compile each JSX bundle, copies assets, packages everything
into <PackageName>.mpk, and verifies the output.

Examples:
  mxcli widget build
  mxcli widget build --dir ./CrusherWidgets`,
	RunE: runWidgetBuild,
}

var widgetInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a built .mpk into a Mendix project's widgets/ folder",
	Long: `Copy a widget .mpk file into <project>/widgets/, creating the directory if needed.

Without --mpk, auto-detects a single *.mpk in the current directory (the output of 'mxcli widget build').

Examples:
  mxcli widget install -p /path/to/app.mpr
  mxcli widget install --mpk TicketStatusBadge.mpk -p /path/to/app.mpr`,
	RunE: runWidgetInstall,
}

func init() {
	widgetExtractCmd.Flags().String("mpk", "", "Path to .mpk widget package file")
	widgetExtractCmd.Flags().StringP("output", "o", "", "Output directory (default: .mxcli/widgets/)")
	widgetExtractCmd.Flags().String("mdl-name", "", "Override the MDL keyword name (default: derived from widget name)")
	widgetExtractCmd.MarkFlagRequired("mpk")

	widgetInitCmd.Flags().StringP("project", "p", "", "Path to .mpr project file")
	widgetInitCmd.MarkFlagRequired("project")
	widgetInitCmd.Flags().Bool("force", false, "Overwrite existing .def.json files")

	widgetDocsCmd.Flags().StringP("project", "p", "", "Path to .mpr project file")
	widgetDocsCmd.MarkFlagRequired("project")

	widgetNewCmd.Flags().String("id", "", "Widget ID (default: com.mendix.widget.custom.<Name>.<Name>)")
	widgetNewCmd.Flags().StringArray("property", nil, "Property spec: key:type or key:type:subtype (repeatable)")
	widgetNewCmd.Flags().Bool("offline", false, "Set offlineCapable=true in the widget XML")
	widgetNewCmd.Flags().Bool("package", false, "Create a multi-widget package project (empty src/)")
	widgetNewCmd.Flags().String("description", "", "Widget description (written into XML and README)")

	widgetAddWidgetCmd.Flags().String("id", "", "Widget ID (default: com.mendix.widget.custom.<Name>.<Name>)")
	widgetAddWidgetCmd.Flags().StringArray("property", nil, "Property spec: key:type or key:type:subtype (repeatable)")
	widgetAddWidgetCmd.Flags().Bool("offline", false, "Set offlineCapable=true")
	widgetAddWidgetCmd.Flags().String("description", "", "Widget description (written into XML)")

	widgetBuildCmd.Flags().String("dir", ".", "Widget project root directory")
	widgetBuildCmd.Flags().Bool("install", false, "Install the built MPK into the Mendix project's widgets/ folder")
	widgetBuildCmd.Flags().StringP("project", "p", "", "Path to Mendix project (.mpr file) — required with --install")

	widgetInstallCmd.Flags().String("mpk", "", "Path to .mpk file (default: auto-detect *.mpk in current directory)")
	widgetInstallCmd.Flags().StringP("project", "p", "", "Path to Mendix project (.mpr file)")
	widgetInstallCmd.MarkFlagRequired("project")

	widgetCmd.AddCommand(widgetExtractCmd)
	widgetCmd.AddCommand(widgetListCmd)
	widgetCmd.AddCommand(widgetInitCmd)
	widgetCmd.AddCommand(widgetDocsCmd)
	widgetCmd.AddCommand(widgetNewCmd)
	widgetCmd.AddCommand(widgetAddWidgetCmd)
	widgetCmd.AddCommand(widgetBuildCmd)
	widgetCmd.AddCommand(widgetInstallCmd)
	rootCmd.AddCommand(widgetCmd)
}

func runWidgetInstall(cmd *cobra.Command, args []string) error {
	mpkFlag, _ := cmd.Flags().GetString("mpk")
	projectPath, _ := cmd.Flags().GetString("project")

	mpkPath := mpkFlag
	if mpkPath == "" {
		var err error
		mpkPath, err = findMPKInCwd()
		if err != nil {
			return err
		}
	}
	return installMPK(mpkPath, projectPath)
}

func runWidgetExtract(cmd *cobra.Command, args []string) error {
	mpkPath, _ := cmd.Flags().GetString("mpk")
	outputDir, _ := cmd.Flags().GetString("output")
	mdlNameOverride, _ := cmd.Flags().GetString("mdl-name")

	// Parse all widgets from the .mpk (multi-widget packages contain several).
	defs, err := mpk.ParseAll(mpkPath)
	if err != nil {
		return fmt.Errorf("failed to parse .mpk: %w", err)
	}
	if len(defs) == 0 {
		return fmt.Errorf("no widget definitions found in %s", mpkPath)
	}

	// Determine output directory
	if outputDir == "" {
		outputDir = filepath.Join(".mxcli", "widgets")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Extracting %d widget definition(s) from %s\n\n", len(defs), filepath.Base(mpkPath))

	for _, mpkDef := range defs {
		// --mdl-name only applies when extracting a single widget.
		mdlName := mdlNameOverride
		if mdlName == "" || len(defs) > 1 {
			mdlName = deriveMDLName(mpkDef.ID)
		}

		defJSON := generateDefJSON(mpkDef, mdlName)

		filename := strings.ToLower(mdlName) + ".def.json"
		outPath := filepath.Join(outputDir, filename)

		data, err := json.MarshalIndent(defJSON, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal definition for %s: %w", mpkDef.ID, err)
		}
		data = append(data, '\n')

		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		// Also generate and save the BSON template JSON derived from MPK XML.
		// This is a "best-effort" template: it may cause CE0463 if Studio Pro
		// expects a slightly different structure. The engine uses this file as
		// a fallback when no Studio Pro-created instance is found in the project.
		tmpl := widgets.GenerateFromMPK(mpkDef)
		tmplFilename := strings.ToLower(mdlName) + ".json"
		tmplPath := filepath.Join(outputDir, tmplFilename)
		if tmplData, err := json.MarshalIndent(tmpl, "", "  "); err == nil {
			tmplData = append(tmplData, '\n')
			if err := os.WriteFile(tmplPath, tmplData, 0644); err == nil {
				fmt.Fprintf(out, "  template:   %s\n", tmplPath)
			}
		}

		fmt.Fprintf(out, "  properties: %d\n\n", len(mpkDef.Properties))
	}

	fmt.Fprintf(out, "Note: .def.json = property mappings; .json = best-effort BSON template.\n")
	fmt.Fprintf(out, "      If page creation fails with CE0463, the template structure doesn't\n")
	fmt.Fprintf(out, "      match Studio Pro's. Fix: drag the widget onto a page in Studio Pro,\n")
	fmt.Fprintf(out, "      save, then run 'mxcli bson dump' to extract the correct template.\n")
	return nil
}

// toPascalCase uppercases the first character of s, leaving the rest unchanged.
// e.g. "cssValue" → "CssValue", "throwValue" → "ThrowValue".
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// deriveMDLName derives an uppercase MDL keyword from a widget ID.
// e.g. "com.mendix.widget.web.combobox.Combobox" → "COMBOBOX"
// e.g. "com.company.widget.MyCustomWidget" → "MYCUSTOMWIDGET"
func deriveMDLName(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	name := parts[len(parts)-1]
	return strings.ToUpper(name)
}

// generateDefJSON creates a skeleton WidgetDefinition from an mpk.WidgetDefinition.
// Properties are handled explicitly from MDL via the engine's explicit property pass,
// so no propertyMappings or childSlots are generated here.
func generateDefJSON(mpkDef *mpk.WidgetDefinition, mdlName string) *executor.WidgetDefinition {
	widgetKind := "custom"
	if mpkDef.IsPluggable {
		widgetKind = "pluggable"
	}
	def := &executor.WidgetDefinition{
		WidgetID:        mpkDef.ID,
		MDLName:         mdlName,
		WidgetKind:      widgetKind,
		TemplateFile:    strings.ToLower(mdlName) + ".json",
		DefaultEditable: "Always",
	}

	// Generate property mappings and child slots from MPK property definitions.
	// Two passes: datasource first (association depends on entityContext set by datasource).
	var assocMappings []executor.PropertyMapping
	for _, p := range mpkDef.Properties {
		switch p.Type {
		case "widgets":
			container := strings.ToUpper(p.Key)
			if p.Key == "content" {
				container = "TEMPLATE"
			}
			def.ChildSlots = append(def.ChildSlots, executor.ChildSlotMapping{
				PropertyKey:  p.Key,
				MDLContainer: container,
				Operation:    "widgets",
			})
		case "datasource":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "DataSource",
				Operation:   "datasource",
				Required:    p.Required,
			})
		case "attribute":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      toPascalCase(p.Key),
				Operation:   "attribute",
				Required:    p.Required,
			})
		case "association":
			assocMappings = append(assocMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Association",
				Operation:   "association",
				Required:    p.Required,
			})
		case "selection":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Selection",
				Operation:   "selection",
				Default:     p.DefaultValue,
				Required:    p.Required,
			})
		case "action":
			// Action properties are handled by the engine's explicit-properties path
			// via propertyTypeIDs; no PropertyMapping entry is needed.
		case "boolean", "integer", "decimal", "string", "enumeration":
			m := executor.PropertyMapping{
				PropertyKey: p.Key,
				Operation:   "primitive",
				Required:    p.Required,
			}
			if p.DefaultValue != "" {
				m.Value = p.DefaultValue
			}
			def.PropertyMappings = append(def.PropertyMappings, m)
		}
	}
	// Append association mappings after datasource (association requires prior entityContext)
	def.PropertyMappings = append(def.PropertyMappings, assocMappings...)

	return def
}

func runWidgetInit(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	force, _ := cmd.Flags().GetBool("force")
	projectDir := filepath.Dir(projectPath)
	widgetsDir := filepath.Join(projectDir, "widgets")
	outputDir := filepath.Join(projectDir, ".mxcli", "widgets")
	out := cmd.OutOrStdout()

	// Load built-in registry to skip widgets that already have hand-crafted definitions
	builtinRegistry, _ := executor.NewWidgetRegistry()

	// Scan widgets/ for .mpk files
	matches, err := filepath.Glob(filepath.Join(widgetsDir, "*.mpk"))
	if err != nil {
		return fmt.Errorf("failed to scan widgets directory: %w", err)
	}
	if len(matches) == 0 {
		fmt.Fprintln(out, "No .mpk files found in widgets/ directory.")
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var extracted, skipped int
	for _, mpkPath := range matches {
		defs, err := mpk.ParseAll(mpkPath)
		if err != nil {
			log.Printf("warning: skipping %s: %v", filepath.Base(mpkPath), err)
			skipped++
			continue
		}

		for _, mpkDef := range defs {
			mdlName := deriveMDLName(mpkDef.ID)
			filename := strings.ToLower(mdlName) + ".def.json"
			outPath := filepath.Join(outputDir, filename)

			// Skip widgets that have hand-crafted built-in definitions (e.g., COMBOBOX, GALLERY)
			if builtinRegistry != nil {
				if _, ok := builtinRegistry.GetByWidgetID(mpkDef.ID); ok {
					skipped++
					continue
				}
			}

			// Skip if already exists on disk (unless --force)
			if !force {
				if _, err := os.Stat(outPath); err == nil {
					skipped++
					continue
				}
			}

			defJSON := generateDefJSON(mpkDef, mdlName)
			data, err := json.MarshalIndent(defJSON, "", "  ")
			if err != nil {
				log.Printf("warning: skipping %s: %v", mpkDef.ID, err)
				skipped++
				continue
			}
			data = append(data, '\n')

			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outPath, err)
			}
			kind := "custom"
			if mpkDef.IsPluggable {
				kind = "pluggable"
			}
			fmt.Fprintf(out, "  %-12s %-20s %s\n", kind, mdlName, mpkDef.ID)
			extracted++
		}
	}

	fmt.Fprintf(out, "\nExtracted: %d, Skipped: %d (existing or unparseable)\n", extracted, skipped)

	// Also generate docs
	fmt.Fprintln(out, "\nGenerating widget documentation...")
	return generateWidgetDocs(projectDir, out)
}

func runWidgetDocs(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	projectDir := filepath.Dir(projectPath)
	return generateWidgetDocs(projectDir, cmd.OutOrStdout())
}

func generateWidgetDocs(projectDir string, out io.Writer) error {
	widgetsDir := filepath.Join(projectDir, "widgets")
	docsDir := filepath.Join(projectDir, ".claude", "skills", "widgets")
	// Also try .ai-context
	if _, err := os.Stat(filepath.Join(projectDir, ".ai-context")); err == nil {
		docsDir = filepath.Join(projectDir, ".ai-context", "skills", "widgets")
	}

	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	matches, err := filepath.Glob(filepath.Join(widgetsDir, "*.mpk"))
	if err != nil {
		return fmt.Errorf("failed to scan widgets directory: %w", err)
	}

	var generated int
	var indexEntries []string

	for _, mpkPath := range matches {
		defs, err := mpk.ParseAll(mpkPath)
		if err != nil {
			continue
		}

		for _, mpkDef := range defs {
			mdlName := deriveMDLName(mpkDef.ID)
			filename := strings.ToLower(mdlName) + ".md"
			outPath := filepath.Join(docsDir, filename)

			doc := generateWidgetDoc(mpkDef, mdlName)

			if err := os.WriteFile(outPath, []byte(doc), 0644); err != nil {
				log.Printf("warning: failed to write %s: %v", filename, err)
				continue
			}

			kind := "CUSTOMWIDGET"
			if mpkDef.IsPluggable {
				kind = "PLUGGABLEWIDGET"
			}
			indexEntries = append(indexEntries, fmt.Sprintf("| `%s` | %s | `%s` | %s | %d |",
				kind, mdlName, mpkDef.ID, mpkDef.Name, len(mpkDef.Properties)))
			generated++
		}
	}

	// Write index
	var indexBuf strings.Builder
	indexBuf.WriteString("# Available Widgets\n\n")
	indexBuf.WriteString("Generated by `mxcli widget docs`. See individual files for property details.\n\n")
	indexBuf.WriteString("| Prefix | Name | Widget ID | Display Name | Props |\n")
	indexBuf.WriteString("|--------|------|-----------|--------------|-------|\n")
	for _, entry := range indexEntries {
		indexBuf.WriteString(entry)
		indexBuf.WriteString("\n")
	}
	indexBuf.WriteString("\n**Usage in MDL:**\n```sql\n")
	indexBuf.WriteString("-- React pluggable widgets\n")
	indexBuf.WriteString("PLUGGABLEWIDGET 'com.mendix.widget.custom.badge.Badge' badge1\n\n")
	indexBuf.WriteString("-- Legacy custom widgets\n")
	indexBuf.WriteString("CUSTOMWIDGET 'com.company.OldWidget' legacy1\n")
	indexBuf.WriteString("```\n")

	indexPath := filepath.Join(docsDir, "_index.md")
	if err := os.WriteFile(indexPath, []byte(indexBuf.String()), 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	fmt.Fprintf(out, "Generated %d widget docs in %s\n", generated, docsDir)
	return nil
}

func generateWidgetDoc(mpkDef *mpk.WidgetDefinition, mdlName string) string {
	var buf strings.Builder

	prefix := "CUSTOMWIDGET"
	if mpkDef.IsPluggable {
		prefix = "PLUGGABLEWIDGET"
	}

	buf.WriteString(fmt.Sprintf("# %s\n\n", mpkDef.Name))
	buf.WriteString(fmt.Sprintf("- **Widget ID:** `%s`\n", mpkDef.ID))
	buf.WriteString(fmt.Sprintf("- **Type:** %s\n", prefix))
	buf.WriteString(fmt.Sprintf("- **Version:** %s\n\n", mpkDef.Version))

	buf.WriteString("## MDL Example\n\n```sql\n")
	buf.WriteString(fmt.Sprintf("%s '%s' widget1\n", prefix, mpkDef.ID))
	buf.WriteString("```\n\n")

	if len(mpkDef.Properties) > 0 {
		buf.WriteString("## Properties\n\n")
		buf.WriteString("| Property | Type | Required | Default | Description |\n")
		buf.WriteString("|----------|------|----------|---------|-------------|\n")

		for _, prop := range mpkDef.Properties {
			if prop.IsSystem {
				continue
			}
			req := ""
			if prop.Required {
				req = "Yes"
			}
			desc := prop.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
				prop.Key, prop.Type, req, prop.DefaultValue, desc))
		}
	}

	buf.WriteString("\n")
	return buf.String()
}

func runWidgetList(cmd *cobra.Command, args []string) error {
	registry, err := executor.NewWidgetRegistry()
	if err != nil {
		return fmt.Errorf("failed to create widget registry: %w", err)
	}

	// Load user definitions and scan project MPKs if project path is available.
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath != "" {
		if err := registry.LoadUserDefinitions(projectPath); err != nil {
			log.Printf("warning: loading user widget definitions: %v", err)
		}
		projectDir := filepath.Dir(projectPath)
		if err := registry.SetProjectDir(projectDir); err != nil {
			log.Printf("warning: scanning project widgets/: %v", err)
		}
	}

	out := cmd.OutOrStdout()
	defs := registry.All()

	fmt.Fprintf(out, "%-16s %-20s %-50s %s\n", "Kind", "MDL Name", "Widget ID", "Template")
	fmt.Fprintf(out, "%-16s %-20s %-50s %s\n", strings.Repeat("-", 16), strings.Repeat("-", 20), strings.Repeat("-", 50), strings.Repeat("-", 20))
	for _, def := range defs {
		kind := def.WidgetKind
		if kind == "" {
			kind = "pluggable"
		}
		fmt.Fprintf(out, "%-16s %-20s %-50s %s\n", kind, def.MDLName, def.WidgetID, def.TemplateFile)
	}

	// Show widgets discovered in widgets/*.mpk but not yet extracted to .def.json.
	discovered := registry.MPKDiscovered()
	if len(discovered) > 0 {
		fmt.Fprintf(out, "\n--- Auto-discovered from widgets/*.mpk ---\n\n")
		fmt.Fprintf(out, "%-22s %-32s %-45s %s\n", "MDL Name (auto)", "Display Name", "Widget ID", "Description")
		fmt.Fprintf(out, "%-22s %-32s %-45s %s\n",
			strings.Repeat("-", 22), strings.Repeat("-", 32), strings.Repeat("-", 45), strings.Repeat("-", 30))
		names := make([]string, 0, len(discovered))
		for name := range discovered {
			names = append(names, name)
		}
		sort.Strings(names)
		var exampleID string
		for _, name := range names {
			w := discovered[name]
			if exampleID == "" {
				exampleID = w.WidgetID
			}
			desc := w.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Fprintf(out, "%-22s %-32s %-45s %s\n", strings.ToLower(name), w.Name, w.WidgetID, desc)
		}
		fmt.Fprintf(out, "\nMDL syntax for MPK widgets:  PLUGGABLEWIDGET '<Widget ID>' name (prop: val)\n")
		fmt.Fprintf(out, "Example:                     PLUGGABLEWIDGET '%s' myWidget1 (...)\n", exampleID)
		fmt.Fprintf(out, "\nMPK widgets are auto-discovered — no extraction needed.\n")
		fmt.Fprintf(out, "To override property mappings: mxcli widget extract --mpk widgets/<file>.mpk\n")
	}

	fmt.Fprintf(out, "\nTotal: %d loaded, %d from MPK\n", len(defs), len(discovered))

	return nil
}
