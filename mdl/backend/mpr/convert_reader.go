// SPDX-License-Identifier: Apache-2.0

// convert_reader.go — gen/* → model.* conversion helpers for msdkReader-based methods.
// These are thin wrappers that map only the fields consumed by the backend interface.
// Phase 2 will replace sdk/mpr.Reader delegations with mprread.* calls + these converters.

package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
)

func enumToModel(e *genEnum.Enumeration) *model.Enumeration {
	return &model.Enumeration{
		BaseElement: model.BaseElement{ID: model.ID(e.ID())},
		Name:        e.Name(),
	}
}

func enumSliceToModel(gs []*genEnum.Enumeration) []*model.Enumeration {
	out := make([]*model.Enumeration, len(gs))
	for i, g := range gs {
		out[i] = enumToModel(g)
	}
	return out
}

func constToModel(c *genConst.Constant) *model.Constant {
	return &model.Constant{
		BaseElement: model.BaseElement{ID: model.ID(c.ID())},
		Name:        c.Name(),
	}
}

func constSliceToModel(gs []*genConst.Constant) []*model.Constant {
	out := make([]*model.Constant, len(gs))
	for i, g := range gs {
		out[i] = constToModel(g)
	}
	return out
}

func schedEventToModel(s *genSched.ScheduledEvent) *model.ScheduledEvent {
	return &model.ScheduledEvent{
		BaseElement: model.BaseElement{ID: model.ID(s.ID())},
		Name:        s.Name(),
	}
}

func schedEventSliceToModel(gs []*genSched.ScheduledEvent) []*model.ScheduledEvent {
	out := make([]*model.ScheduledEvent, len(gs))
	for i, g := range gs {
		out[i] = schedEventToModel(g)
	}
	return out
}
