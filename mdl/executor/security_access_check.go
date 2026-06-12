// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMF "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// entityAccessSummary captures the effective access a module role has on an entity.
type entityAccessSummary struct {
	canRead   bool // DefaultMemberAccessRights is ReadOnly or ReadWrite
	canWrite  bool // DefaultMemberAccessRights is ReadWrite
	canCreate bool
	canDelete bool
}

// AnalyzeAccess scans the connected MPR and returns all permission gaps
// found between page/MF grants and entity/execute grants.
// Returns nil when no MPR is connected.
// The second return value carries non-fatal warnings (e.g. pages that could
// not be parsed); callers should surface these to the user.
func (e *Executor) AnalyzeAccess() ([]AccessGap, []string, error) {
	ctx := e.newExecContext(context.Background())
	if ctx == nil || ctx.Backend == nil || !ctx.Connected() {
		return nil, nil, nil
	}

	urToMR, err := buildUserRoleMap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	entityGrants, err := buildEntityGrants(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	// ACC-001: load pages and microflows once; share between grant-builders.
	pl, err := loadPages(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}
	ml, err := loadMicroflows(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	pageGrants, mfGrants, err := buildDocumentGrants(ctx, pl, ml)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	mfMetaMap, err := buildMFMeta(ctx, ml)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	pageModels, warnings, err := buildPageModels(ctx, pl)
	if err != nil {
		return nil, nil, fmt.Errorf("AnalyzeAccess: %w", err)
	}

	return detectGaps(urToMR, entityGrants, pageGrants, mfGrants, mfMetaMap, pageModels), warnings, nil
}

// buildUserRoleMap reads ProjectSecurity and returns UserRoleName → []ModuleRoleQN.
func buildUserRoleMap(ctx *ExecContext) (map[string][]string, error) {
	ps, err := ctx.SecurityProjectManager.GetProjectSecurityGen()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	if ps == nil {
		return result, nil
	}
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok {
			continue
		}
		result[ur.Name()] = ur.ModuleRolesQualifiedNames()
	}
	return result, nil
}

// buildEntityGrants reads every domain model's AccessRules and returns
// ModuleRoleQN → (EntityQN → entityAccessSummary).
func buildEntityGrants(ctx *ExecContext) (map[string]map[string]entityAccessSummary, error) {
	result := make(map[string]map[string]entityAccessSummary)

	dms, err := ctx.DomainModelReader.ListDomainModelsGen()
	if err != nil {
		return nil, err
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil, err
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(dm.ID()))
		moduleName := h.GetModuleName(modID)
		for _, entItem := range dm.EntitiesItems() {
			ent, ok := entItem.(*genDM.Entity)
			if !ok {
				continue
			}
			entityQN := moduleName + "." + ent.Name()
			for _, arItem := range ent.AccessRulesItems() {
				ar, ok := arItem.(*genDM.AccessRule)
				if !ok {
					continue
				}
				rights := ar.DefaultMemberAccessRights()
				summary := entityAccessSummary{
					canCreate: ar.AllowCreate(),
					canDelete: ar.AllowDelete(),
					canRead:   rights == "ReadOnly" || rights == "ReadWrite",
					canWrite:  rights == "ReadWrite",
				}
				for _, mrQN := range ar.ModuleRolesQualifiedNames() {
					if result[mrQN] == nil {
						result[mrQN] = make(map[string]entityAccessSummary)
					}
					// Merge: any rule granting a permission wins.
					existing := result[mrQN][entityQN]
					existing.canRead = existing.canRead || summary.canRead
					existing.canWrite = existing.canWrite || summary.canWrite
					existing.canCreate = existing.canCreate || summary.canCreate
					existing.canDelete = existing.canDelete || summary.canDelete
					result[mrQN][entityQN] = existing
				}
			}
		}
	}
	return result, nil
}

// pageLoad holds the results of a single ListPagesGen call plus the hierarchy,
// so both buildDocumentGrants and buildPageModels share the same data (ACC-001).
type pageLoad struct {
	pages     []*genPg.Page
	hierarchy *ContainerHierarchy
}

// mfLoad holds the results of a single ListMicroflowsGen call plus hierarchy (ACC-001).
type mfLoad struct {
	mfs       []*genMF.Microflow
	hierarchy *ContainerHierarchy
}

