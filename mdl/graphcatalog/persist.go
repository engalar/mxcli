package graphcatalog

import (
	"fmt"
	"path/filepath"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// MarshalSnapshot 将图序列化为 gob 二进制（委托给 mxgraph.MarshalSnapshot）。
func (pg *ProjectGraph) MarshalSnapshot() ([]byte, error) {
	return mxgraph.MarshalSnapshot(pg.mgr.Query())
}

// UnmarshalSnapshot 从 gob 二进制恢复 ProjectGraph（只读，无注册适配器）。
func UnmarshalSnapshot(data []byte) (*ProjectGraph, error) {
	g, err := mxgraph.UnmarshalSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("graphcatalog: unmarshal snapshot: %w", err)
	}
	return &ProjectGraph{mgr: mxgraph.NewIndexManagerFromGraph(g)}, nil
}

// SnapshotPath 返回给定项目目录下 graph.gob 的标准路径，替代旧的 catalog.db。
func SnapshotPath(projectDir string) string {
	return filepath.Join(projectDir, ".mxcli", "graph.gob")
}
