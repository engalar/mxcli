// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReconcileChange records a single member-access change made during
// reconciliation.  Action is either "added" (new entry written) or "stripped"
// (stale entry removed, e.g. a CrossAssociation reference that Studio Pro
// rejects with CE0066).  The String() format is the per-entity log line
// printed by UPDATE SECURITY.
type ReconcileChange struct {
	Entity string // entity short name
	Member string // association or attribute short name
	Action string // "added" | "stripped"
}

func (c ReconcileChange) String() string {
	if c.Action == "stripped" {
		return fmt.Sprintf("%s: stripped stale MemberAccess for %s", c.Entity, c.Member)
	}
	return fmt.Sprintf("%s: added MemberAccess for %s", c.Entity, c.Member)
}

// PatchReconcileMemberAccesses reconciles entity member accesses in a raw
// DomainModel BSON document for the given module. Pure BSON manipulation —
// no database access required.
// Returns patched bytes, a list of every MemberAccess entry that was added,
// and error.  A non-empty changes list means the caller must write back the
// patched bytes.
func PatchReconcileMemberAccesses(rawBytes []byte, moduleName string) ([]byte, []ReconcileChange, error) {
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		return nil, nil, fmt.Errorf("unmarshal: %w", err)
	}
	doc, changes := secPatchReconcileMemberAccessesDoc(doc, moduleName)
	out, err := bson.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal: %w", err)
	}
	return out, changes, nil
}

