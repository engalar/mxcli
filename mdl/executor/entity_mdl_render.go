// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// entityMDLSpec is the flat intermediate used to render entity MDL text. It is
// produced from either an AST statement (entitySpecFromAST) or a gen-typed BSON
// entity (entitySpecFromGen), then handed to renderEntityMDL. This replaces the
// retired mdl/canonical/entity lifecycle package.
type entityMDLSpec struct {
	module        string
	name          string
	kind          string // "persistent", "non-persistent", "view", "external"
	documentation string
	hasPosition   bool
	positionX     int
	positionY     int
	extendsQN     string
	attributes    []attrMDLSpec
	indexes       []indexMDLSpec
	eventHandlers []eventHandlerMDLSpec
	systemMembers []string
	oql           string
}

type attrMDLSpec struct {
	name           string
	documentation  string
	dataType       canonical.DataType
	notNull        bool
	notNullError   string
	unique         bool
	uniqueError    string
	hasDefault     bool
	defaultValue   string
	calculated     bool
	calculatedMFQN string
}

type indexMDLSpec struct {
	name    string
	columns []indexColumnMDLSpec
}

type indexColumnMDLSpec struct {
	name      string
	ascending bool
}

type eventHandlerMDLSpec struct {
	moment            string
	event             string
	microflowQN       string
	raiseErrorOnFalse bool
	passEventObject   bool
}

// renderEntityMDL renders the spec as deterministic MDL text. When
// createOrModify is true the statement begins with `create or modify` so the
// output is idempotent on re-execution (used by DESCRIBE); otherwise it begins
// with `create`. The prefix is injected at the statement line — never via
// post-hoc string substitution — so documentation blocks that happen to contain
// the word "create" are preserved verbatim.
func renderEntityMDL(spec entityMDLSpec, createOrModify bool) string {
	var sb strings.Builder
	if spec.documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", spec.documentation)
	}
	if spec.hasPosition {
		fmt.Fprintf(&sb, "@Position(%d, %d)\n", spec.positionX, spec.positionY)
	}
	kindStr := spec.kind
	if kindStr == "" {
		kindStr = "persistent"
	}
	prefix := "create"
	if createOrModify {
		prefix = "create or modify"
	}
	qn := spec.name
	if spec.module != "" {
		qn = spec.module + "." + spec.name
	}
	if spec.extendsQN != "" {
		fmt.Fprintf(&sb, "%s %s entity %s extends %s (\n", prefix, kindStr, qn, spec.extendsQN)
	} else {
		fmt.Fprintf(&sb, "%s %s entity %s (\n", prefix, kindStr, qn)
	}
	for i, attr := range spec.attributes {
		if attr.documentation != "" {
			fmt.Fprintf(&sb, "  /** %s */\n", attr.documentation)
		}
		comma := ","
		if i == len(spec.attributes)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "  %s: %s%s%s\n", attr.name, entityDataTypeToMDL(attr.dataType), entityAttrConstraintsToMDL(attr), comma)
	}
	sb.WriteString(")")
	// Note: index Name may be a UUID-formatted DataStorageGuid (e.g.
	// "d3f9a8b2-1234-...") which contains hyphens and is not a valid MDL
	// IDENTIFIER. This causes DESCRIBE output to be non-re-parseable for
	// entities with named indexes. Tracked as a known gap.
	for _, idx := range spec.indexes {
		cols := make([]string, 0, len(idx.columns))
		for _, col := range idx.columns {
			if col.ascending {
				cols = append(cols, col.name)
			} else {
				cols = append(cols, col.name+" desc")
			}
		}
		if idx.name != "" {
			fmt.Fprintf(&sb, "\nindex %s (%s)", idx.name, strings.Join(cols, ", "))
		} else {
			fmt.Fprintf(&sb, "\nindex (%s)", strings.Join(cols, ", "))
		}
	}
	for _, eh := range spec.eventHandlers {
		paramStr := "()"
		if eh.passEventObject {
			paramStr = "($currentObject)"
		}
		options := ""
		if eh.raiseErrorOnFalse && strings.EqualFold(eh.moment, "Before") {
			options = " raise error"
		}
		fmt.Fprintf(&sb, "\non %s %s call %s%s%s",
			strings.ToLower(eh.moment), strings.ToLower(eh.event),
			eh.microflowQN, paramStr, options)
	}
	if len(spec.systemMembers) > 0 {
		fmt.Fprintf(&sb, "\nsystem members (%s)", strings.Join(spec.systemMembers, ", "))
	}
	if spec.kind == "view" && spec.oql != "" {
		sb.WriteString(" as (\n")
		for _, line := range strings.Split(spec.oql, "\n") {
			trimmed := strings.TrimLeft(line, " \t")
			fmt.Fprintf(&sb, "  %s\n", trimmed)
		}
		sb.WriteString(")")
	}
	return sb.String()
}

