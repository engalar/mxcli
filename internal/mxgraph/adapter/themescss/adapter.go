package themescss

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// scssSource 描述一个 SCSS 文件及其在主题系统中的角色。
type scssSource struct {
	RelativePath string // 相对于 ProjectDir
	Source       string // "project-custom-variables" | "project-main" | "module-variables" | ...
	Module       string // 模块名（仅模块级时有效）
}

// ThemeScssAdapter 索引所有 SCSS 文件中的主题变量为 ThemeVariable 节点。
type ThemeScssAdapter struct {
	ProjectDir string
}

func (a *ThemeScssAdapter) Name() string { return "themescss" }

func (a *ThemeScssAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"ThemeVariable"},
	}
}

func (a *ThemeScssAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	sources := a.discoverSources()
	var events []mxgraph.Event

	for _, src := range sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fullPath := filepath.Join(a.ProjectDir, src.RelativePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		doc, err := scss.Parse(src.RelativePath, string(data))
		if err != nil {
			continue
		}

		for _, decl := range doc.Vars {
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: varNode(decl, src.Source, src.Module, src.RelativePath),
			})
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// discoverSources 扫描项目目录，发现所有主题 SCSS 文件。
func (a *ThemeScssAdapter) discoverSources() []scssSource {
	var sources []scssSource

	// 项目级文件
	projectFiles := []struct {
		relPath string
		source  string
	}{
		{"theme/web/custom-variables.scss", "project-custom-variables"},
		{"theme/web/main.scss", "project-main"},
	}
	for _, pf := range projectFiles {
		if fileExists(filepath.Join(a.ProjectDir, pf.relPath)) {
			sources = append(sources, scssSource{
				RelativePath: pf.relPath,
				Source:       pf.source,
			})
		}
	}

	// Atlas Core 默认值（只读参考）
	atlasVarsPath := filepath.Join("themesource", "atlas_core", "web", "_variables.scss")
	if fileExists(filepath.Join(a.ProjectDir, atlasVarsPath)) {
		sources = append(sources, scssSource{
			RelativePath: atlasVarsPath,
			Source:       "atlas-core-default",
		})
	}

	// 模块级文件
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err != nil {
		return sources
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		module := entry.Name()
		modSources := []struct {
			relPath string
			source  string
		}{
			{filepath.Join("themesource", module, "web", "variables.scss"), "module-variables"},
			{filepath.Join("themesource", module, "web", "main.scss"), "module-main"},
		}
		for _, ms := range modSources {
			if fileExists(filepath.Join(a.ProjectDir, ms.relPath)) {
				sources = append(sources, scssSource{
					RelativePath: ms.relPath,
					Source:       ms.source,
					Module:       module,
				})
			}
		}
	}

	return sources
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *ThemeScssAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