// secPatchReconcileMemberAccessesDoc performs reconciliation on a parsed BSON
// DomainModel document: ensures each entity's access rules have correct
// MemberAccess entries for all current attributes and associations.
func secPatchReconcileMemberAccessesDoc(doc bson.D, moduleName string) (bson.D, []ReconcileChange) {
	var changes []ReconcileChange

	entitiesArr := secGetBsonArray(doc, "Entities")
	if entitiesArr == nil {
		return doc, nil
	}

	assocNames := map[string]bool{}
	assocArr := secGetBsonArray(doc, "Associations")
	for _, item := range assocArr {
		assocDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		for _, f := range assocDoc {
			if f.Key == "Name" {
				if name, ok := f.Value.(string); ok {
					assocNames[name] = true
				}
				break
			}
		}
	}
	crossArr := secGetBsonArray(doc, "CrossAssociations")
	for _, item := range crossArr {
		crossDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		for _, f := range crossDoc {
			if f.Key == "Name" {
				if name, ok := f.Value.(string); ok {
					assocNames[name] = true
				}
				break
			}
		}
	}

	for i, item := range entitiesArr {
		entityDoc, ok := item.(bson.D)
		if !ok {
			continue
		}

		entityName := ""
		for _, f := range entityDoc {
			if f.Key == "Name" {
				entityName = secBsonString(f.Value)
				break
			}
		}
		if entityName == "" {
			continue
		}

		attrNames := map[string]bool{}
		calculatedAttrs := map[string]bool{}
		attrsArr := secGetBsonArray(entityDoc, "Attributes")
		for _, attrItem := range attrsArr {
			attrDoc, ok := attrItem.(bson.D)
			if !ok {
				continue
			}
			attrName := ""
			isCalculated := false
			for _, f := range attrDoc {
				if f.Key == "Name" {
					attrName = secBsonString(f.Value)
				}
				if f.Key == "Value" {
					if valueDoc, ok := f.Value.(bson.D); ok {
						for _, vf := range valueDoc {
							if vf.Key == "$Type" {
								if vt, ok := vf.Value.(string); ok && vt == "DomainModels$CalculatedValue" {
									isCalculated = true
								}
							}
						}
					}
				}
			}
			if attrName != "" {
				attrNames[attrName] = true
				if isCalculated {
					calculatedAttrs[attrName] = true
				}
			}
		}

		entityID := ""
		for _, f := range entityDoc {
			if f.Key == "$ID" {
				entityID = secExtractBsonIDValue(f.Value)
				break
			}
		}
		entityAssocNames := map[string]bool{}

		systemAssocRefs := map[string]bool{}
		for _, f := range entityDoc {
			if f.Key == "Generalization" || f.Key == "MaybeGeneralization" {
				if genDoc, ok := f.Value.(bson.D); ok {
					for _, gf := range genDoc {
						if gf.Key == "$Type" {
							if gt, ok := gf.Value.(string); ok && gt == "DomainModels$NoGeneralization" {
								for _, ngf := range genDoc {
									switch ngf.Key {
									case "HasOwner":
										if v, ok := ngf.Value.(bool); ok && v {
											systemAssocRefs["System.owner"] = true
										}
									case "HasChangedBy":
										if v, ok := ngf.Value.(bool); ok && v {
											systemAssocRefs["System.changedBy"] = true
										}
									}
								}
							}
						}
					}
				}
				break
			}
		}
		for _, aItem := range assocArr {
			aDoc, ok := aItem.(bson.D)
			if !ok {
				continue
			}
			aParentID := ""
			aName := ""
			for _, f := range aDoc {
				switch f.Key {
				case "ParentPointer":
					aParentID = secExtractBsonIDValue(f.Value)
				case "Name":
					aName = secBsonString(f.Value)
				}
			}
			if aParentID == entityID && aName != "" {
				entityAssocNames[aName] = true
			}
		}
		for _, caItem := range crossArr {
			caDoc, ok := caItem.(bson.D)
			if !ok {
				continue
			}
			parentID := ""
			caName := ""
			for _, f := range caDoc {
				if f.Key == "ParentPointer" {
					parentID = secExtractBsonIDValue(f.Value)
				}
				if f.Key == "Name" {
					caName = secBsonString(f.Value)
				}
			}
			if parentID == entityID && caName != "" {
				entityAssocNames[caName] = true
			}
		}

		for j, f := range entityDoc {
			if f.Key != "AccessRules" {
				continue
			}
			rulesArr, ok := f.Value.(bson.A)
			if !ok {
				break
			}

			for k, ruleItem := range rulesArr {
				ruleDoc, ok := ruleItem.(bson.D)
				if !ok {
					continue
				}

				ruleDoc, stripped := secStripInvalidAccessRuleProps(ruleDoc)
				if stripped {
					rulesArr[k] = ruleDoc
					// stripped is counted implicitly via changed below
				}

				for m, rf := range ruleDoc {
					if rf.Key != "MemberAccesses" {
						continue
					}
					maArr, ok := rf.Value.(bson.A)
					if !ok {
						break
					}
					// NOTE: do NOT skip when len(maArr) <= 1 (only version prefix).
					// Entities with empty MemberAccesses still need association
					// entries added — the previous guard caused silent CE0066.

					defaultRights := "ReadWrite"
					for _, drf := range ruleDoc {
						if drf.Key == "DefaultMemberAccessRights" {
							if dr, ok := drf.Value.(string); ok {
								defaultRights = dr
							}
							break
						}
					}

					coveredAttrs := map[string]bool{}
					coveredAssocs := map[string]bool{}
					changed := stripped // propagate stripped-props change
					// Preserve version prefix; fall back to int32(3) if absent.
					var filtered bson.A
					if len(maArr) > 0 {
						filtered = bson.A{maArr[0]}
					} else {
						filtered = bson.A{int32(3)}
					}

					coveredSystemAssocs := map[string]bool{}
					for _, maItem := range maArr[1:] {
						maDoc, ok := maItem.(bson.D)
						if !ok {
							continue
						}
						attrRef := ""
						assocRef := ""
						for _, mf := range maDoc {
							if mf.Key == "Attribute" {
								attrRef = secBsonString(mf.Value)
							}
							if mf.Key == "Association" {
								assocRef = secBsonString(mf.Value)
							}
						}

						if attrRef != "" {
							parts := secSplitQualifiedRef(attrRef)
							if parts != "" && attrNames[parts] {
								coveredAttrs[parts] = true
								if calculatedAttrs[parts] {
									maDoc = secDowngradeCalculatedAttrRights(maDoc)
								}
								filtered = append(filtered, maDoc)
							} else {
								// Stale attribute ref (attribute was deleted or renamed).
								changes = append(changes, ReconcileChange{Entity: entityName, Member: parts, Action: "stripped"})
								changed = true
							}
						} else if assocRef != "" {
							if systemAssocRefs[assocRef] {
								coveredSystemAssocs[assocRef] = true
								filtered = append(filtered, maItem)
							} else {
								parts := secSplitAssocRef(assocRef)
								if parts != "" && entityAssocNames[parts] {
									coveredAssocs[parts] = true
									filtered = append(filtered, maItem)
								} else {
									// Stale or cross-module association ref — strip it.
									// Critically: CrossAssociation names are not in entityAssocNames,
									// so they fall here and are removed (CE0066 fix).
									changes = append(changes, ReconcileChange{
										Entity: entityName,
										Member: parts,
										Action: "stripped",
									})
									changed = true
								}
							}
						} else {
							filtered = append(filtered, maItem)
						}
					}

					for attrName := range attrNames {
						if !coveredAttrs[attrName] {
							rights := defaultRights
							if calculatedAttrs[attrName] && (rights == "ReadWrite" || rights == "WriteOnly") {
								rights = "ReadOnly"
							}
							newMA := bson.D{
								{Key: "$Type", Value: "DomainModels$MemberAccess"},
								{Key: "$ID", Value: idToBsonBinary(generateUUID())},
								{Key: "AccessRights", Value: rights},
								{Key: "Attribute", Value: moduleName + "." + entityName + "." + attrName},
							}
							filtered = append(filtered, newMA)
							changes = append(changes, ReconcileChange{Entity: entityName, Member: attrName})
							changed = true
						}
					}

					for aName := range entityAssocNames {
						if !coveredAssocs[aName] {
							newMA := bson.D{
								{Key: "$Type", Value: "DomainModels$MemberAccess"},
								{Key: "$ID", Value: idToBsonBinary(generateUUID())},
								{Key: "AccessRights", Value: defaultRights},
								{Key: "Association", Value: moduleName + "." + aName},
							}
							filtered = append(filtered, newMA)
							changes = append(changes, ReconcileChange{Entity: entityName, Member: aName})
							changed = true
						}
					}

					for sysRef := range systemAssocRefs {
						if !coveredSystemAssocs[sysRef] {
							newMA := bson.D{
								{Key: "$Type", Value: "DomainModels$MemberAccess"},
								{Key: "$ID", Value: idToBsonBinary(generateUUID())},
								{Key: "AccessRights", Value: defaultRights},
								{Key: "Association", Value: sysRef},
							}
							filtered = append(filtered, newMA)
							changes = append(changes, ReconcileChange{Entity: entityName, Member: sysRef})
							changed = true
						}
					}

					if changed {
						ruleDoc[m].Value = filtered
						rulesArr[k] = ruleDoc
					}

					break
				}
			}

			entityDoc[j].Value = rulesArr
			break
		}

		entitiesArr[i] = entityDoc
	}

	return secSetBsonField(doc, "Entities", entitiesArr), changes
}

