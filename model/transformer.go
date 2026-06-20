// SPDX-License-Identifier: Apache-2.0

package model

type DataTransformer struct {
	BaseElement
	ContainerID ID                     `json:"containerId"`
	Name        string                 `json:"name"`
	SourceType  string                 `json:"sourceType,omitempty"` // "JSON", "XML"
	SourceJSON  string                 `json:"sourceJson,omitempty"` // source content
	Steps       []*DataTransformerStep `json:"steps,omitempty"`
	Excluded    bool                   `json:"excluded,omitempty"`
}

func (t *DataTransformer) GetName() string { return t.Name }

func (t *DataTransformer) GetContainerID() ID { return t.ContainerID }

type DataTransformerStep struct {
	Technology string `json:"technology"` // "JSLT", "XSLT"
	Expression string `json:"expression"` // the transformation expression
}
