// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// extractEnUSText returns the en_US translation text from a *genTexts.Text,
// or "" when the message has no translations / is nil / is not a Text. We
// fall back to the first translation if no en_US entry exists.
func extractEnUSText(msg element.Element) string {
	t, ok := msg.(*genTexts.Text)
	if !ok || t == nil {
		return ""
	}
	var first string
	for _, item := range t.TranslationsItems() {
		tr, ok := item.(*genTexts.Translation)
		if !ok {
			continue
		}
		if first == "" {
			first = tr.Text()
		}
		if tr.LanguageCode() == "en_US" {
			return tr.Text()
		}
	}
	return first
}

// Hydrate builds a canonical EntityModel from a gen-typed *genDm.Entity.
// moduleName supplies the owning module (gen Entity stores only the bare name).
// Any unrecognised child types are surfaced as Warning entries — they do not
// abort the conversion.
func Hydrate(moduleName string, e *genDm.Entity) (*EntityModel, []canonical.Warning, error) {
	if e == nil {
		return nil, nil, fmt.Errorf("entity.Hydrate: nil entity")
	}
	var warns []canonical.Warning
	m := &EntityModel{
		Name:          QualifiedName{Module: moduleName, Name: e.Name()},
		Documentation: e.Documentation(),
		Kind:          hydrateKind(e),
	}
	if loc := e.Location(); loc != "" {
		if x, y, ok := parseLocation(loc); ok {
			m.Position = &Position{X: x, Y: y}
		}
	}
	if ext := generalizationQN(e); ext != "" {
		parts := strings.SplitN(ext, ".", 2)
		if len(parts) == 2 {
			qn := QualifiedName{Module: parts[0], Name: parts[1]}
			m.Extends = &qn
		}
	}

	notNullAttrs := make(map[string]bool)
	uniqueAttrs := make(map[string]bool)
	notNullErrors := make(map[string]string)
	uniqueErrors := make(map[string]string)
	for _, item := range e.ValidationRulesItems() {
		vr, ok := item.(*genDm.ValidationRule)
		if !ok {
			warns = append(warns, canonical.Warning{Field: "ValidationRules", Message: fmt.Sprintf("unexpected type %T", item)})
			continue
		}
		attrName := lastSegment(vr.AttributeQualifiedName())
		ruleType := ""
		if ri := vr.RuleInfo(); ri != nil {
			ruleType = ri.TypeName()
		}
		msg := extractEnUSText(vr.ErrorMessage())
		switch ruleType {
		case "DomainModels$RequiredRuleInfo":
			notNullAttrs[attrName] = true
			if msg != "" {
				notNullErrors[attrName] = msg
			}
		case "DomainModels$UniqueRuleInfo":
			uniqueAttrs[attrName] = true
			if msg != "" {
				uniqueErrors[attrName] = msg
			}
		}
	}

	for _, item := range e.AttributesItems() {
		attr, ok := item.(*genDm.Attribute)
		if !ok {
			warns = append(warns, canonical.Warning{Field: "Attributes", Message: fmt.Sprintf("unexpected type %T", item)})
			continue
		}
		am := hydrateAttribute(attr, notNullAttrs, uniqueAttrs)
		if msg, ok := notNullErrors[attr.Name()]; ok {
			am.NotNullError = msg
		}
		if msg, ok := uniqueErrors[attr.Name()]; ok {
			am.UniqueError = msg
		}
		m.Attributes = append(m.Attributes, am)
	}

	for _, item := range e.IndexesItems() {
		idx, ok := item.(*genDm.Index)
		if !ok {
			warns = append(warns, canonical.Warning{Field: "Indexes", Message: fmt.Sprintf("unexpected type %T", item)})
			continue
		}
		m.Indexes = append(m.Indexes, hydrateIndex(idx, e))
	}
	return m, warns, nil
}

func hydrateKind(e *genDm.Entity) EntityKind {
	if src := e.Source(); src != nil && strings.Contains(src.TypeName(), "OqlView") {
		return EntityView
	}
	if g, ok := e.Generalization().(*genDm.NoGeneralization); ok && !g.Persistable() {
		return EntityNonPersistent
	}
	return EntityPersistent
}