// ── Private helpers (sec* prefix to avoid collisions) ────────────────────────

// secSetBsonField sets a top-level field in a bson.D, adding it if not found.
func secSetBsonField(doc bson.D, key string, value any) bson.D {
	for i, elem := range doc {
		if elem.Key == key {
			doc[i].Value = value
			return doc
		}
	}
	return append(doc, bson.E{Key: key, Value: value})
}

// secGetBsonArray returns the bson.A for a named field, or nil if absent.
func secGetBsonArray(doc bson.D, key string) bson.A {
	for _, elem := range doc {
		if elem.Key == key {
			if arr, ok := elem.Value.(bson.A); ok {
				return arr
			}
		}
	}
	return nil
}

// secBsonString extracts a string value from a BSON any value (best-effort).
func secBsonString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// secExtractBsonIDValue extracts a string ID from a BSON field value.
func secExtractBsonIDValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bson.Binary:
		return blobToUUID(val.Data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// secSplitQualifiedRef extracts the last component from "Module.Entity.AttrName".
func secSplitQualifiedRef(ref string) string {
	parts := secSplitByDot(ref)
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}

// secSplitAssocRef extracts the association name from "Module.AssocName".
func secSplitAssocRef(ref string) string {
	parts := secSplitByDot(ref)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// secSplitByDot splits a string by ".".
func secSplitByDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// secDowngradeCalculatedAttrRights downgrades ReadWrite/WriteOnly to ReadOnly
// for calculated attributes (which cannot be written).
func secDowngradeCalculatedAttrRights(doc bson.D) bson.D {
	for i, f := range doc {
		if f.Key == "AccessRights" {
			if rights, ok := f.Value.(string); ok && (rights == "ReadWrite" || rights == "WriteOnly") {
				doc[i].Value = "ReadOnly"
			}
		}
	}
	return doc
}

// secInvalidAccessRuleProps lists BSON keys that are NOT valid Mendix metamodel
// properties on DomainModels$AccessRule. Old mxcli versions wrote these;
// Studio Pro crashes with "Sequence contains no matching element" if present.
var secInvalidAccessRuleProps = map[string]bool{
	"AllowRead":  true,
	"AllowWrite": true,
}

// secStripInvalidAccessRuleProps removes invalid properties from an AccessRule
// BSON document. Returns the cleaned document and true if anything was removed.
func secStripInvalidAccessRuleProps(doc bson.D) (bson.D, bool) {
	cleaned := make(bson.D, 0, len(doc))
	stripped := false
	for _, f := range doc {
		if secInvalidAccessRuleProps[f.Key] {
			stripped = true
			continue
		}
		cleaned = append(cleaned, f)
	}
	return cleaned, stripped
}