// loadPages fetches all pages from the backend exactly once.
func loadPages(ctx *ExecContext) (*pageLoad, error) {
	pages, err := ctx.PageReader.ListPagesGen()
	if err != nil {
		return nil, err
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil, err
	}
	return &pageLoad{pages: pages, hierarchy: h}, nil
}

// loadMicroflows fetches all microflows from the backend exactly once.
func loadMicroflows(ctx *ExecContext) (*mfLoad, error) {
	mfs, err := ctx.MicroflowReader.ListMicroflowsGen()
	if err != nil {
		return nil, err
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil, err
	}
	return &mfLoad{mfs: mfs, hierarchy: h}, nil
}

// buildDocumentGrants returns:
//   - pageGrants: ModuleRoleQN → []pageQN
//   - mfGrants:   ModuleRoleQN → []mfQN  (execute grants)
func buildDocumentGrants(_ *ExecContext, pl *pageLoad, ml *mfLoad) (pageGrants, mfGrants map[string][]string, err error) {
	pageGrants = make(map[string][]string)
	mfGrants = make(map[string][]string)

	for _, pg := range pl.pages {
		if pg == nil {
			continue
		}
		pageQN := pl.hierarchy.GetQualifiedName(model.ID(pg.ID()), pg.Name())
		for _, mrQN := range pg.AllowedRolesQualifiedNames() {
			pageGrants[mrQN] = append(pageGrants[mrQN], pageQN)
		}
	}

	for _, mf := range ml.mfs {
		if mf == nil {
			continue
		}
		mfQN := ml.hierarchy.GetQualifiedName(model.ID(mf.ID()), mf.Name())
		for _, mrQN := range mf.AllowedModuleRolesQualifiedNames() {
			mfGrants[mrQN] = append(mfGrants[mrQN], mfQN)
		}
	}
	return pageGrants, mfGrants, nil
}

// mfMeta holds per-MF metadata needed for access analysis.
type mfMeta struct {
	applyEntityAccess bool
	entityQNs         []string // entities retrieved/changed/created/deleted by this MF
}

// buildMFMeta returns mfQN → mfMeta.
// It uses the pre-loaded ml to avoid a redundant ListMicroflowsGen call (ACC-001).
func buildMFMeta(_ *ExecContext, ml *mfLoad) (map[string]mfMeta, error) {
	result := make(map[string]mfMeta)
	for _, mf := range ml.mfs {
		if mf == nil {
			continue
		}
		mfQN := ml.hierarchy.GetQualifiedName(model.ID(mf.ID()), mf.Name())
		m := mfMeta{
			applyEntityAccess: mf.ApplyEntityAccess(),
		}
		if m.applyEntityAccess {
			m.entityQNs = collectMFEntityRefs(mf)
		}
		result[mfQN] = m
	}
	return result, nil
}

// collectMFEntityRefs walks a Microflow's object collection and collects the
// qualified entity names touched by retrieve/change/create/delete actions.
//
// TODO(Task 5): implement via ObjectsItems() traversal. Returning nil for now
// means ApplyEntityAccess entity gaps inside microflow bodies are not reported;
// page-level and execute-grant gaps are unaffected.
func collectMFEntityRefs(mf *genMF.Microflow) []string {
	_ = mf
	return nil
}

