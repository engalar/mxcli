package mpr

import "sync"

// BsonDocCache 缓存按 unit ID 解码的 BSON map，避免多个 adapter
// 对同一页面进行重复的 bson.Unmarshal。
//
// 使用方式：在 cmd_graph.go 中创建实例，注入到所有 page-walking adapter。
type BsonDocCache interface {
	// Get 返回已缓存的解码文档，ok=false 表示无缓存。
	Get(unitID string) (doc map[string]any, ok bool)
	// Set 存入解码文档。
	Set(unitID string, doc map[string]any)
}

// MapBsonDocCache 基于 sync.Map 的 BsonDocCache 实现。
type MapBsonDocCache struct {
	m sync.Map
}

func NewBsonDocCache() *MapBsonDocCache {
	return &MapBsonDocCache{}
}

func (c *MapBsonDocCache) Get(unitID string) (map[string]any, bool) {
	v, ok := c.m.Load(unitID)
	if !ok {
		return nil, false
	}
	doc, ok := v.(map[string]any)
	return doc, ok
}

func (c *MapBsonDocCache) Set(unitID string, doc map[string]any) {
	c.m.Store(unitID, doc)
}
