// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type Module struct {
	BaseElement
	Name                string `json:"name"`
	Documentation       string `json:"documentation,omitempty"`
	Excluded            bool   `json:"excluded,omitempty"`
	FromAppStore        bool   `json:"fromAppStore,omitempty"`
	AppStoreVersion     string `json:"appStoreVersion,omitempty"`
	AppStoreGuid        string `json:"appStoreGuid,omitempty"`
	IsReusableComponent bool   `json:"isReusableComponent,omitempty"`

	// Contained units
	DomainModelID ID   `json:"domainModelId,omitempty"`
	Documents     []ID `json:"documents,omitempty"`
}

func (m *Module) GetName() string {
	return m.Name
}

type Project struct {
	BaseElement
	Name            string    `json:"name"`
	MendixVersion   string    `json:"mendixVersion"`
	ProjectID       string    `json:"projectId,omitempty"`
	IsSystemProject bool      `json:"isSystemProject,omitempty"`
	CreatedDate     time.Time `json:"createdDate,omitempty"`

	// Project-level settings
	Modules          []ID `json:"modules,omitempty"`
	ProjectDocuments []ID `json:"projectDocuments,omitempty"`
}

func (p *Project) GetName() string {
	return p.Name
}

type Folder struct {
	BaseElement
	ContainerID ID     `json:"containerId"`
	Name        string `json:"name"`
	Documents   []ID   `json:"documents,omitempty"`
	Folders     []ID   `json:"folders,omitempty"`
}

func (f *Folder) GetName() string {
	return f.Name
}

func (f *Folder) GetContainerID() ID {
	return f.ContainerID
}