// checkPageGaps returns all access gaps for a single (moduleRole, page) pair (ACC-002).
// It checks entity read grants for widget datasources and execute grants for
// microflows called directly from the page or from MFs with ApplyEntityAccess=ON.
func checkPageGaps(
	userRole, mrQN, pageQN string,
	pm *types.PageModel,
	entityGrants map[string]map[string]entityAccessSummary,
	mfGrants map[string][]string,
	mfMetaMap map[string]mfMeta,
) []AccessGap {
	var gaps []AccessGap
	refs := collectWidgetRefs(pm, mfMetaMap)

	// Entity read grants.
	for _, entityQN := range refs.entityQNs {
		if isSystemEntity(entityQN) {
			continue
		}
		if !entityGrants[mrQN][entityQN].canRead {
			gaps = append(gaps, AccessGap{
				UserRole:   userRole,
				ModuleRole: mrQN,
				Path:       fmt.Sprintf("page %s → entity %s", pageQN, entityQN),
				EntityQN:   entityQN,
				GapType:    GapEntityRead,
			})
		}
	}

	// MF execute grants (direct MFs called from page).
	for _, mfQN := range refs.directMFQNs {
		if !hasMFGrant(mfGrants, mrQN, mfQN) {
			gaps = append(gaps, AccessGap{
				UserRole:   userRole,
				ModuleRole: mrQN,
				Path:       fmt.Sprintf("page %s → microflow %s", pageQN, mfQN),
				MFQN:       mfQN,
				GapType:    GapMFExecute,
			})
		}
		// If MF has ApplyEntityAccess=ON, check entity grants inside MF body.
		if meta, ok := mfMetaMap[mfQN]; ok && meta.applyEntityAccess {
			for _, entityQN := range meta.entityQNs {
				if isSystemEntity(entityQN) {
					continue
				}
				if !entityGrants[mrQN][entityQN].canRead {
					gaps = append(gaps, AccessGap{
						UserRole:   userRole,
						ModuleRole: mrQN,
						Path:       fmt.Sprintf("page %s → microflow %s (ApplyEntityAccess) → entity %s", pageQN, mfQN, entityQN),
						EntityQN:   entityQN,
						GapType:    GapEntityRead,
					})
				}
			}
		}
	}
	return gaps
}

// checkMFGaps returns all access gaps for a single (moduleRole, microflow) pair (ACC-002).
// Only checks entity grants when the microflow has ApplyEntityAccess=ON.
func checkMFGaps(
	userRole, mrQN, mfQN string,
	meta mfMeta,
	entityGrants map[string]map[string]entityAccessSummary,
) []AccessGap {
	if !meta.applyEntityAccess {
		return nil
	}
	var gaps []AccessGap
	for _, entityQN := range meta.entityQNs {
		if isSystemEntity(entityQN) {
			continue
		}
		if !entityGrants[mrQN][entityQN].canRead {
			gaps = append(gaps, AccessGap{
				UserRole:   userRole,
				ModuleRole: mrQN,
				Path:       fmt.Sprintf("microflow %s (ApplyEntityAccess) → entity %s", mfQN, entityQN),
				EntityQN:   entityQN,
				GapType:    GapEntityRead,
			})
		}
	}
	return gaps
}

// detectGaps compares access grants against widget/MF references and returns gaps.
// It delegates per-page and per-MF checks to checkPageGaps and checkMFGaps (ACC-002).
func detectGaps(
	urToMR map[string][]string,
	entityGrants map[string]map[string]entityAccessSummary,
	pageGrants map[string][]string,
	mfGrants map[string][]string,
	mfMetaMap map[string]mfMeta,
	pageModels map[string]*types.PageModel,
) []AccessGap {
	var gaps []AccessGap
	seen := make(map[string]bool) // deduplicate by "role+entity+mf+gapType"

	addGaps := func(newGaps []AccessGap) {
		for _, g := range newGaps {
			key := g.ModuleRole + "|" + g.EntityQN + "|" + g.MFQN + "|" + string(g.GapType)
			if !seen[key] {
				seen[key] = true
				gaps = append(gaps, g)
			}
		}
	}

	for userRole, moduleRoles := range urToMR {
		for _, mrQN := range moduleRoles {

			// --- Entry point 1: pages ---
			for _, pageQN := range pageGrants[mrQN] {
				pm := pageModels[pageQN]
				if pm == nil {
					continue
				}
				addGaps(checkPageGaps(userRole, mrQN, pageQN, pm, entityGrants, mfGrants, mfMetaMap))
			}

			// --- Entry point 2: execute-granted microflows ---
			for _, mfQN := range mfGrants[mrQN] {
				meta, ok := mfMetaMap[mfQN]
				if !ok {
					continue
				}
				addGaps(checkMFGaps(userRole, mrQN, mfQN, meta, entityGrants))
			}
		}
	}
	return gaps
}

// isSystemEntity returns true for entities in modules that are always accessible.
func isSystemEntity(qn string) bool {
	return strings.HasPrefix(qn, "System.") || strings.HasPrefix(qn, "Administration.")
}

// hasMFGrant checks if moduleRole has an execute grant on mfQN.
func hasMFGrant(mfGrants map[string][]string, mrQN, mfQN string) bool {
	for _, grantedQN := range mfGrants[mrQN] {
		if grantedQN == mfQN {
			return true
		}
	}
	return false
}

