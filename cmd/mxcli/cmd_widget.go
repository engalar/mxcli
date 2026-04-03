// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
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
	Short: "Extract definitions for all project widgets",
	Long: `Scan the project's widgets/ directory, extract .def.json for each .mpk,
and generate skill documentation in .claude/skills/widgets/.

This enables CREATE PAGE to use any project widget via the pluggable engine.`,
	RunE: runWidgetInit,
}

var widgetDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate widget skill documentation",
	Long:  `Generate per-widget markdown documentation in .claude/skills/widgets/ from .mpk definitions.`,
	RunE:  runWidgetDocs,
}

func init() {
	widgetExtractCmd.Flags().String("mpk", "", "Path to .mpk widget package file")
	widgetExtractCmd.Flags().StringP("output", "o", "", "Output directory (default: .mxcli/widgets/)")
	widgetExtractCmd.Flags().String("mdl-name", "", "Override the MDL keyword name (default: derived from widget name)")
	widgetExtractCmd.MarkFlagRequired("mpk")

	widgetInitCmd.Flags().StringP("project", "p", "", "Path to .mpr project file")
	widgetInitCmd.MarkFlagRequired("project")

	widgetDocsCmd.Flags().StringP("project", "p", "", "Path to .mpr project file")
	widgetDocsCmd.MarkFlagRequired("project")

	widgetCmd.AddCommand(widgetExtractCmd)
	widgetCmd.AddCommand(widgetListCmd)
	widgetCmd.AddCommand(widgetInitCmd)
	widgetCmd.AddCommand(widgetDocsCmd)
	rootCmd.AddCommand(widgetCmd)
}

func runWidgetExtract(cmd *cobra.Command, args []string) error {
	mpkPath, _ := cmd.Flags().GetString("mpk")
	outputDir, _ := cmd.Flags().GetString("output")
	mdlNameOverride, _ := cmd.Flags().GetString("mdl-name")

	// Parse .mpk
	mpkDef, err := mpk.ParseMPK(mpkPath)
	if err != nil {
		return fmt.Errorf("failed to parse .mpk: %w", err)
	}

	// Determine output directory
	if outputDir == "" {
		outputDir = filepath.Join(".mxcli", "widgets")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine MDL name
	mdlName := mdlNameOverride
	if mdlName == "" {
		mdlName = deriveMDLName(mpkDef.ID)
	}

	// Generate .def.json
	defJSON := generateDefJSON(mpkDef, mdlName)

	// Determine output filename
	filename := strings.ToLower(mdlName) + ".def.json"
	outPath := filepath.Join(outputDir, filename)

	data, err := json.MarshalIndent(defJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Printf("Extracted widget definition:\n")
	fmt.Printf("  Widget ID:  %s\n", mpkDef.ID)
	fmt.Printf("  MDL name:   %s\n", mdlName)
	fmt.Printf("  Properties: %d\n", len(mpkDef.Properties))
	fmt.Printf("  Output:     %s\n", outPath)

	return nil
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

	// Generate property mappings and child slots from MPK property definitions
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
			})
		case "attribute":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Attribute",
				Operation:   "attribute",
			})
		case "association":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Association",
				Operation:   "association",
			})
		case "selection":
			def.PropertyMappings = append(def.PropertyMappings, executor.PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Selection",
				Operation:   "selection",
				Default:     p.DefaultValue,
			})
		case "boolean", "integer", "decimal", "string", "enumeration":
			m := executor.PropertyMapping{
				PropertyKey: p.Key,
				Operation:   "primitive",
			}
			if p.DefaultValue != "" {
				m.Value = p.DefaultValue
			}
			def.PropertyMappings = append(def.PropertyMappings, m)
		}
	}

	return def
}

func runWidgetInit(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	projectDir := filepath.Dir(projectPath)
	widgetsDir := filepath.Join(projectDir, "widgets")
	outputDir := filepath.Join(projectDir, ".mxcli", "widgets")

	// Load built-in registry to skip widgets that already have hand-crafted definitions
	builtinRegistry, _ := executor.NewWidgetRegistry()

	// Scan widgets/ for .mpk files
	matches, err := filepath.Glob(filepath.Join(widgetsDir, "*.mpk"))
	if err != nil {
		return fmt.Errorf("failed to scan widgets directory: %w", err)
	}
	if len(matches) == 0 {
		fmt.Println("No .mpk files found in widgets/ directory.")
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var extracted, skipped int
	for _, mpkPath := range matches {
		mpkDef, err := mpk.ParseMPK(mpkPath)
		if err != nil {
			log.Printf("warning: skipping %s: %v", filepath.Base(mpkPath), err)
			skipped++
			continue
		}

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

		// Skip if already exists on disk
		if _, err := os.Stat(outPath); err == nil {
			skipped++
			continue
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
		fmt.Printf("  %-12s %-20s %s\n", kind, mdlName, mpkDef.ID)
		extracted++
	}

	fmt.Printf("\nExtracted: %d, Skipped: %d (existing or unparseable)\n", extracted, skipped)

	// Also generate docs
	fmt.Println("\nGenerating widget documentation...")
	return generateWidgetDocs(projectDir)
}

func runWidgetDocs(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	projectDir := filepath.Dir(projectPath)
	return generateWidgetDocs(projectDir)
}

func generateWidgetDocs(projectDir string) error {
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
		mpkDef, err := mpk.ParseMPK(mpkPath)
		if err != nil {
			continue
		}

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

	fmt.Printf("Generated %d widget docs in %s\n", generated, docsDir)
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

	// Load user definitions if project path available
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath != "" {
		if err := registry.LoadUserDefinitions(projectPath); err != nil {
			log.Printf("warning: loading user widget definitions: %v", err)
		}
	}

	defs := registry.All()
	if len(defs) == 0 {
		fmt.Println("No widget definitions registered.")
		return nil
	}

	fmt.Printf("%-16s %-20s %-50s %s\n", "Kind", "MDL Name", "Widget ID", "Template")
	fmt.Printf("%-16s %-20s %-50s %s\n", strings.Repeat("-", 16), strings.Repeat("-", 20), strings.Repeat("-", 50), strings.Repeat("-", 20))
	for _, def := range defs {
		kind := def.WidgetKind
		if kind == "" {
			kind = "pluggable"
		}
		fmt.Printf("%-16s %-20s %-50s %s\n", kind, def.MDLName, def.WidgetID, def.TemplateFile)
	}
	fmt.Printf("\nTotal: %d definitions\n", len(defs))

	return nil
}
