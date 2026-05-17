// SPDX-License-Identifier: Apache-2.0

// convert_reader.go — gen/* → model.* conversion helpers for msdkReader-based methods.
//
// Phase 4B: these converters extract the fields consumed by the executor
// from gen-typed reads (via mprread free functions), preserving the
// backend.* interface signatures that callers depend on.
//
// Keep each converter narrow: only populate fields actually read by the
// executor. New fields should be added on demand, not preemptively.

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genProj "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

// ---------------------------------------------------------------------------
// Enumeration
// ---------------------------------------------------------------------------

func enumToModel(u mprread.Unit[*genEnum.Enumeration]) *model.Enumeration {
	e := u.Element
	out := &model.Enumeration{
		BaseElement: model.BaseElement{
			ID:       model.ID(e.ID()),
			TypeName: "Enumerations$Enumeration",
		},
		ContainerID:   u.ContainerID,
		Name:          e.Name(),
		Documentation: e.Documentation(),
	}
	for _, item := range e.ValuesItems() {
		ev, ok := item.(*genEnum.EnumerationValue)
		if !ok {
			continue
		}
		out.Values = append(out.Values, enumValueToModel(ev))
	}
	return out
}

func enumValueToModel(v *genEnum.EnumerationValue) model.EnumerationValue {
	out := model.EnumerationValue{
		BaseElement: model.BaseElement{
			ID:       model.ID(v.ID()),
			TypeName: "Enumerations$EnumerationValue",
		},
		Name: v.Name(),
	}
	if cap := v.Caption(); cap != nil {
		out.Caption = textElementToModel(cap)
	}
	return out
}

// textElementToModel extracts translations from a gen Texts$Text element.
// Returns nil if the element does not expose translations. Uses duck-typing
// to avoid importing the texts gen package directly.
func textElementToModel(el element.Element) *model.Text {
	type translationsAccessor interface {
		TranslationsItems() []element.Element
	}
	type translationAccessor interface {
		LanguageCode() string
		Text() string
	}
	ta, ok := el.(translationsAccessor)
	if !ok {
		return nil
	}
	out := &model.Text{Translations: map[string]string{}}
	for _, item := range ta.TranslationsItems() {
		tr, ok := item.(translationAccessor)
		if !ok {
			continue
		}
		if code := tr.LanguageCode(); code != "" {
			out.Translations[code] = tr.Text()
		}
	}
	return out
}

