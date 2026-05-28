// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var extractTemplatesCmd = &cobra.Command{
	Use:   "extract-templates",
	Short: "Extract widget templates from a Mendix project",
	Long: `Extract pluggable widget type definitions from a Mendix project
and save them as JSON templates for use in mxcli.

This command scans all pages and snippets in the project for pluggable
widgets and extracts their type definitions. The resulting JSON files can
be embedded in mxcli for consistent widget creation across projects.

Example:
  mxcli extract-templates -p app.mpr -o templates/mendix-11.6/`,
	RunE: runExtractTemplates,
}

func init() {
	extractTemplatesCmd.Flags().StringP("project", "p", "", "Path to the Mendix project (.mpr file)")
	extractTemplatesCmd.Flags().StringP("output", "o", "", "Output directory for templates")
	extractTemplatesCmd.MarkFlagRequired("project")
	extractTemplatesCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(extractTemplatesCmd)
}

// extractedWidgetTemplate is the JSON structure for a widget template file.
type extractedWidgetTemplate struct {
	WidgetID      string         `json:"widgetId"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	ExtractedFrom string         `json:"extractedFrom"`
	Type          map[string]any `json:"type"`
	Object        map[string]any `json:"object,omitempty"`
}

func runExtractTemplates(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	outputDir, _ := cmd.Flags().GetString("output")

	reader, err := mmpr.Open(projectPath)
	if err != nil {
		return fmt.Errorf("failed to open project: %w", err)
	}
	defer reader.Close()

	version, _ := reader.GetMendixVersion()
	fmt.Printf("Extracting templates from Mendix %s project\n", version)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	widgets, err := reader.ListAllCustomWidgetTypes()
	if err != nil {
		return fmt.Errorf("failed to scan project for widget types: %w", err)
	}
	if len(widgets) == 0 {
		fmt.Println("No pluggable widget types found in this project.")
		return nil
	}
	fmt.Printf("Found %d widget type(s)\n", len(widgets))

	extracted := 0
	for _, w := range widgets {
		rawTypeD, ok := w.RawType.(bson.D)
		if !ok {
			fmt.Printf("  [SKIP] %s: RawType is not bson.D\n", w.WidgetID)
			continue
		}
		typeMap, err := bsonDToMap(rawTypeD)
		if err != nil {
			fmt.Printf("  [SKIP] %s: failed to convert type BSON: %v\n", w.WidgetID, err)
			continue
		}

		var objectMap map[string]any
		if w.RawObject != nil {
			rawObjectD, ok := w.RawObject.(bson.D)
			if !ok {
				fmt.Printf("  [SKIP] %s: RawObject is not bson.D\n", w.WidgetID)
				continue
			}
			objectMap, err = bsonDToMap(rawObjectD)
			if err != nil {
				fmt.Printf("  [SKIP] %s: failed to convert object BSON: %v\n", w.WidgetID, err)
				continue
			}
		}

		template := extractedWidgetTemplate{
			WidgetID:      w.WidgetID,
			Version:       version,
			ExtractedFrom: w.UnitID,
			Type:          typeMap,
			Object:        objectMap,
		}

		filename := filenameFromWidgetID(w.WidgetID)
		outPath := filepath.Join(outputDir, filename)
		data, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			fmt.Printf("  [SKIP] %s: failed to marshal JSON: %v\n", w.WidgetID, err)
			continue
		}

		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Printf("  [SKIP] %s: failed to write file: %v\n", w.WidgetID, err)
			continue
		}

		fmt.Printf("  [OK] %s -> %s\n", w.WidgetID, filename)
		extracted++
	}

	fmt.Printf("\nExtracted %d/%d widget templates to %s\n", extracted, len(widgets), outputDir)
	return nil
}

// bsonDToMap converts a bson.D to a JSON-compatible map.
func bsonDToMap(doc bson.D) (map[string]any, error) {
	result := make(map[string]any)
	for _, elem := range doc {
		result[elem.Key] = convertBsonValue(elem.Value)
	}
	return result, nil
}

// convertBsonValue converts BSON values to JSON-compatible types.
func convertBsonValue(v any) any {
	switch val := v.(type) {
	case bson.D:
		m := make(map[string]any)
		for _, elem := range val {
			m[elem.Key] = convertBsonValue(elem.Value)
		}
		return m
	case bson.A:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = convertBsonValue(item)
		}
		return arr
	case bson.Binary:
		return fmt.Sprintf("%x", val.Data)
	case []byte:
		return fmt.Sprintf("%x", val)
	default:
		return val
	}
}

// filenameFromWidgetID generates a kebab-case filename from a widget ID.
// e.g. "com.mendix.widget.web.combobox.Combobox" -> "combobox.json"
func filenameFromWidgetID(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	name := parts[len(parts)-1]
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String()) + ".json"
}