// buildPageModels reads every Page from the MPR and returns pageQN → *types.PageModel.
// It uses the pre-loaded pl to avoid a redundant ListPagesGen call (ACC-001).
// Pages that cannot be parsed are skipped; their errors are collected and
// returned as warnings so the caller can surface them to the user (ACC-003).
func buildPageModels(ctx *ExecContext, pl *pageLoad) (map[string]*types.PageModel, []string, error) {
	result := make(map[string]*types.PageModel)
	var warnings []string
	for _, pg := range pl.pages {
		if pg == nil {
			continue
		}
		pageQN := pl.hierarchy.GetQualifiedName(model.ID(pg.ID()), pg.Name())
		pm, err := ctx.PageModelAccess.GetPageModel(model.ID(pg.ID()))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("page %s: %v", pageQN, err))
			continue
		}
		result[pageQN] = pm
	}
	return result, warnings, nil
}

// widgetRefs holds entity QNs and direct MF QNs collected from a page's widget tree.
type widgetRefs struct {
	entityQNs   []string // entities used directly (datasource.Entity or page param)
	directMFQNs []string // microflows directly callable from the page (datasource or button action)
}

// collectWidgetRefs walks the PageModel widget tree and collects entity QNs and
// direct microflow QNs. "Direct" means the first microflow hop from the page.
// mfMetaMap is used to distinguish MF/NF OnClick actions from page-navigation
// actions — only qualified names present in the map are treated as MF calls (ACC-004).
func collectWidgetRefs(pm *types.PageModel, mfMetaMap map[string]mfMeta) widgetRefs {
	seenEnt := make(map[string]bool)
	seenMF := make(map[string]bool)

	// Page params are entities directly visible to the page.
	for _, p := range pm.Params {
		if p.EntityName != "" {
			seenEnt[p.EntityName] = true
		}
	}

	var walkNode func(n *types.WidgetNode)
	walkNode = func(n *types.WidgetNode) {
		if n == nil {
			return
		}
		// Datasource entity (database, parameter association).
		if n.DataSource != nil {
			if n.DataSource.Entity != "" {
				seenEnt[n.DataSource.Entity] = true
			}
			// Microflow / nanoflow datasource.
			if (n.DataSource.Kind == types.DataSourceMicroflow || n.DataSource.Kind == types.DataSourceNanoflow) &&
				n.DataSource.Reference != "" {
				seenMF[n.DataSource.Reference] = true
			}
		}
		// Entity context provided by a dataview to its children.
		if n.EntityCtx != "" {
			seenEnt[n.EntityCtx] = true
		}
		// Button / action OnClick — only treat as a microflow call when the
		// qualified name is present in mfMetaMap. This prevents page-navigation
		// actions (ShowPageAction) from being misclassified as microflow calls (ACC-004).
		if n.OnClick != "" && strings.Contains(n.OnClick, ".") {
			if _, isMF := mfMetaMap[n.OnClick]; isMF {
				seenMF[n.OnClick] = true
			}
		}

		for _, child := range n.Children {
			walkNode(child)
		}
		for _, f := range n.Footer {
			walkNode(f)
		}
		// DataGrid content widgets.
		if n.DataGrid != nil {
			for _, col := range n.DataGrid.Columns {
				for _, cw := range col.ContentWidgets {
					walkNode(cw)
				}
			}
			for _, cw := range n.DataGrid.ControlBar {
				walkNode(cw)
			}
			for _, cw := range n.DataGrid.FilterWidgets {
				walkNode(cw)
			}
		}
		// Gallery content/filter widgets.
		if n.Gallery != nil {
			for _, cw := range n.Gallery.ContentWidgets {
				walkNode(cw)
			}
			for _, cw := range n.Gallery.FilterWidgets {
				walkNode(cw)
			}
		}
	}

	for _, w := range pm.Widgets {
		walkNode(w)
	}

	refs := widgetRefs{}
	for qn := range seenEnt {
		refs.entityQNs = append(refs.entityQNs, qn)
	}
	for qn := range seenMF {
		refs.directMFQNs = append(refs.directMFQNs, qn)
	}
	return refs
}