// entityDataTypeToMDL renders a canonical DataType as its MDL surface syntax.
func entityDataTypeToMDL(dt canonical.DataType) string {
	switch dt.Kind {
	case canonical.KindString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case canonical.KindInteger:
		return "Integer"
	case canonical.KindLong:
		return "Long"
	case canonical.KindDecimal:
		if dt.Precision > 0 {
			return fmt.Sprintf("Decimal(%d, %d)", dt.Precision, dt.Scale)
		}
		return "Decimal"
	case canonical.KindBoolean:
		return "Boolean"
	case canonical.KindDateTime:
		return "DateTime"
	case canonical.KindBinary:
		return "Binary"
	case canonical.KindAutoNumber:
		return "AutoNumber"
	case canonical.KindEnumRef, canonical.KindEntityRef, canonical.KindUnresolvedRef:
		return dt.Ref
	case canonical.KindListOf:
		return "List of " + dt.Ref
	default:
		return "Unknown"
	}
}

// entityAttrConstraintsToMDL renders the trailing constraint clauses (not null,
// unique, default, calculated) for one attribute.
func entityAttrConstraintsToMDL(attr attrMDLSpec) string {
	var sb strings.Builder
	if attr.notNull {
		sb.WriteString(" not null")
		if attr.notNullError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.notNullError, "'", "''"))
		}
	}
	if attr.unique {
		sb.WriteString(" unique")
		if attr.uniqueError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.uniqueError, "'", "''"))
		}
	}
	if attr.hasDefault {
		fmt.Fprintf(&sb, " default %s", attr.defaultValue)
	}
	if attr.calculated && attr.calculatedMFQN != "" {
		fmt.Fprintf(&sb, " calculated by %s", attr.calculatedMFQN)
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// AST -> spec (CREATE ENTITY render + diff proposed side)
// ---------------------------------------------------------------------------

// entitySpecFromAST builds an entityMDLSpec from a parsed CREATE ENTITY
// statement. It does NOT inject Mendix write-path invariants (e.g. Boolean
// `default false`) — those belong to the write path, not the renderer; the
// AST's HasDefault / DefaultValue values are used verbatim.
func entitySpecFromAST(s *ast.CreateEntityStmt) entityMDLSpec {
	if s == nil {
		return entityMDLSpec{}
	}
	spec := entityMDLSpec{
		module:        s.Name.Module,
		name:          s.Name.Name,
		kind:          entityKindStr(s.Kind),
		documentation: s.Documentation,
		systemMembers: append([]string(nil), s.SystemMembers...),
	}
	if s.Position != nil {
		spec.hasPosition = true
		spec.positionX = s.Position.X
		spec.positionY = s.Position.Y
	}
	if s.Generalization != nil {
		spec.extendsQN = s.Generalization.String()
	}
	for _, a := range s.Attributes {
		as := attrMDLSpec{
			name:          a.Name,
			documentation: a.Documentation,
			dataType:      astDataTypeToCanonical(a.Type),
			notNull:       a.NotNull,
			notNullError:  a.NotNullError,
			unique:        a.Unique,
			uniqueError:   a.UniqueError,
			hasDefault:    a.HasDefault,
			calculated:    a.Calculated,
		}
		if a.HasDefault {
			as.defaultValue = formatEntityDefault(a.DefaultValue)
		}
		if a.CalculatedMicroflow != nil {
			as.calculatedMFQN = a.CalculatedMicroflow.String()
		}
		spec.attributes = append(spec.attributes, as)
	}
	for _, idx := range s.Indexes {
		im := indexMDLSpec{}
		for _, col := range idx.Columns {
			im.columns = append(im.columns, indexColumnMDLSpec{name: col.Name, ascending: !col.Descending})
		}
		spec.indexes = append(spec.indexes, im)
	}
	for _, eh := range s.EventHandlers {
		spec.eventHandlers = append(spec.eventHandlers, eventHandlerMDLSpec{
			moment:            eh.Moment,
			event:             eh.Event,
			microflowQN:       eh.Microflow.String(),
			raiseErrorOnFalse: eh.RaiseErrorOnFalse,
			passEventObject:   eh.PassEventObject,
		})
	}
	return spec
}

// entityKindStr maps an ast.EntityKind to the MDL keyword form.
func entityKindStr(k ast.EntityKind) string {
	switch k {
	case ast.EntityNonPersistent:
		return "non-persistent"
	case ast.EntityView:
		return "view"
	case ast.EntityExternal:
		return "external"
	default:
		return "persistent"
	}
}

// astDataTypeToCanonical collapses TypeEnumeration into KindUnresolvedRef (the
// parser cannot disambiguate enum vs entity for a bare qualified name); other
// kinds map directly.
func astDataTypeToCanonical(dt ast.DataType) canonical.DataType {
	switch dt.Kind {
	case ast.TypeString:
		return canonical.DataType{Kind: canonical.KindString, Length: dt.Length}
	case ast.TypeInteger:
		return canonical.DataType{Kind: canonical.KindInteger}
	case ast.TypeLong:
		return canonical.DataType{Kind: canonical.KindLong}
	case ast.TypeDecimal:
		return canonical.DataType{Kind: canonical.KindDecimal, Precision: dt.Precision, Scale: dt.Scale}
	case ast.TypeBoolean:
		return canonical.DataType{Kind: canonical.KindBoolean}
	case ast.TypeDateTime, ast.TypeDate:
		return canonical.DataType{Kind: canonical.KindDateTime}
	case ast.TypeBinary:
		return canonical.DataType{Kind: canonical.KindBinary}
	case ast.TypeAutoNumber:
		return canonical.DataType{Kind: canonical.KindAutoNumber}
	case ast.TypeEnumeration:
		ref := ""
		if dt.EnumRef != nil {
			ref = dt.EnumRef.String()
		}
		return canonical.DataType{Kind: canonical.KindUnresolvedRef, Ref: ref}
	case ast.TypeEntity:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindEntityRef, Ref: ref}
	case ast.TypeListOf:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindListOf, Ref: ref}
	default:
		return canonical.DataType{Kind: canonical.KindUnknown}
	}
}

