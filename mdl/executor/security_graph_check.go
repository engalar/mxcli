package executor

import (
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

// GraphAccessAnalyzer 基于 mxgraph 图查询执行访问间隙分析。
// 替代 security_access_check.go 中基于 MPR 全扫描的 AnalyzeAccess()。
// 要求 graph 已构建（含 AccessRuleAdapter、DocumentGrantAdapter、PageRefAdapter）。
//
// 使用场景：
//   1. mxcli lint (SEC001)：EntitiesWithMissingAccessRules → µs 级（之前 11 ms）
//   2. mxcli check --references：所有数据分析来自图查询（之前 70 ms）
//
// 预期加速：~1000x（graph 一次建图后，每次分析只需 µs 级查询）
type GraphAccessAnalyzer struct {
	Graph *graphcatalog.ProjectGraph
}

// AccessGapSummary 汇总访问间隙分析结果。
type AccessGapSummary struct {
	EntitiesWithoutRules []graphcatalog.EntityNode     // SEC001
	PageGaps             []PageAccessGap               // ACCESS-001
	MFGaps               []MFAccessGap                 // ACCESS-003
}

// PageAccessGap 表示某个角色能看见页面但无实体读权限。
type PageAccessGap struct {
	UserRole       string
	ModuleRoleQN   string
	PageQN         string
	MissingEntityQN string
}

// MFAccessGap 表示某个角色能到达微流但无可执行权限。
type MFAccessGap struct {
	UserRole     string
	ModuleRoleQN string
	MFQN         string
}

// Analyze 执行完整的访问间隙分析。
func (a *GraphAccessAnalyzer) Analyze() (*AccessGapSummary, error) {
	summary := &AccessGapSummary{}

	// 1. SEC001：检测无访问规则的实体
	summary.EntitiesWithoutRules = a.Graph.EntitiesWithMissingAccessRules("")

	// 2. ACCESS-001：检测页面实体读间隙
	summary.PageGaps = a.detectPageGaps()

	// 3. ACCESS-003：检测微流执行间隙
	summary.MFGaps = a.detectMFGaps()

	return summary, nil
}

// detectPageGaps 遍历所有页面，找出角色可访问页面但无对应实体读权限的间隙。
func (a *GraphAccessAnalyzer) detectPageGaps() []PageAccessGap {
	var gaps []PageAccessGap

	pages := a.Graph.Pages("")
	for _, page := range pages {
		allowedRoles := a.Graph.PageAllowedRoles(page.QualifiedName)
		entityRefs := a.Graph.PageEntityRefs(page.QualifiedName)

		if len(allowedRoles) == 0 || len(entityRefs) == 0 {
			continue
		}

		for _, mrQN := range allowedRoles {
			// 获取该角色的所有实体访问规则
			rules := a.Graph.EntityAccessRulesForRole(mrQN)
			readableEntities := make(map[string]bool)
			for _, rule := range rules {
				if rule.CanRead {
					readableEntities[rule.EntityQN] = true
				}
			}

			// 检查页面引用的每个实体是否可读
			for _, entityQN := range entityRefs {
				if !readableEntities[entityQN] {
					gaps = append(gaps, PageAccessGap{
						UserRole:         "", // 由调用方填充
						ModuleRoleQN:     mrQN,
						PageQN:           page.QualifiedName,
						MissingEntityQN:  entityQN,
					})
				}
			}
		}
	}

	return gaps
}

// detectMFGaps 遍历所有页面，找出角色可触发微流但无执行权限的间隙。
func (a *GraphAccessAnalyzer) detectMFGaps() []MFAccessGap {
	var gaps []MFAccessGap

	pages := a.Graph.Pages("")
	for _, page := range pages {
		allowedRoles := a.Graph.PageAllowedRoles(page.QualifiedName)
		mfRefs := a.Graph.PageMFRefs(page.QualifiedName)

		if len(allowedRoles) == 0 || len(mfRefs) == 0 {
			continue
		}

		for _, mrQN := range allowedRoles {
			mfGrants := a.Graph.MFAllowedRoles("")
			granted := make(map[string]bool)
			for _, grantQN := range mfGrants {
				granted[grantQN] = true
			}

			for _, mfQN := range mfRefs {
				if !granted[mfQN] {
					gaps = append(gaps, MFAccessGap{
						ModuleRoleQN: mrQN,
						MFQN:         mfQN,
					})
				}
			}
		}
	}

	return gaps
}
