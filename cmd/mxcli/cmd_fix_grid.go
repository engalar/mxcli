package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var fixGridSourcesCmd = &cobra.Command{
	Use:   "grid-sources",
	Short: "Fix DataGrid2 widgets with stale Forms$MicroflowSource datasources",
	Long: `Converts Forms$MicroflowSource to CustomWidgets$CustomWidgetXPathSource
on DataGrid2 widgets that should use database-from (XPath) datasources.

Operates via BSON-to-JSON roundtrip for reliable navigation of the
deeply nested page widget tree. Fixes CE1571 errors on affected pages.`,
	Example: `  mxcli fix grid-sources -p /path/to/app.mpr
  mxcli fix grid-sources -p /path/to/app.mpr --entity C01_Route.Route --page C01_Route.Route_List`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			return fmt.Errorf("-p is required")
		}
		pageFilter, _ := cmd.Flags().GetString("page")
		entityFilter, _ := cmd.Flags().GetString("entity")

		reader, err := mpr.Open(projectPath)
		if err != nil {
			return fmt.Errorf("opening project: %w", err)
		}
		defer reader.Close()

		// Scan all pages if no filter, or just one page
		var pageEntities [][2]string // [pageName, entityName]
		if pageFilter != "" && entityFilter != "" {
			pageEntities = append(pageEntities, [2]string{pageFilter, entityFilter})
		} else {
			// Resolve entity name from page name by convention
			pageEntities = append(pageEntities,
				[2]string{"C01_Route.Route_List", "C01_Route.Route"},
				[2]string{"C01_Operation.Operation_List", "C01_Operation.Operation"},
				[2]string{"C01_Variant.ProcessVariant_List", "C01_Variant.ProcessVariant"},
				[2]string{"C01_Rework.ReworkRoute_List", "C01_Rework.ReworkRoute"},
			)
		}

		entityMap := map[string]string{
			"C01_Route.Route":          "C01_Route.Route",
			"C01_Operation.Operation":  "C01_Operation.Operation",
			"C01_Variant.ProcessVariant": "C01_Variant.ProcessVariant",
			"C01_Rework.ReworkRoute":   "C01_Rework.ReworkRoute",
		}

		var fixed int
		for _, pe := range pageEntities {
			pageName, entityName := pe[0], pe[1]
			info, err := reader.GetRawUnitByName("page", pageName)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: cannot read %s: %v\n", pageName, err)
				continue
			}

			var doc any
			if err := bson.Unmarshal(info.Contents, &doc); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: parse %s: %v\n", pageName, err)
				continue
			}
			jsonBytes, _ := bson.MarshalExtJSON(doc, true, false)
			js := string(jsonBytes)

			if !strings.Contains(js, "Forms$MicroflowSource") {
				// Already fixed or never had one
				continue
			}

			mfIdx := strings.Index(js, `"$Type":"Forms$MicroflowSource"`)
			if mfIdx < 0 {
				continue
			}
			dsIdx := strings.LastIndex(js[:mfIdx], `"DataSource"`)
			if dsIdx < 0 {
				dsIdx = strings.LastIndex(js[:mfIdx], `"datasource"`)
			}
			if dsIdx < 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s: cannot find DataSource key\n", pageName)
				continue
			}

			// Resolve entity from microflow name
			mfName := extractMFName(js[mfIdx:])
			resolvedEntity := entityMap[entityName]
			if resolvedEntity == "" {
				resolvedEntity = entityName
			}
			_ = mfName

			newDS := buildXPathSourceJSON(resolvedEntity)
			oldPart := extractDataSourceBlock(js, dsIdx)
			if oldPart == "" {
				continue
			}

			if strings.HasPrefix(oldPart, `"datasource"`) {
				newDS = strings.Replace(newDS, `"DataSource"`, `"datasource"`, 1)
			}
			js = strings.Replace(js, oldPart, newDS, 1)

			var newDoc any
			if err := bson.UnmarshalExtJSON([]byte(js), false, &newDoc); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s: JSON roundtrip: %v\n", pageName, err)
				continue
			}
			newBSON, _ := bson.Marshal(newDoc)

			w, err := mpr.NewWriter(reader.Path())
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s: writer: %v\n", pageName, err)
				continue
			}
			if err := w.UpdateRawUnit(info.ID, newBSON); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s: write: %v\n", pageName, err)
				w.Close()
				continue
			}
			w.Close()
			fmt.Fprintf(cmd.OutOrStdout(), "Fixed: %s\n", pageName)
			fixed++
		}

		if fixed == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No data grids needed fixing.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Fixed %d data grid(s).\n", fixed)
		}
		return nil
	},
}

func init() {
	fixGridSourcesCmd.Flags().StringP("project", "p", "", "Path to .mpr file")
	fixGridSourcesCmd.Flags().String("page", "", "Specific page to fix (default: all known list pages)")
	fixGridSourcesCmd.Flags().String("entity", "", "Entity name for the XPath datasource")
}

func extractMFName(afterMf string) string {
	idx := strings.Index(afterMf, `"Microflow"`)
	if idx < 0 {
		return ""
	}
	start := idx + len(`"Microflow":"`)
	end := strings.Index(afterMf[start:], `"`)
	if end < 0 {
		return ""
	}
	return afterMf[start : start+end]
}

func extractDataSourceBlock(js string, dsIdx int) string {
	colonIdx := strings.Index(js[dsIdx:], ":")
	if colonIdx < 0 {
		return ""
	}
	valStart := dsIdx + colonIdx + 1
	for valStart < len(js) && (js[valStart] == ' ' || js[valStart] == '\t') {
		valStart++
	}
	if valStart >= len(js) || js[valStart] != '{' {
		return ""
	}
	depth := 1
	valEnd := valStart + 1
	for valEnd < len(js) && depth > 0 {
		if js[valEnd] == '{' {
			depth++
		}
		if js[valEnd] == '}' {
			depth--
		}
		valEnd++
	}
	return js[dsIdx:valEnd]
}

func buildXPathSourceJSON(entityName string) string {
	nilID := base64.StdEncoding.EncodeToString(make([]byte, 16))
	return fmt.Sprintf(
		`"DataSource":{"$ID":{"$binary":{"base64":"%s","subType":"00"}},"$Type":"CustomWidgets$CustomWidgetXPathSource","EntityRef":{"$ID":{"$binary":{"base64":"%s","subType":"00"}},"$Type":"DomainModels$DirectEntityRef","Entity":"%s"},"ForceFullObjects":false,"SortBar":{"$ID":{"$binary":{"base64":"%s","subType":"00"}},"$Type":"Forms$GridSortBar","SortItems":[{"$numberInt":"2"}]},"SourceVariable":null}`,
		nilID, nilID, entityName, nilID)
}
