// SPDX-License-Identifier: Apache-2.0

package model

type Enumeration struct {
	BaseElement
	ContainerID   ID                 `json:"containerId"`
	Name          string             `json:"name"`
	Documentation string             `json:"documentation,omitempty"`
	Values        []EnumerationValue `json:"values,omitempty"`
}

func (e *Enumeration) GetName() string {
	return e.Name
}

func (e *Enumeration) GetContainerID() ID {
	return e.ContainerID
}

type EnumerationValue struct {
	BaseElement
	Name    string `json:"name"`
	Caption *Text  `json:"caption,omitempty"`
	Image   *Image `json:"image,omitempty"`
}

func (v *EnumerationValue) GetName() string {
	return v.Name
}

type RegularExpression struct {
	BaseElement
	ContainerID   ID     `json:"containerId"`
	Name          string `json:"name"`
	Documentation string `json:"documentation,omitempty"`
	Expression    string `json:"expression"`
}

func (r *RegularExpression) GetName() string {
	return r.Name
}

func (r *RegularExpression) GetContainerID() ID {
	return r.ContainerID
}