func hydrateAttribute(attr *genDm.Attribute, notNull, unique map[string]bool) AttributeModel {
	am := AttributeModel{
		Name:          attr.Name(),
		Documentation: attr.Documentation(),
		Type:          hydrateDataType(attr.Type()),
		NotNull:       notNull[attr.Name()],
		Unique:        unique[attr.Name()],
	}
	if cv, ok := attr.Value().(*genDm.CalculatedValue); ok {
		am.Calculated = true
		if mfn := cv.MicroflowQualifiedName(); mfn != "" {
			parts := strings.SplitN(mfn, ".", 2)
			if len(parts) == 2 {
				qn := QualifiedName{Module: parts[0], Name: parts[1]}
				am.CalculatedMicroflow = &qn
			}
		}
	} else if sv, ok := attr.Value().(*genDm.StoredValue); ok && sv.DefaultValue() != "" {
		am.HasDefault = true
		raw := sv.DefaultValue()
		if _, isStr := attr.Type().(*genDm.StringAttributeType); isStr {
			am.DefaultValue = "'" + strings.ReplaceAll(raw, "'", "''") + "'"
		} else {
			am.DefaultValue = raw
		}
	}
	return am
}

func hydrateDataType(t any) canonical.DataType {
	switch v := t.(type) {
	case *genDm.StringAttributeType:
		return canonical.DataType{Kind: canonical.KindString, Length: int(v.Length())}
	case *genDm.IntegerAttributeType:
		return canonical.DataType{Kind: canonical.KindInteger}
	case *genDm.LongAttributeType:
		return canonical.DataType{Kind: canonical.KindLong}
	case *genDm.DecimalAttributeType:
		return canonical.DataType{Kind: canonical.KindDecimal}
	case *genDm.BooleanAttributeType:
		return canonical.DataType{Kind: canonical.KindBoolean}
	case *genDm.DateTimeAttributeType:
		return canonical.DataType{Kind: canonical.KindDateTime}
	case *genDm.BinaryAttributeType:
		return canonical.DataType{Kind: canonical.KindBinary}
	case *genDm.AutoNumberAttributeType:
		return canonical.DataType{Kind: canonical.KindAutoNumber}
	case *genDm.EnumerationAttributeType:
		return canonical.DataType{Kind: canonical.KindEnumRef, Ref: v.EnumerationQualifiedName()}
	default:
		return canonical.DataType{Kind: canonical.KindUnknown}
	}
}

func hydrateIndex(idx *genDm.Index, e *genDm.Entity) IndexModel {
	attrNames := make(map[string]string)
	for _, item := range e.AttributesItems() {
		if a, ok := item.(*genDm.Attribute); ok {
			attrNames[string(a.ID())] = a.Name()
		}
	}
	im := IndexModel{Name: idx.DataStorageGuid()}
	for _, item := range idx.AttributesItems() {
		ia, ok := item.(*genDm.IndexedAttribute)
		if !ok {
			continue
		}
		refID := string(ia.AttributeRefID())
		name := attrNames[refID]
		if name == "" {
			name = refID
		}
		im.Columns = append(im.Columns, IndexColumn{Name: name, Ascending: ia.Ascending()})
	}
	return im
}

// generalizationQN returns the parent qualified name for entities that have a
// Generalization (i.e., `extends Module.Parent`); returns "" otherwise.
func generalizationQN(e *genDm.Entity) string {
	if g, ok := e.Generalization().(*genDm.Generalization); ok {
		return g.GeneralizationQualifiedName()
	}
	return ""
}

// parseLocation parses Mendix location strings. Mendix uses semicolon-separated
// "X;Y" format in MPR BSON; older mxcli versions incorrectly wrote space-separated
// "X Y". Both are accepted for backward compatibility.
func parseLocation(loc string) (x, y int, ok bool) {
	if _, err := fmt.Sscanf(loc, "%d;%d", &x, &y); err == nil {
		return x, y, true
	}
	_, err := fmt.Sscanf(loc, "%d %d", &x, &y)
	return x, y, err == nil
}

// lastSegment returns the trailing component of a dotted name
// ("Mod.Ent.Attr" -> "Attr").
func lastSegment(qn string) string {
	parts := strings.Split(qn, ".")
	if len(parts) == 0 {
		return qn
	}
	return parts[len(parts)-1]
}
