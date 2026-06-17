package mpr

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"
)

// WidgetAdapter emits Widget nodes from .mxcli/widgets/*.def.json and
// widgets/*.mpk files. .def.json entries have priority — duplicates by
// widget ID are resolved in favour of .def.json.
type WidgetAdapter struct {
	ProjectDir string
}

func (a *WidgetAdapter) Name() string { return "widget" }

func (a *WidgetAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Widget"},
	}
}

func (a *WidgetAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

func (a *WidgetAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	if a.ProjectDir == "" {
		return nil
	}
	var events []mxgraph.Event
	seen := make(map[string]bool)

	// Priority 1: .mxcli/widgets/*.def.json
	defDir := filepath.Join(a.ProjectDir, ".mxcli", "widgets")
	if entries, err := os.ReadDir(defDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".def.json") {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			data, err := os.ReadFile(filepath.Join(defDir, entry.Name()))
			if err != nil {
				log.Printf("warning: reading %s: %v", entry.Name(), err)
				continue
			}
			var def struct {
				WidgetID   string `json:"widgetId"`
				MDLName    string `json:"mdlName"`
				WidgetKind string `json:"widgetKind,omitempty"`
			}
			if err := json.Unmarshal(data, &def); err != nil {
				log.Printf("warning: parsing %s: %v", entry.Name(), err)
				continue
			}
			if def.WidgetID == "" {
				continue
			}
			seen[def.WidgetID] = true
			node := &mxgraph.Node{
				ID:    mxgraph.NodeID(def.WidgetID),
				Label: "Widget",
				Props: map[string]any{
					"WidgetID":   def.WidgetID,
					"MDLName":    strings.ToUpper(def.MDLName),
					"WidgetKind": def.WidgetKind,
					"Source":     "def.json",
				},
			}
			if def.MDLName != "" {
				node.Props["Name"] = def.MDLName
			}
			events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
		}
	}

	// Priority 2: widgets/*.mpk (only for widget IDs not already indexed)
	mpkMap, err := mpk.FindAllMPK(a.ProjectDir)
	if err != nil {
		log.Printf("warning: scanning MPK files: %v", err)
	}
	for widgetID, mpkPath := range mpkMap {
		if seen[widgetID] {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		mpkDef, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
		if err != nil || mpkDef == nil {
			continue
		}
		widgetKind := "custom"
		if mpkDef.IsPluggable {
			widgetKind = "pluggable"
		}
		mdlName := lastIDSegment(widgetID)
		node := &mxgraph.Node{
			ID:    mxgraph.NodeID(widgetID),
			Label: "Widget",
			Props: map[string]any{
				"WidgetID":   widgetID,
				"MDLName":    strings.ToUpper(mdlName),
				"Name":       mpkDef.Name,
				"WidgetKind": widgetKind,
				"Source":     "mpk",
			},
		}
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func lastIDSegment(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	return strings.ToLower(parts[len(parts)-1])
}
