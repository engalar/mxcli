// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// unquoteIdentifier strips surrounding double-quotes or backticks from a quoted identifier.
func unquoteIdentifier(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// unquoteQualifiedName strips quotes from each segment of a dotted qualified name.
func unquoteQualifiedName(s string) string {
	parts := strings.Split(s, ".")
	for i, p := range parts {
		parts[i] = unquoteIdentifier(p)
	}
	return strings.Join(parts, ".")
}

// newAttributeRef creates a DomainModels$AttributeRef element for a fully qualified
// attribute name (e.g. "Module.Entity.Attribute"). This is the modern Mendix format
// for binding form input widgets to entity attributes; the legacy AttributePath string
// field is left nil so both old and new readers see a consistent state.
func newAttributeRef(qualifiedAttr string) element.Element {
	ar := genDm.NewAttributeRef()
	ar.SetAttributeQualifiedName(qualifiedAttr)
	return ar
}

// resolveAttributePath resolves a short attribute name to a fully qualified name
// using the current entity context. If the attribute already has dots or no entity
// context is available, the attribute is returned as-is.
func (pb *pageBuilder) resolveAttributePath(attr string) string {
	if attr == "" {
		return ""
	}
	// If the attribute already contains a dot, it's already qualified
	if strings.Contains(attr, ".") {
		return attr
	}
	// If we have an entity context, prefix the attribute with it
	if pb.entityContext != "" {
		return pb.entityContext + "." + attr
	}
	return attr
}

// resolveAssociationPath resolves a short association name to a fully qualified name.
// Associations are module-level objects, so the path is Module.AssociationName (2-part).
// If the name already contains a dot, it's returned as-is.
func (pb *pageBuilder) resolveAssociationPath(assocName string) string {
	if assocName == "" {
		return ""
	}
	// If already qualified (contains a dot), return as-is
	if strings.Contains(assocName, ".") {
		return assocName
	}
	// Extract module name from entity context (e.g., "PgTest.Order" → "PgTest")
	if pb.entityContext != "" {
		parts := strings.SplitN(pb.entityContext, ".", 2)
		if len(parts) >= 1 {
			return parts[0] + "." + assocName
		}
	}
	return assocName
}

// resolveSnippetRef resolves a snippet qualified name to its ID.
func (pb *pageBuilder) resolveSnippetRef(snippetRef string) (model.ID, error) {
	if snippetRef == "" {
		return "", mdlerrors.NewValidation("empty snippet reference")
	}

	snippetRef = unquoteQualifiedName(snippetRef)
	parts := strings.Split(snippetRef, ".")
	var moduleName, snippetName string
	if len(parts) >= 2 {
		moduleName = parts[0]
		snippetName = parts[len(parts)-1]
	} else {
		snippetName = snippetRef
	}

	// First, check if the snippet was created during this session
	// (not yet visible via reader)
	if pb.execCache != nil && pb.execCache.createdSnippets != nil {
		if info, ok := pb.execCache.createdSnippets[snippetRef]; ok {
			return info.ID, nil
		}
		if moduleName != "" {
			if info, ok := pb.execCache.createdSnippets[moduleName+"."+snippetName]; ok {
				return info.ID, nil
			}
		}
	}

	// Gen-typed path via snippetsRepo (preferred when available).
	if pb.snippetsRepo != nil {
		s, err := pb.snippetsRepo.FindByQualifiedName(snippetRef)
		if err != nil {
			return "", err
		}
		if s != nil {
			return model.ID(s.ID()), nil
		}
		return "", mdlerrors.NewNotFound("snippet", snippetRef)
	}

	// Fallback: gen-typed backend listing without per-call container resolution.
	// (snippetsRepo is the preferred path; this branch covers the no-repo case.)
	snippets, err := pb.backend.ListSnippetsGen()
	if err != nil {
		return "", err
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, s := range snippets {
		if s == nil {
			continue
		}
		containerID, err := pb.backend.GetPageContainerUUID(model.ID(s.ID()))
		if err != nil {
			continue
		}
		modID := h.FindModuleID(containerID)
		modName := h.GetModuleName(modID)
		if s.Name() == snippetName && (moduleName == "" || modName == moduleName) {
			return model.ID(s.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("snippet", snippetRef)
}

func (pb *pageBuilder) resolveMicroflow(qualifiedName string) (model.ID, error) {
	qualifiedName = unquoteQualifiedName(qualifiedName)
	// Parse qualified name
	parts := strings.Split(qualifiedName, ".")
	if len(parts) < 2 {
		return "", mdlerrors.NewValidationf("invalid microflow name: %s", qualifiedName)
	}
	moduleName := parts[0]
	mfName := strings.Join(parts[1:], ".")

	// First, check if the microflow was created during this session
	// (not yet visible via reader)
	if pb.execCache != nil && pb.execCache.createdMicroflows != nil {
		if info, ok := pb.execCache.createdMicroflows[qualifiedName]; ok {
			return info.ID, nil
		}
	}

	// Get microflows from backend
	mfs, err := pb.getMicroflows()
	if err != nil {
		return "", mdlerrors.NewBackend("list microflows", err)
	}

	// Use hierarchy to resolve module names (handles microflows in folders)
	h, err := pb.getHierarchy()
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	// Find matching microflow. Stage 3.2.6.5: gen Microflow exposes
	// container via the repo's GetContainerUUID lookup since the gen
	// element doesn't carry container linkage post-roundtrip.
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, _ := pb.microflowsRepo.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == moduleName && mf.Name() == mfName {
			return model.ID(mf.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("microflow", qualifiedName)
}

func (pb *pageBuilder) resolvePageRef(pageRef string) (model.ID, error) {
	if pageRef == "" {
		return "", mdlerrors.NewValidation("empty page reference")
	}

	pageRef = unquoteQualifiedName(pageRef)
	parts := strings.Split(pageRef, ".")
	var moduleName, pageName string
	if len(parts) >= 2 {
		moduleName = parts[0]
		pageName = parts[len(parts)-1]
	} else {
		pageName = pageRef
	}

	// First, check if the page was created during this session
	// (not yet visible via reader)
	if pb.execCache != nil && pb.execCache.createdPages != nil {
		if info, ok := pb.execCache.createdPages[pageRef]; ok {
			return info.ID, nil
		}
		// Also check with module prefix if not found
		if moduleName != "" {
			if info, ok := pb.execCache.createdPages[moduleName+"."+pageName]; ok {
				return info.ID, nil
			}
		}
	}

	pgs, err := pb.getPages()
	if err != nil {
		return "", err
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, p := range pgs {
		containerID, _ := pb.backend.GetPageContainerUUID(model.ID(p.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if p.Name() == pageName && (moduleName == "" || modName == moduleName) {
			return model.ID(p.ID()), nil
		}
	}

	return "", mdlerrors.NewNotFound("page", pageRef)
}
