# Domainmodel CRUD Move Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move entity and association CRUD "real implementations" from `mdl/executor/cmd_*.go` into `mdl/executor/domainmodel/` subpackage, following the same pattern as `enumerations_create.go`.

**Architecture:** Create `domainmodel/entities.go` and `domainmodel/associations.go` with DomainModelDeps-based functions. Export needed AST→gen helper functions from the executor package so domainmodel can call them. Update `domainmodel/handler.go` to call the new local implementations.

**Key constraint:** The old ExecContext-based functions in `cmd_*.go` must remain in place because `handler_deps.go` (in the `executor` package) still registers them as backward-compatible fallback handlers. The domainmodel package functions serve as the "override" registrations from `domainmodel/handler.go`.

## Global Constraints

- All new domainmodel functions take `(ctx context.Context, s *ast.Stmt, d DomainModelDeps)` — no ExecContext
- Replace `findModule(ctx, name)` with `d.FindModule(name)`
- Replace `findOrCreateModule(ctx, name)` with `d.FindOrCreateModule(name)`
- Replace `getDomainModelGenCached(ctx, id)` with `d.GetDomainModelGenCached(id)`
- Replace `setDomainModelGenCached(ctx, id, dm)` with `d.SetDomainModelGenCached(id, dm)`
- Replace `invalidateDomainModelGenForModule(ctx, id)` with `d.InvalidateDomainModelGenCache(id)`
- Replace `invalidateDomainModelsCache(ctx)` with `d.InvalidateDomainModelsCache()`
- Replace `invalidateHierarchy(ctx)` with `d.InvalidateHierarchy()`
- Replace `findEntityGen(ctx, qn)` with `d.FindEntityGen(qn)`
- Replace `entityPersistableGen(entity)` with `d.EntityPersistable(entity)`
- Replace `checkFeature(ctx, ...)` with `d.CheckFeature(...)`
- Replace `warnEntityReferences(ctx, qn)` with `d.WarnEntityReferences(qn)`
- Replace `warnMicroflowEntityParamRefs(ctx, qn)` with `d.WarnMicroflowEntityParamRefs(qn)`
- Replace `ctx.trackModifiedDomainModel(id, name)` with `d.TrackModifiedDomainModel(id, name)`
- Replace `ctx.Output` with `d.Output`
- Replace `ctx.DomainModelWriter` with `d.DomainModelWriter`
- Replace `ctx.SecurityEntityAccessManager` with `d.SecurityEntityAccessManager`
- Replace `ctx.ModuleLister` with `d.ModuleLister`
- Replace `findEnumeration(ctx, ...)` with `d.FindEnumeration(...)`

---
