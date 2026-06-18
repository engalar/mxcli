package designdprops

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

type jsonCache map[string]map[string]map[string]struct{} // filePath → widgetType → {name1, name2, ...}

// watchFiles 启动文件系统监听，在 design-properties.json 变更时发送增量事件。
func (a *DesignPropertyAdapter) watchFiles(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// 监视所有的 themesource/*/web/ 目录
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				webDir := filepath.Join(tsDir, entry.Name(), "web")
				if info, statErr := os.Stat(webDir); statErr == nil && info.IsDir() {
					watcher.Add(webDir)
				}
			}
		}
	}

	cache := make(jsonCache)

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
				if !strings.HasSuffix(event.Name, "design-properties.json") {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				relPath, err := filepath.Rel(a.ProjectDir, event.Name)
				if err != nil {
					continue
				}

				// 重新解析
				data, err := os.ReadFile(event.Name)
				if err != nil {
					continue
				}

				var fileProps map[string][]rawDesignPropDef
				if err := parseDesignProps(data, &fileProps); err != nil {
					continue
				}

				// 对比缓存 diff
				oldKeys := cache[event.Name]
				newKeys := buildKeySet(fileProps)

				var fileEvents []mxgraph.Event
				module := extractModuleName(relPath)

				// 新增/更新
				for wt, names := range newKeys {
					for name := range names {
						dpID := mxgraph.NodeID(module + "." + wt + "." + name)
						if _, exists := oldKeys[wt][name]; exists {
							// 更新
							fileEvents = append(fileEvents, mxgraph.Event{
								Type: mxgraph.NodeUpdated,
								Node: &mxgraph.Node{ID: dpID},
							})
						} else {
							// 新增
							fileEvents = append(fileEvents, mxgraph.Event{
								Type: mxgraph.NodeCreated,
								Node: &mxgraph.Node{ID: dpID},
							})
						}
					}
				}
				// 删除
				for wt, names := range oldKeys {
					for name := range names {
						if _, exists := newKeys[wt][name]; !exists {
							dpID := mxgraph.NodeID(module + "." + wt + "." + name)
							fileEvents = append(fileEvents, mxgraph.Event{
								Type: mxgraph.NodeDeleted,
								Node: &mxgraph.Node{ID: dpID},
							})
						}
					}
				}

				if len(fileEvents) > 0 {
					sink.Emit(fileEvents)
				}
				cache[event.Name] = newKeys

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("designdprops watch error: %v", err)
			}
		}
	}()

	stop := func() { watcher.Close() }
	return stop, nil
}

func parseDesignProps(data []byte, out *map[string][]rawDesignPropDef) error {
	return json.Unmarshal(data, out)
}

func buildKeySet(fileProps map[string][]rawDesignPropDef) map[string]map[string]struct{} {
	keys := make(map[string]map[string]struct{})
	for wt, props := range fileProps {
		if keys[wt] == nil {
			keys[wt] = make(map[string]struct{})
		}
		for _, p := range props {
			keys[wt][p.Name] = struct{}{}
		}
	}
	return keys
}

func extractModuleName(relPath string) string {
	parts := strings.SplitN(relPath, "/", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
