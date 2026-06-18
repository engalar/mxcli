package designdprops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// DesignPropertyAdapter 索引设计属性定义（design-properties.json）为 DesignProperty 节点。
type DesignPropertyAdapter struct {
	ProjectDir string
}

func (a *DesignPropertyAdapter) Name() string { return "designdprops" }

func (a *DesignPropertyAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"DesignProperty"},
	}
}

// rawDesignPropDef 对应 design-properties.json 中的单个属性条目。
type rawDesignPropDef struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Category    string            `json:"category,omitempty"`
	Description string            `json:"description,omitempty"`
	Class       string            `json:"class,omitempty"`
	Options     []rawDesignOption `json:"options,omitempty"`
}

type rawDesignOption struct {
	Name    string `json:"name"`
	Class   string `json:"class,omitempty"`
	Preview string `json:"preview,omitempty"`
	Variable string `json:"variable,omitempty"`
}

func (a *DesignPropertyAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err != nil {
		return nil
	}

	var events []mxgraph.Event
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		module := entry.Name()
		dpPath := filepath.Join(tsDir, module, "web", "design-properties.json")
		data, err := os.ReadFile(dpPath)
		if err != nil {
			continue
		}

		var fileProps map[string][]rawDesignPropDef
		if err := json.Unmarshal(data, &fileProps); err != nil {
			continue
		}

		for widgetType, props := range fileProps {
			for _, p := range props {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				refVars := extractReferencedVars(p)
				options := make([]string, len(p.Options))
				for i, o := range p.Options {
					options[i] = o.Name
				}

				nodeID := mxgraph.NodeID(fmt.Sprintf("%s.%s.%s", module, widgetType, p.Name))
				nodeProps := map[string]any{
					"$Type":          "DesignProperty",
					"WidgetType":     widgetType,
					"Name":           p.Name,
					"Type":           p.Type,
					"Category":       p.Category,
					"Description":    p.Description,
					"Class":          p.Class,
					"Options":        options,
					"ReferencedVars": refVars,
					"SourceModule":   module,
					"QualifiedName":  fmt.Sprintf("%s.%s.%s", module, widgetType, p.Name),
				}

				events = append(events, mxgraph.Event{
					Type: mxgraph.NodeCreated,
					Node: &mxgraph.Node{ID: nodeID, Label: "DesignProperty", Props: nodeProps},
				})
			}
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// extractReferencedVars 从设计属性选项中提取引用的主题变量名。
func extractReferencedVars(p rawDesignPropDef) []string {
	var vars []string
	seen := map[string]bool{}
	for _, o := range p.Options {
		for _, ref := range []string{o.Preview, o.Variable} {
			if ref != "" && !seen[ref] {
				seen[ref] = true
				vars = append(vars, ref)
			}
		}
	}
	return vars
}

func (a *DesignPropertyAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
