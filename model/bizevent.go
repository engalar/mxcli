// SPDX-License-Identifier: Apache-2.0

package model

type BusinessEventService struct {
	BaseElement
	ContainerID              ID                       `json:"containerId"`
	Name                     string                   `json:"name"`
	Documentation            string                   `json:"documentation,omitempty"`
	Excluded                 bool                     `json:"excluded,omitempty"`
	ExportLevel              string                   `json:"exportLevel,omitempty"`
	Definition               *BusinessEventDefinition `json:"definition,omitempty"`
	OperationImplementations []*ServiceOperation      `json:"operationImplementations,omitempty"`

	// Cached AsyncAPI contract (for consumed/client services)
	Document string `json:"document,omitempty"` // AsyncAPI YAML document
}

func (s *BusinessEventService) GetName() string {
	return s.Name
}

func (s *BusinessEventService) GetContainerID() ID {
	return s.ContainerID
}

type BusinessEventDefinition struct {
	BaseElement
	ServiceName     string                  `json:"serviceName"`
	EventNamePrefix string                  `json:"eventNamePrefix,omitempty"`
	Description     string                  `json:"description,omitempty"`
	Summary         string                  `json:"summary,omitempty"`
	Channels        []*BusinessEventChannel `json:"channels,omitempty"`
}

type BusinessEventChannel struct {
	BaseElement
	ChannelName string                  `json:"channelName"`
	Description string                  `json:"description,omitempty"`
	Messages    []*BusinessEventMessage `json:"messages,omitempty"`
}

type BusinessEventMessage struct {
	BaseElement
	MessageName  string                    `json:"messageName"`
	Description  string                    `json:"description,omitempty"`
	CanPublish   bool                      `json:"canPublish"`
	CanSubscribe bool                      `json:"canSubscribe"`
	Attributes   []*BusinessEventAttribute `json:"attributes,omitempty"`
}

type BusinessEventAttribute struct {
	BaseElement
	AttributeName string `json:"attributeName"`
	AttributeType string `json:"attributeType"` // "Long", "String", "Integer", "Boolean", "DateTime", "Decimal"
	Description   string `json:"description,omitempty"`
}

type ServiceOperation struct {
	BaseElement
	MessageName string `json:"messageName"`
	Operation   string `json:"operation"`           // "publish" or "subscribe"
	Entity      string `json:"entity"`              // BY_NAME qualified ref: "Module.EntityName"
	Microflow   string `json:"microflow,omitempty"` // BY_NAME qualified ref (optional handler)
}

