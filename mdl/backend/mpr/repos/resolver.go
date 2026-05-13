// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type sqlResolver struct {
	r *mmpr.Reader
}

func NewQualifiedNameResolver(w *mmpr.Writer) repos.QualifiedNameResolver {
	return &sqlResolver{r: w.ConcreteReader()}
}

// ModuleNameByID returns the module's display name for the given module
// unit ID. Backed by mmpr.Reader.GetModule which scans
// "Projects$ModuleImpl" units.
func (s *sqlResolver) ModuleNameByID(id model.ID) (string, error) {
	m, err := s.r.GetModule(string(id))
	if err != nil {
		return "", err
	}
	return m.Name, nil
}

// resolveKindEntry maps a "kind" string (microflow / page / entity / …)
// to the BSON $Type prefix used by Mendix and reused by the repo
// FindByQualifiedName paths. The order in resolveKinds drives the
// probe order in ResolveQualifiedName — earlier entries win when a
// project happens to contain same-named units in different kinds
// (extremely rare in practice, but deterministic here).
type resolveKindEntry struct {
	kind     string // returned to callers
	typeName string // BSON $Type used by ListUnitsByType (strict-equality filtered)
}

// resolveKinds is the canonical kind-probe order for kinds that ship
// as their own top-level units. New kinds can be appended freely; the
// strict ref.Type == typeName filter avoids the known prefix-matching
// pitfalls (Forms$Page is a prefix of Forms$PageTemplate;
// Microflows$Microflow vs Microflows$Rule etc).
//
// "entity" is intentionally NOT in this list: entities are nested
// inside DomainModels$DomainModel units, so they need a different
// walk (see lookupEntity below).
var resolveKinds = []resolveKindEntry{
	{"microflow", "Microflows$Microflow"},
	{"nanoflow", "Microflows$Nanoflow"},
	{"rule", "Microflows$Rule"},
	{"page", "Forms$Page"},
	{"snippet", "Forms$Snippet"},
	{"layout", "Forms$Layout"},
	{"pagetemplate", "Forms$PageTemplate"},
	{"enumeration", "Enumerations$Enumeration"},
}

// ResolveQualifiedName parses "Module.ElementName" and probes each
// known kind in resolveKinds order until a unit whose containing
// module + Name match is found. Returns (unitID, kind, nil) on hit;
// ("", "", error) if the QN is malformed or no kind matches.
func (s *sqlResolver) ResolveQualifiedName(qn string) (model.ID, string, error) {
	moduleName, simpleName, ok := splitQN(qn)
	if !ok {
		return "", "", fmt.Errorf("ResolveQualifiedName: invalid qualified name %q (want Module.Name)", qn)
	}
	mods, err := s.r.ListModules()
	if err != nil {
		return "", "", fmt.Errorf("ResolveQualifiedName: list modules: %w", err)
	}
	moduleMap := make(map[string]string, len(mods))
	for _, m := range mods {
		moduleMap[m.ID] = m.Name
	}
	parents, err := s.r.BuildContainerParent()
	if err != nil {
		return "", "", fmt.Errorf("ResolveQualifiedName: build container parents: %w", err)
	}

	for _, k := range resolveKinds {
		id, found, err := s.lookupInKind(k.typeName, moduleName, simpleName, moduleMap, parents)
		if err != nil {
			return "", "", err
		}
		if found {
			return id, k.kind, nil
		}
	}
	// Entities are nested inside DomainModels$DomainModel units, not
	// addressable as their own top-level units — handled separately.
	if id, found, err := s.lookupEntity(moduleName, simpleName, moduleMap, parents); err != nil {
		return "", "", err
	} else if found {
		return id, "entity", nil
	}
	return "", "", fmt.Errorf("ResolveQualifiedName: %q not found", qn)
}

// lookupEntity walks DomainModels$DomainModel units in the named
// module, scans their Entities array, and returns the entity ID +
// found-flag for the first match. The returned ID is the entity's
// own $ID (suitable for cross-references), not the parent
// DomainModel's unit ID.
func (s *sqlResolver) lookupEntity(
	moduleName, simpleName string,
	moduleMap map[string]string,
	parents map[string]string,
) (model.ID, bool, error) {
	const dmType = "DomainModels$DomainModel"
	refs, err := s.r.ListUnitsByType(dmType)
	if err != nil {
		return "", false, fmt.Errorf("ResolveQualifiedName: list %s: %w", dmType, err)
	}
	for _, ref := range refs {
		if ref.Type != dmType {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		bts, err := s.r.GetRawUnitBytes(ref.ID)
		if err != nil || len(bts) == 0 {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(bts, &raw); err != nil {
			continue
		}
		entities, _ := raw["Entities"].(bson.A)
		// Mendix versioned-array convention: index 0 is the int32 version
		// prefix, real entries start at index 1.
		for i, e := range entities {
			if i == 0 {
				continue
			}
			doc, ok := e.(bson.M)
			if !ok {
				if d, dok := e.(map[string]any); dok {
					doc = bson.M(d)
				} else {
					continue
				}
			}
			name, _ := doc["Name"].(string)
			if name != simpleName {
				continue
			}
			id := entityIDFromDoc(doc)
			if id == "" {
				continue
			}
			return model.ID(id), true, nil
		}
	}
	return "", false, nil
}

// entityIDFromDoc extracts the entity's $ID, accepting either Mendix
// binary-UUID (primitive.Binary) or string forms.
func entityIDFromDoc(doc bson.M) string {
	v, ok := doc["$ID"]
	if !ok {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case primitive.Binary:
		return mmpr.BlobToUUID(x.Data)
	case []byte:
		return mmpr.BlobToUUID(x)
	}
	return ""
}

// lookupInKind scans all units of typeName whose containing module
// resolves to moduleName, and returns the first one whose decoded
// "Name" field equals simpleName. Filters strictly by ref.Type ==
// typeName because mmpr.ListUnitsByType is prefix-matched.
func (s *sqlResolver) lookupInKind(
	typeName, moduleName, simpleName string,
	moduleMap map[string]string,
	parents map[string]string,
) (model.ID, bool, error) {
	refs, err := s.r.ListUnitsByType(typeName)
	if err != nil {
		return "", false, fmt.Errorf("ResolveQualifiedName: list %s: %w", typeName, err)
	}
	for _, ref := range refs {
		if ref.Type != typeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		bts, err := s.r.GetRawUnitBytes(ref.ID)
		if err != nil || len(bts) == 0 {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(bts, &raw); err != nil {
			continue
		}
		if name, _ := raw["Name"].(string); name == simpleName {
			return model.ID(ref.ID), true, nil
		}
	}
	return "", false, nil
}