// formatEntityDefault renders an attribute default value for MDL serialisation.
// Strings are emitted verbatim (the AST already carries the quoted/escaped
// form); everything else gets the default %v representation.
func formatEntityDefault(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}

// ---------------------------------------------------------------------------
// gen -> spec (DESCRIBE ENTITY + diff current side)
// ---------------------------------------------------------------------------

// entitySpecFromGen builds an entityMDLSpec from a gen-typed BSON entity.
// moduleName supplies the owning module (gen Entity stores only the bare name).
func entitySpecFromGen(moduleName string, e *genDm.Entity) entityMDLSpec {
	if e == nil {
		return entityMDLSpec{module: moduleName}
	}
	spec := entityMDLSpec{
		module:        moduleName,
		name:          e.Name(),
		documentation: e.Documentation(),
		kind:          entityKindFromGen(e),
	}
	if loc := e.Location(); loc != "" {
		if x, y, ok := parseEntityLocation(loc); ok {
			spec.hasPosition = true
			spec.positionX = x
			spec.positionY = y
		}
	}
	if ext := genGeneralizationQN(e); ext != "" {
		spec.extendsQN = ext
	}

	notNull := make(map[string]bool)
	unique := make(map[string]bool)
	notNullErr := make(map[string]string)
	uniqueErr := make(map[string]string)
	for _, item := range e.ValidationRulesItems() {
		vr, ok := item.(*genDm.ValidationRule)
		if !ok {
			continue
		}
		attrName := lastAttrSegment(vr.AttributeQualifiedName())
		msg := genENUSText(vr.ErrorMessage())
		switch vr.RuleInfo().(type) {
		case *genDm.RequiredRuleInfo:
			notNull[attrName] = true
			if msg != "" {
				notNullErr[attrName] = msg
			}
		case *genDm.UniqueRuleInfo:
			unique[attrName] = true
			if msg != "" {
				uniqueErr[attrName] = msg
			}
		}
	}

	attrNames := make(map[string]string)
	for _, item := range e.AttributesItems() {
		attr, ok := item.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		as := attrSpecFromGen(attr, notNull, unique)
		if msg, ok := notNullErr[attr.Name()]; ok {
			as.notNullError = msg
		}
		if msg, ok := uniqueErr[attr.Name()]; ok {
			as.uniqueError = msg
		}
		spec.attributes = append(spec.attributes, as)
	}

	for _, item := range e.IndexesItems() {
		idx, ok := item.(*genDm.Index)
		if !ok {
			continue
		}
		im := indexMDLSpec{name: idx.DataStorageGuid()}
		for _, c := range idx.AttributesItems() {
			ia, ok := c.(*genDm.IndexedAttribute)
			if !ok {
				continue
			}
			refID := string(ia.AttributeRefID())
			name := attrNames[refID]
			if name == "" {
				name = refID
			}
			im.columns = append(im.columns, indexColumnMDLSpec{name: name, ascending: ia.Ascending()})
		}
		spec.indexes = append(spec.indexes, im)
	}

	for _, item := range e.EventHandlersItems() {
		h, ok := item.(*genDm.EventHandler)
		if !ok || h.MicroflowQualifiedName() == "" {
			continue
		}
		spec.eventHandlers = append(spec.eventHandlers, eventHandlerMDLSpec{
			moment:            h.Moment(),
			event:             h.Event(),
			microflowQN:       h.MicroflowQualifiedName(),
			raiseErrorOnFalse: h.RaiseErrorOnFalse(),
			passEventObject:   h.PassEventObject(),
		})
	}

	if g, ok := e.Generalization().(*genDm.NoGeneralization); ok {
		if g.HasOwner() {
			spec.systemMembers = append(spec.systemMembers, "owner")
		}
		if g.HasCreatedDate() {
			spec.systemMembers = append(spec.systemMembers, "createdDate")
		}
		if g.HasChangedDate() {
			spec.systemMembers = append(spec.systemMembers, "changedDate")
		}
		if g.HasChangedBy() {
			spec.systemMembers = append(spec.systemMembers, "changedBy")
		}
	}

	if spec.kind == "view" {
		if src := e.Source(); src != nil {
			type oqlSource interface{ Oql() string }
			if oq, ok := src.(oqlSource); ok {
				spec.oql = oq.Oql()
			}
		}
	}
	return spec
}

