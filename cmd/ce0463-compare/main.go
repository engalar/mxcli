package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCw "github.com/mendixlabs/mxcli/modelsdk/gen/customwidgets"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <built.mpr> <golden.mpr>", os.Args[0])
	}

	builtPath := os.Args[1]
	goldenPath := os.Args[2]

	dec := codec.NewDecoder(codec.DefaultRegistry)

	builtWidgets := extractWidgets(builtPath, dec)
	goldenWidgets := extractWidgets(goldenPath, dec)

	fmt.Printf("Built:  %d DataGrid2 widgets\n", len(builtWidgets))
	fmt.Printf("Golden: %d DataGrid2 widgets\n", len(goldenWidgets))

	matched := 0
	for name, builtElem := range builtWidgets {
		goldenElem, ok := goldenWidgets[name]
		if !ok {
			fmt.Printf("\nWIDGET %q: only in built, skipping\n", name)
			continue
		}
		matched++

		fmt.Printf("\n========================================\n")
		fmt.Printf("COMPARING WIDGET: %q\n", name)
		fmt.Printf("========================================\n")

		builtDump := dumpElement(builtElem)
		goldenDump := dumpElement(goldenElem)

		bTotal, bNonID := countDump(builtDump)
		gTotal, gNonID := countDump(goldenDump)
		fmt.Printf("Built fields:  %d total, %d non-ID\n", bTotal, bNonID)
		fmt.Printf("Golden fields: %d total, %d non-ID\n", gTotal, gNonID)

		builtFiltered := filterDump(builtDump, false)
		goldenFiltered := filterDump(goldenDump, false)

		onlyInA, onlyInB, different := compareDumps(goldenFiltered, builtFiltered)
		printDiff(onlyInA, onlyInB, different)

		if len(onlyInA) == 0 && len(onlyInB) == 0 && len(different) == 0 {
			fmt.Println("\n✓ BUILDER MATCHES GOLDEN for this widget")
		}
	}

	if matched == 0 {
		fmt.Println("\nNo matching widgets found between built and golden")
	}

	for name := range goldenWidgets {
		if _, ok := builtWidgets[name]; !ok {
			fmt.Printf("\nWIDGET %q: only in golden (not in built)\n", name)
		}
	}
}

func extractWidgets(path string, dec *codec.Decoder) map[string]element.Element {
	result := make(map[string]element.Element)

	store, err := codec.Open(path)
	if err != nil {
		log.Printf("Warning: cannot open %s: %v", path, err)
		return result
	}
	defer store.Close()

	for _, unitID := range store.ListUnits() {
		raw, err := store.LoadUnit(unitID.ID)
		if err != nil {
			continue
		}

		elem, err := dec.Decode(raw)
		if err != nil {
			continue
		}

		page, ok := elem.(*genPg.Page)
		if !ok {
			continue
		}

		element.Walk(page, func(w element.Element) bool {
			cw, ok := w.(*genCw.CustomWidget)
			if !ok {
				return true
			}
			cwType, ok := cw.Type().(*genCw.CustomWidgetType)
			if !ok {
				return true
			}
			widgetID := cwType.WidgetId()
			if widgetID != "com.mendix.widget.web.datagrid.Datagrid" &&
				widgetID != "com.mendix.widget.web.gallery.Gallery" {
				return true
			}
			name := cw.Name()
			result[name] = cw
			return true
		})
	}

	return result
}

func init() {
	_ = genPg.Page{}
	_ = genCw.CustomWidget{}
	_ = bson.MarshalExtJSON
	// BSON $Type "Forms$FormCallArgument" is a storage name used by Mendix 11+
	// that maps to the same structure as "Forms$LayoutCallArgument" (has Widgets).
	codec.DefaultRegistry.RegisterAlias("Forms$FormCallArgument", "Forms$LayoutCallArgument")
}
