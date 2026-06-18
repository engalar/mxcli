package themescss

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// fileCache 缓存每个文件的 SCSS 文档，用于 diff。
type fileCache map[string]*scss.ScssDocument

// watchFiles 启动文件系统监听，在 SCSS 文件变更时发送增量事件。
// 由 adapter.go 中的 Watch 方法调用。
func (a *ThemeScssAdapter) watchFiles(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// 收集需要监听的目录
	watchDirs := collectWatchDirs(a.ProjectDir)
	for _, dir := range watchDirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("themescss watch: cannot watch %s: %v", dir, err)
		}
	}

	cache := make(fileCache)

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !strings.HasSuffix(event.Name, ".scss") {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				relPath, err := filepath.Rel(a.ProjectDir, event.Name)
				if err != nil {
					continue
				}

				data, err := os.ReadFile(event.Name)
				if err != nil {
					continue
				}

				newDoc, err := scss.Parse(event.Name, string(data))
				if err != nil {
					continue
				}

				oldDoc := cache[event.Name]
				if oldDoc == nil {
					// 首次变更：全量 emit
					var events []mxgraph.Event
					source := classifySource(relPath)
					module := extractModule(relPath)
					for _, decl := range newDoc.Vars {
						events = append(events, mxgraph.Event{
							Type: mxgraph.NodeCreated,
							Node: varNode(decl, source, module, relPath),
						})
					}
					if len(events) > 0 {
						sink.Emit(events)
					}
				} else {
					// 增量 diff
					events := diffDocuments(oldDoc, newDoc, relPath)
					if len(events) > 0 {
						sink.Emit(events)
					}
				}
				cache[event.Name] = newDoc

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("themescss watch error: %v", err)
			}
		}
	}()

	stop := func() { watcher.Close() }
	return stop, nil
}

// collectWatchDirs 收集所有需要监听的 SCSS 目录。
func collectWatchDirs(projectDir string) []string {
	dirs := []string{
		filepath.Join(projectDir, "theme", "web"),
	}
	tsDir := filepath.Join(projectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				webDir := filepath.Join(tsDir, entry.Name(), "web")
				if info, statErr := os.Stat(webDir); statErr == nil && info.IsDir() {
					dirs = append(dirs, webDir)
				}
			}
		}
	}
	return dirs
}

// diffDocuments 比较两个版本的 SCSS 文档，返回增量事件。
func diffDocuments(oldDoc, newDoc *scss.ScssDocument, relPath string) []mxgraph.Event {
	var events []mxgraph.Event

	oldVars := make(map[string]scss.ScssVarDecl)
	for _, v := range oldDoc.Vars {
		oldVars[v.Name] = v
	}
	newVars := make(map[string]scss.ScssVarDecl)
	for _, v := range newDoc.Vars {
		newVars[v.Name] = v
	}

	source := classifySource(relPath)
	module := extractModule(relPath)

	// 新增和更新
	for name, newDecl := range newVars {
		if oldDecl, ok := oldVars[name]; !ok {
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: varNode(newDecl, source, module, relPath),
			})
		} else if oldDecl.Value != newDecl.Value || oldDecl.IsActive != newDecl.IsActive {
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeUpdated,
				Node: varNode(newDecl, source, module, relPath),
			})
		}
	}

	// 删除
	for name, oldDecl := range oldVars {
		if _, ok := newVars[name]; !ok {
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeDeleted,
				Node: varNode(oldDecl, source, module, relPath),
			})
		}
	}

	return events
}

func classifySource(relPath string) string {
	switch {
	case strings.Contains(relPath, "theme/web/custom-variables.scss"):
		return "project-custom-variables"
	case strings.Contains(relPath, "theme/web/main.scss"):
		return "project-main"
	case strings.Contains(relPath, "themesource/"):
		if strings.Contains(relPath, "_variables.scss") {
			return "atlas-core-default"
		}
		if strings.Contains(relPath, "/main.scss") {
			return "module-main"
		}
		return "module-variables"
	default:
		return "other"
	}
}

func extractModule(relPath string) string {
	if strings.HasPrefix(relPath, "themesource/") {
		parts := strings.SplitN(relPath, "/", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}