// resolveViewEntityOqlFromDoc resolves the OQL from a ViewEntitySourceDocument
// when the inline Oql field on the entity's Source is empty (OQL migrated to
// separate document in Mendix >= 10.21).
func resolveViewEntityOqlFromDoc(e *genDm.Entity, ur backend.UnitReader) string {
	if e == nil || ur == nil {
		return ""
	}
	src := e.Source()
	if src == nil {
		return ""
	}
	type sdSrc interface{ SourceDocumentQualifiedName() string }
	sd, ok := src.(sdSrc)
	if !ok {
		return ""
	}
	docQN := sd.SourceDocumentQualifiedName()
	if docQN == "" {
		return ""
	}
	rawUnits, err := ur.ListRawUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return ""
	}
	var docBytes []byte
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		rawData, err := ur.GetRawUnit(ru.ID)
		if err != nil {
			continue
		}
		// Match the document by checking if docQN ends with the document's short Name
		nameVal, _ := rawData["Name"].(string)
		if nameVal == "" {
			continue
		}
		candidate := nameVal
		// docQN is "FT.DispatchSummary", document Name is "DispatchSummary"
		if strings.HasSuffix(docQN, candidate) || candidate == docQN {
			docBytes, err = ur.GetRawUnitBytes(ru.ID)
			if err == nil {
				break
			}
		}
	}
	if docBytes == nil {
		return ""
	}
	docElem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(docBytes))
	if err != nil {
		return ""
	}
	doc, ok := docElem.(*genDm.ViewEntitySourceDocument)
	if !ok || doc == nil {
		return ""
	}
	return doc.Oql()
}