func enumUnitsToModel(units []mprread.Unit[*genEnum.Enumeration]) []*model.Enumeration {
	out := make([]*model.Enumeration, len(units))
	for i, u := range units {
		out[i] = enumToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// Constant
// ---------------------------------------------------------------------------

func constToModel(u mprread.Unit[*genConst.Constant]) *model.Constant {
	c := u.Element
	return &model.Constant{
		BaseElement: model.BaseElement{
			ID:       model.ID(c.ID()),
			TypeName: "Constants$Constant",
		},
		ContainerID:     u.ContainerID,
		Name:            c.Name(),
		Documentation:   c.Documentation(),
		Type:            constantDataTypeToModel(c.Type()),
		DefaultValue:    c.DefaultValue(),
		ExposedToClient: c.ExposedToClient(),
		Excluded:        c.Excluded(),
		ExportLevel:     c.ExportLevel(),
	}
}

func constantDataTypeToModel(typ element.Element) model.ConstantDataType {
	dt := model.ConstantDataType{Kind: "Unknown"}
	if typ == nil {
		return dt
	}
	switch typ.TypeName() {
	case "DataTypes$StringType":
		dt.Kind = "String"
	case "DataTypes$IntegerType":
		dt.Kind = "Integer"
	case "DataTypes$LongType":
		dt.Kind = "Long"
	case "DataTypes$DecimalType":
		dt.Kind = "Decimal"
	case "DataTypes$BooleanType":
		dt.Kind = "Boolean"
	case "DataTypes$DateTimeType":
		dt.Kind = "DateTime"
	case "DataTypes$BinaryType":
		dt.Kind = "Binary"
	case "DataTypes$FloatType":
		dt.Kind = "Float"
	case "DataTypes$EnumerationType":
		dt.Kind = "Enumeration"
		if et, ok := typ.(*genDT.EnumerationType); ok {
			dt.EnumRef = et.EnumerationQualifiedName()
		}
	case "DataTypes$ObjectType":
		dt.Kind = "Object"
		if ot, ok := typ.(*genDT.ObjectType); ok {
			dt.EntityRef = ot.EntityQualifiedName()
		}
	case "DataTypes$ListType":
		dt.Kind = "List"
		if lt, ok := typ.(*genDT.ListType); ok {
			dt.EntityRef = lt.EntityQualifiedName()
		}
	}
	return dt
}

func constUnitsToModel(units []mprread.Unit[*genConst.Constant]) []*model.Constant {
	out := make([]*model.Constant, len(units))
	for i, u := range units {
		out[i] = constToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// ScheduledEvent
// ---------------------------------------------------------------------------

func schedEventToModel(u mprread.Unit[*genSched.ScheduledEvent]) *model.ScheduledEvent {
	s := u.Element
	return &model.ScheduledEvent{
		BaseElement: model.BaseElement{
			ID:       model.ID(s.ID()),
			TypeName: "ScheduledEvents$ScheduledEvent",
		},
		ContainerID:   u.ContainerID,
		Name:          s.Name(),
		Documentation: s.Documentation(),
		Interval:      int(s.Interval()),
		IntervalType:  s.IntervalType(),
		Enabled:       s.Enabled(),
	}
}

func schedEventUnitsToModel(units []mprread.Unit[*genSched.ScheduledEvent]) []*model.ScheduledEvent {
	out := make([]*model.ScheduledEvent, len(units))
	for i, u := range units {
		out[i] = schedEventToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// ModuleSettings
// ---------------------------------------------------------------------------

func moduleSettingsToTypes(u mprread.Unit[*genProj.ModuleSettings]) *types.ModuleSettings {
	ms := u.Element
	out := &types.ModuleSettings{
		ID:                  model.ID(ms.ID()),
		ContainerID:         u.ContainerID,
		ExportLevel:         ms.ExportLevel(),
		ProtectedModuleType: ms.ProtectedModuleType(),
		Version:             ms.Version(),
		BasedOnVersion:      ms.BasedOnVersion(),
		ExtensionName:       ms.ExtensionName(),
		SolutionIdentifier:  ms.SolutionIdentifier(),
	}
	for _, item := range ms.JarDependenciesItems() {
		jd, ok := item.(*genProj.JarDependency)
		if !ok {
			continue
		}
		out.JarDependencies = append(out.JarDependencies, jarDepToTypes(jd))
	}
	return out
}

func jarDepToTypes(jd *genProj.JarDependency) *types.JarDependency {
	out := &types.JarDependency{
		ID:         model.ID(jd.ID()),
		GroupID:    jd.GroupId(),
		ArtifactID: jd.ArtifactId(),
		Version:    jd.Version(),
		IsIncluded: jd.IsIncluded(),
	}
	for _, item := range jd.ExclusionsItems() {
		ex, ok := item.(*genProj.JarDependencyExclusion)
		if !ok {
			continue
		}
		out.Exclusions = append(out.Exclusions, &types.JarDependencyExclusion{
			ID:         model.ID(ex.ID()),
			GroupID:    ex.GroupId(),
			ArtifactID: ex.ArtifactId(),
		})
	}
	return out
}

func moduleSettingsUnitsToTypes(units []mprread.Unit[*genProj.ModuleSettings]) []*types.ModuleSettings {
	out := make([]*types.ModuleSettings, len(units))
	for i, u := range units {
		out[i] = moduleSettingsToTypes(u)
	}
	return out
}
