// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type ScheduledEvent struct {
	BaseElement
	ContainerID   ID         `json:"containerId"`
	Name          string     `json:"name"`
	Documentation string     `json:"documentation,omitempty"`
	MicroflowID   ID         `json:"microflowId,omitempty"`
	StartDateTime *time.Time `json:"startDateTime,omitempty"`
	TimeZone      string     `json:"timeZone,omitempty"`
	Interval      int        `json:"interval,omitempty"`
	IntervalType  string     `json:"intervalType,omitempty"`
	Enabled       bool       `json:"enabled"`
}

func (s *ScheduledEvent) GetName() string {
	return s.Name
}

func (s *ScheduledEvent) GetContainerID() ID {
	return s.ContainerID
}
