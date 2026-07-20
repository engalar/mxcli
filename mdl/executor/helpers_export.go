// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"

	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// Exported wrappers for unexported helpers, so the domainmodel sub-package
// can call them while keeping originals unexported within executor.

func BuildEntityFromAST(moduleName string, s *ast.CreateEntityStmt) (*genDm.Entity, error) {
	return buildEntityFromAST(moduleName, s)
}

func AstToAttributeGen(a *ast.Attribute) *genDm.Attribute { return astToAttributeGen(a) }

func AstToAttributeTypeGen(dt ast.DataType) element.Element { return astToAttributeTypeGen(dt) }

func AstToValidationRulesGen(a *ast.Attribute, entityQN string) []*genDm.ValidationRule {
	return astToValidationRulesGen(a, entityQN)
}

func AstToEventHandlerGen(eh *ast.EventHandlerDef) *genDm.EventHandler { return astToEventHandlerGen(eh) }

func AstToIndexGen(idx *ast.Index, attrNameToID map[string]model.ID) *genDm.Index {
	return astToIndexGen(idx, attrNameToID)
}

func LayoutPos(x, y int) string { return layoutPos(x, y) }

func ParseLocationBSON(loc string) (x, y int, ok bool) { return parseLocationBSON(loc) }

func FindEntityInDMGenByName(dm *genDm.DomainModel, name string) *genDm.Entity {
	return findEntityInDMGenByName(dm, name)
}

func FindAttributeGenByName(entity *genDm.Entity, name string) *genDm.Attribute {
	return findAttributeGenByName(entity, name)
}

func FindAttributeGenWithIndexByName(entity *genDm.Entity, name string) (*genDm.Attribute, int) {
	return findAttributeGenWithIndexByName(entity, name)
}

func ApplyPseudoAttributeDropGen(entity *genDm.Entity, attrName string) (bool, error) {
	return applyPseudoAttributeDropGen(entity, attrName)
}

func CleanupDroppedAttributeReferencesGen(entity *genDm.Entity, droppedID model.ID, attrQN string) int {
	return cleanupDroppedAttributeReferencesGen(entity, droppedID, attrQN)
}

func LowerASCII(s string) string { return lowerASCII(s) }

func StringTitleLowerASCII(s string) string { return stringTitleLowerASCII(s) }

func AstToAssociationGen(s *ast.CreateAssociationStmt, fromID, toID element.ID) *genDm.Association {
	return astToAssociationGen(s, fromID, toID)
}

func AstAssociationTypeStringGen(s *ast.CreateAssociationStmt) string {
	return astAssociationTypeStringGen(s)
}

func AstAssociationOwnerStringGen(s *ast.CreateAssociationStmt) string {
	return astAssociationOwnerStringGen(s)
}

func AstAssociationStorageStringGen(s *ast.CreateAssociationStmt) string {
	return astAssociationStorageStringGen(s)
}

func AstAssociationDeleteBehaviorGen(s *ast.CreateAssociationStmt) element.Element {
	return astAssociationDeleteBehaviorGen(s)
}

func AssociationDocumentation(s *ast.CreateAssociationStmt) string {
	return associationDocumentation(s)
}

func NewAssociationDeleteBehaviorGen(db ast.DeleteBehavior) *genDm.AssociationDeleteBehavior {
	return newAssociationDeleteBehaviorGen(db)
}

func AstToViewEntityGen(s *ast.CreateViewEntityStmt, sourceDocRef string, location model.Point) *genDm.Entity {
	return astToViewEntityGen(s, sourceDocRef, location)
}

func PreserveViewEntityIDs(entity, existing *genDm.Entity) {
	preserveViewEntityIDs(entity, existing)
}

func AutoLayoutLocationGen(pos *ast.Position, existing *genDm.Entity, dm *genDm.DomainModel) model.Point {
	return autoLayoutLocationGen(pos, existing, dm)
}

// ExecContext-based lookup wrappers for domainmodel handler.go.
func FindModuleWrap(ctx *ExecContext, name string) (*model.Module, error) {
	return findModule(ctx, name)
}

func FindOrCreateModuleWrap(ctx *ExecContext, name string) (*model.Module, error) {
	return findOrCreateModule(ctx, name)
}

func FindEntityGenWrap(ctx *ExecContext, qn ast.QualifiedName) (*genDm.Entity, string, error) {
	return findEntityGen(ctx, qn)
}

func FindEnumerationWrap(ctx *ExecContext, moduleName, enumName string) *model.Enumeration {
	return findEnumeration(ctx, moduleName, enumName)
}

func CheckFeatureWrap(ctx *ExecContext, area, name, statement, hint string) error {
	return checkFeature(ctx, area, name, statement, hint)
}

func WarnEntityReferencesWrap(ctx *ExecContext, entityQN string) {
	warnEntityReferences(ctx, entityQN)
}

func WarnMicroflowEntityParamRefsWrap(ctx *ExecContext, entityQN string) {
	warnMicroflowEntityParamRefs(ctx, entityQN)
}

func TrackModifiedDomainModelWrap(ctx *ExecContext, moduleID model.ID, moduleName string) {
	ctx.trackModifiedDomainModel(moduleID, moduleName)
}