// entityKindFromGen derives the MDL kind keyword from a gen entity's source /
// generalization presence bits.
func entityKindFromGen(e *genDm.Entity) string {
	if src := e.Source(); src != nil && strings.Contains(src.TypeName(), "OqlView") {
		return "view"
	}
	if g, ok := e.Generalization().(*genDm.NoGeneralization); ok && !g.Persistable() {
		return "non-persistent"
	}
	return "persistent"
}

// attrSpecFromGen builds an attrMDLSpec from a gen-typed attribute. notNull and
// unique carry the validation-rule presence maps keyed by bare attribute name.
func attrSpecFromGen(attr *genDm.Attribute, notNull, unique map[string]bool) attrMDLSpec {
	as := attrMDLSpec{
		name:          attr.Name(),
		documentation: attr.Documentation(),
		dataType:      genAttrTypeToCanonical(attr.Type()),
		notNull:       notNull[attr.Name()],
		unique:        unique[attr.Name()],
	}
	if cv, ok := attr.Value().(*genDm.CalculatedValue); ok {
		as.calculated = true
		as.calculatedMFQN = cv.MicroflowQualifiedName()
	} else if sv, ok := attr.Value().(*genDm.StoredValue); ok && sv.DefaultValue() != "" {
		as.hasDefault = true
		raw := sv.DefaultValue()
		if _, isStr := attr.Type().(*genDm.StringAttributeType); isStr {
			as.defaultValue = "'" + strings.ReplaceAll(raw, "'", "''") + "'"
		} else {
			as.defaultValue = raw
		}
	}
	return as
}

// genAttrTypeToCanonical maps a gen attribute-type element to a canonical
// DataType.
func genAttrTypeToCanonical(t any) canonical.DataType {
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

// genGeneralizationQN returns the parent qualified name for entities that have a
// Generalization (i.e., `extends Module.Parent`); returns "" otherwise.
func genGeneralizationQN(e *genDm.Entity) string {
	if g, ok := e.Generalization().(*genDm.Generalization); ok {
		return g.GeneralizationQualifiedName()
	}
	return ""
}

// genENUSText returns the en_US translation text from a *genTexts.Text, or ""
// when the message has no translations / is nil / is not a Text. Falls back to
// the first translation when no en_US entry exists.
func genENUSText(msg element.Element) string {
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

// lastAttrSegment returns the trailing component of a dotted name
// ("Mod.Ent.Attr" -> "Attr").
func lastAttrSegment(qn string) string {
	parts := strings.Split(qn, ".")
	if len(parts) == 0 {
		return qn
	}
	return parts[len(parts)-1]
}
