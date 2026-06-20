// SPDX-License-Identifier: Apache-2.0

package model

type ConsumedODataService struct {
	BaseElement
	ContainerID       ID     `json:"containerId"`
	Name              string `json:"name"`
	Documentation     string `json:"documentation,omitempty"`
	Version           string `json:"version,omitempty"`
	ServiceName       string `json:"serviceName,omitempty"`
	ODataVersion      string `json:"odataVersion,omitempty"`
	MetadataUrl       string `json:"metadataUrl,omitempty"`
	TimeoutExpression string `json:"timeoutExpression,omitempty"`
	ProxyType         string `json:"proxyType,omitempty"`
	Description       string `json:"description,omitempty"`
	Validated         bool   `json:"validated,omitempty"`
	Excluded          bool   `json:"excluded,omitempty"`

	// HTTP configuration (nested Microflows$HttpConfiguration part)
	HttpConfiguration *HttpConfiguration `json:"httpConfiguration,omitempty"`

	// Microflow references (BY_NAME)
	ConfigurationMicroflow string `json:"configurationMicroflow,omitempty"` // Microflow for configuring requests
	ErrorHandlingMicroflow string `json:"errorHandlingMicroflow,omitempty"` // Microflow for handling errors

	// Proxy constant references (BY_NAME to Constants$Constant)
	ProxyHost     string `json:"proxyHost,omitempty"`
	ProxyPort     string `json:"proxyPort,omitempty"`
	ProxyUsername string `json:"proxyUsername,omitempty"`
	ProxyPassword string `json:"proxyPassword,omitempty"`

	// Cached contract metadata (from $metadata endpoint)
	Metadata     string `json:"metadata,omitempty"`     // Full $metadata XML (EDMX/CSDL)
	MetadataHash string `json:"metadataHash,omitempty"` // SHA-256 hash of metadata for change detection

	// Mendix Catalog integration
	ApplicationId   string `json:"applicationId,omitempty"`
	EndpointId      string `json:"endpointId,omitempty"`
	CatalogUrl      string `json:"catalogUrl,omitempty"`
	EnvironmentType string `json:"environmentType,omitempty"`
}

func (s *ConsumedODataService) GetName() string {
	return s.Name
}

func (s *ConsumedODataService) GetContainerID() ID {
	return s.ContainerID
}

type HttpConfiguration struct {
	BaseElement
	UseAuthentication bool               `json:"useAuthentication,omitempty"`
	Username          string             `json:"username,omitempty"`          // Expression for username
	Password          string             `json:"password,omitempty"`          // Expression for password
	HttpMethod        string             `json:"httpMethod,omitempty"`        // Get, Post, Put, Patch, Delete, Head, Options
	OverrideLocation  bool               `json:"overrideLocation,omitempty"`  // Whether to use custom location
	CustomLocation    string             `json:"customLocation,omitempty"`    // Custom URL expression
	ClientCertificate string             `json:"clientCertificate,omitempty"` // Client certificate identifier
	HeaderEntries     []*HttpHeaderEntry `json:"headerEntries,omitempty"`
}

type HttpHeaderEntry struct {
	BaseElement
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"` // Expression for value
}

type PublishedODataService struct {
	BaseElement
	ContainerID         ID                     `json:"containerId"`
	Name                string                 `json:"name"`
	Documentation       string                 `json:"documentation,omitempty"`
	Path                string                 `json:"path,omitempty"`
	Namespace           string                 `json:"namespace,omitempty"`
	ServiceName         string                 `json:"serviceName,omitempty"`
	Version             string                 `json:"version,omitempty"`
	ODataVersion        string                 `json:"odataVersion,omitempty"`
	Summary             string                 `json:"summary,omitempty"`
	Description         string                 `json:"description,omitempty"`
	PublishAssociations bool                   `json:"publishAssociations,omitempty"`
	UseGeneralization   bool                   `json:"useGeneralization,omitempty"`
	AuthenticationTypes []string               `json:"authenticationTypes,omitempty"`
	AuthMicroflow       string                 `json:"authMicroflow,omitempty"`
	EntityTypes         []*PublishedEntityType `json:"entityTypes,omitempty"`
	EntitySets          []*PublishedEntitySet  `json:"entitySets,omitempty"`
	AllowedModuleRoles  []string               `json:"allowedModuleRoles,omitempty"`
	Excluded            bool                   `json:"excluded,omitempty"`
}

func (s *PublishedODataService) GetName() string {
	return s.Name
}

func (s *PublishedODataService) GetContainerID() ID {
	return s.ContainerID
}

type PublishedEntityType struct {
	BaseElement
	Entity      string             `json:"entity,omitempty"`      // BY_NAME reference to entity
	ExposedName string             `json:"exposedName,omitempty"` // Name exposed in OData service
	Summary     string             `json:"summary,omitempty"`
	Description string             `json:"description,omitempty"`
	Members     []*PublishedMember `json:"members,omitempty"`
}

type PublishedEntitySet struct {
	BaseElement
	ExposedName    string `json:"exposedName,omitempty"`
	EntityTypeName string `json:"entityTypeName,omitempty"` // Resolved entity type name
	ReadMode       string `json:"readMode,omitempty"`
	InsertMode     string `json:"insertMode,omitempty"`
	UpdateMode     string `json:"updateMode,omitempty"`
	DeleteMode     string `json:"deleteMode,omitempty"`
	UsePaging      bool   `json:"usePaging,omitempty"`
	PageSize       int    `json:"pageSize,omitempty"`
}

type PublishedMember struct {
	BaseElement
	Kind        string `json:"kind,omitempty"`        // "attribute", "association", "id"
	Name        string `json:"name,omitempty"`        // BY_NAME reference to attribute/association
	ExposedName string `json:"exposedName,omitempty"` // Name exposed in OData service
	Filterable  bool   `json:"filterable,omitempty"`
	Sortable    bool   `json:"sortable,omitempty"`
	IsPartOfKey bool   `json:"isPartOfKey,omitempty"`
}

