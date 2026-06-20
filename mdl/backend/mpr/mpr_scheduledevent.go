// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

type scheduledEventBackend struct {
	reader *modelsdkmpr.Reader
}

func newScheduledEventBackend(reader *modelsdkmpr.Reader) *scheduledEventBackend {
	return &scheduledEventBackend{reader: reader}
}

func (b *scheduledEventBackend) ListScheduledEvents() ([]*model.ScheduledEvent, error) {
	units, err := mprread.ListUnitsWithContainer[*genSched.ScheduledEvent](b.reader)
	if err != nil {
		return nil, err
	}
	return schedEventUnitsToModel(units), nil
}

func (b *scheduledEventBackend) GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error) {
	events, err := b.ListScheduledEvents()
	if err != nil {
		return nil, err
	}
	for _, s := range events {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("scheduled event not found: %s", id)
}