func TrackCreatedEntityWrap(ctx *ExecContext, moduleName, entityName string, entityID model.ID) {
	ctx.trackCreatedEntity(moduleName, entityName, entityID)
}

func InvalidateHierarchyWrap(ctx *ExecContext) {
	invalidateHierarchy(ctx)
}

// ─────────────────────────────────────────────────────────────
// Page/snippet/layout wrapper helpers for page subpackage.
// ─────────────────────────────────────────────────────────────

func InvalidatePagesGenCacheWrap(ctx *ExecContext) {
	invalidatePagesGenCache(ctx)
}

func TrackCreatedPageWrap(ctx *ExecContext, moduleName, pageName string, pageID, moduleID model.ID) {
	ctx.trackCreatedPage(moduleName, pageName, pageID, moduleID)
}

func TrackCreatedSnippetWrap(ctx *ExecContext, moduleName, snippetName string, snippetID, moduleID model.ID) {
	ctx.trackCreatedSnippet(moduleName, snippetName, snippetID, moduleID)
}

func DefaultDocumentAccessRoleQNamesWrap(ctx *ExecContext, module *model.Module) []string {
	return defaultDocumentAccessRoleQNames(ctx, module)
}

func BuildLayoutContentWrap(ctx *ExecContext, s *ast.CreateLayoutStmt) element.Element {
	return buildLayoutContent(s)
}

func PageASTToModelWrap(ctx *ExecContext, s *ast.CreatePageStmtV3, moduleName string) (*types.PageModel, error) {
	return pageASTToModel(s, moduleName)
}

func PageModelHasLossyWidgetWrap(pm *types.PageModel) bool {
	return pageModelHasLossyWidget(pm)
}

// ─────────────────────────────────────────────────────────────
// Security wrapper helpers for security subpackage.
// ─────────────────────────────────────────────────────────────

func GetProjectSecurityGenWrap(ctx *ExecContext) (*genSec.ProjectSecurity, error) {
	return getProjectSecurityGen(ctx)
}

func InvalidateProjectSecurityCacheWrap(ctx *ExecContext) {
	invalidateProjectSecurityCache(ctx)
}

func InvalidateModuleSecurityCacheWrap(ctx *ExecContext) {
	invalidateModuleSecurityCache(ctx)
}

func GetDomainModelGenCachedWrap(ctx *ExecContext, moduleID model.ID) (*genDm.DomainModel, error) {
	return getDomainModelGenCached(ctx, moduleID)
}

func InvalidateDomainModelGenForModuleWrap(ctx *ExecContext, moduleID model.ID) {
	invalidateDomainModelGenForModule(ctx, moduleID)
}

func InvalidateDomainModelsCacheWrap(ctx *ExecContext) {
	invalidateDomainModelsCache(ctx)
}

func GetModulesFromCacheWrap(ctx *ExecContext) ([]*model.Module, error) {
	return getModulesFromCache(ctx)
}

func FormatAccessRuleResultWrap(ctx *ExecContext, moduleName, entityName string, roleNames []string) string {
	return formatAccessRuleResult(ctx, moduleName, entityName, roleNames)
}

func DetectUserEntityGenWrap(ctx *ExecContext) (string, error) {
	return detectUserEntityGen(ctx)
}

func CachedDomainModelsGenWrap(ctx *ExecContext) ([]*genDm.DomainModel, error) {
	return cachedDomainModelsGen(ctx)
}

func EntityGeneralizationQNWrap(entity *genDm.Entity) string {
	return entityGeneralizationQNGen(entity)
}

// ─────────────────────────────────────────────────────────────
// Additional security wrapper helpers for security subpackage.
// ─────────────────────────────────────────────────────────────

func ValidateModuleRoleWrap(ctx *ExecContext, role ast.QualifiedName) (bool, error) {
	return validateModuleRole(ctx, role)
}

func CascadeRemoveRoleFromMicroflowsWrap(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	return cascadeRemoveRoleFromMicroflowsGen(ctx, moduleID, qualifiedRole)
}

func CascadeRemoveRoleFromNanoflowsWrap(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	return cascadeRemoveRoleFromNanoflowsGen(ctx, moduleID, qualifiedRole)
}

func PruneInvalidUserRolesWrap(ctx *ExecContext, _ *model.ID) error {
	return pruneInvalidUserRoles(ctx, nil)
}

func LookupCreatedPageIDWrap(ctx *ExecContext, qualifiedName string) (model.ID, error) {
	return lookupCreatedPageID(ctx, qualifiedName)
}

func FilterAutoDocumentRolesWrap(ctx *ExecContext, roles []string) []string {
	return filterAutoDocumentRoles(roles)
}

// MergeAllowedRolesStatic is the exported version of mergeAllowedRoles.
func MergeAllowedRolesStatic(existing []string, valid []ast.QualifiedName) ([]string, []string) {
	return mergeAllowedRoles(existing, valid)
}

// FilterAllowedRolesStatic is the exported version of filterAllowedRoles.
func FilterAllowedRolesStatic(existing []string, roles []ast.QualifiedName) ([]string, []string) {
	return filterAllowedRoles(existing, roles)
}
