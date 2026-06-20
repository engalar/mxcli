// SPDX-License-Identifier: Apache-2.0

package model

type PublishedRestService struct {
	BaseElement
	ContainerID  ID                       `json:"containerId"`
	Name         string                   `json:"name"`
	Path         string                   `json:"path,omitempty"`
	Version      string                   `json:"version,omitempty"`
	ServiceName  string                   `json:"serviceName,omitempty"`
	Excluded     bool                     `json:"excluded,omitempty"`
	AllowedRoles []string                 `json:"allowedRoles,omitempty"`
	Resources    []*PublishedRestResource `json:"resources,omitempty"`
}

func (s *PublishedRestService) GetName() string {
	return s.Name
}

func (s *PublishedRestService) GetContainerID() ID {
	return s.ContainerID
}

type PublishedRestResource struct {
	BaseElement
	Name       string                    `json:"name"`
	Operations []*PublishedRestOperation `json:"operations,omitempty"`
}

type PublishedRestOperation struct {
	BaseElement
	Path       string   `json:"path,omitempty"`
	HTTPMethod string   `json:"httpMethod,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Microflow  string   `json:"microflow,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`
	Parameters []string `json:"parameters,omitempty"` // path parameter names extracted from {param} in Path
}

type ConsumedRestService struct {
	BaseElement
	ContainerID    ID                     `json:"containerId"`
	Name           string                 `json:"name"`
	Documentation  string                 `json:"documentation,omitempty"`
	Excluded       bool                   `json:"excluded,omitempty"`
	BaseUrl        string                 `json:"baseUrl"`
	Authentication *RestAuthentication    `json:"authentication,omitempty"`
	Operations     []*RestClientOperation `json:"operations,omitempty"`
	OpenApiContent string                 `json:"openApiContent,omitempty"` // raw spec text (stored in OpenApiFile.Content BSON field)
}

func (s *ConsumedRestService) GetName() string {
	return s.Name
}

func (s *ConsumedRestService) GetContainerID() ID {
	return s.ContainerID
}

type RestAuthentication struct {
	Scheme   string `json:"scheme"`             // "Basic"
	Username string `json:"username,omitempty"` // literal value or constant reference
	Password string `json:"password,omitempty"` // literal value or constant reference
}

type RestClientOperation struct {
	Name             string                 `json:"name"`
	Documentation    string                 `json:"documentation,omitempty"`
	HttpMethod       string                 `json:"httpMethod"`                // "GET", "POST", etc.
	Path             string                 `json:"path"`                      // e.g. "/pet/{petId}"
	Tags             []string               `json:"tags,omitempty"`            // resource group labels (from OpenAPI tags[0])
	Parameters       []*RestClientParameter `json:"parameters,omitempty"`      // path parameters
	QueryParameters  []*RestClientParameter `json:"queryParameters,omitempty"` // query parameters
	Headers          []*RestClientHeader    `json:"headers,omitempty"`
	BodyType         string                 `json:"bodyType,omitempty"`     // "JSON", "FILE", "TEMPLATE", "EXPORT_MAPPING", ""
	BodyVariable     string                 `json:"bodyVariable,omitempty"` // variable name, template expression, or entity name
	BodyMappings     []*RestResponseMapping `json:"bodyMappings,omitempty"` // export mapping tree (Entity → JSON) for EXPORT_MAPPING bodies
	ResponseType     string                 `json:"responseType"`           // "JSON", "STRING", "FILE", "STATUS", "NONE", "MAPPING"
	ResponseVariable string                 `json:"responseVariable,omitempty"`
	ResponseEntity   string                 `json:"responseEntity,omitempty"`   // target entity for implicit mapping response
	ResponseMappings []*RestResponseMapping `json:"responseMappings,omitempty"` // JSON field → entity attribute
	Timeout          int                    `json:"timeout,omitempty"`          // 0 = default (300s)
}

type RestClientParameter struct {
	Name     string `json:"name"`     // parameter name (without $ prefix)
	DataType string `json:"dataType"` // "String", "Integer", "Boolean", "Decimal"
}

type RestClientHeader struct {
	Name  string `json:"name"`  // header name, e.g. "Accept"
	Value string `json:"value"` // literal, $var, or expression like "'Bearer ' + $Token"
}

type RestResponseMapping struct {
	// Value mapping: maps a JSON field to an entity attribute
	Attribute   string `json:"attribute,omitempty"` // entity attribute short name
	ExposedName string `json:"exposedName"`         // JSON field name
	JsonPath    string `json:"jsonPath,omitempty"`  // e.g. "(Object)|args|queryparam_1"

	// Object mapping: nested entity linked by association
	Entity      string                 `json:"entity,omitempty"`      // child entity (e.g. "RestDemo.Args")
	Association string                 `json:"association,omitempty"` // e.g. "RestDemo.Args_PostDemo2Response"
	Children    []*RestResponseMapping `json:"children,omitempty"`    // recursive children
}

