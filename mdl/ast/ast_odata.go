// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// OData Write Statements
// ============================================================================

// CreateODataClientStmt represents: CREATE ODATA CLIENT Module.Name (...)
type CreateODataClientStmt struct {
	Name              QualifiedName
	Version           string
	ODataVersion      string
	MetadataUrl       string
	TimeoutExpression string
	ProxyType         string
	Description       string
	Documentation     string
	Folder            string // Folder path within module (e.g., "Integration/APIs")
	CreateOrModify    bool   // True if CREATE OR MODIFY was used

	// HTTP configuration
	ServiceUrl        string // Custom service URL (overrides metadata-derived URL)
	UseAuthentication bool
	HttpUsername      string // Mendix expression for username
	HttpPassword      string // Mendix expression for password
	ClientCertificate string

	// Microflow references
	ConfigurationMicroflow string // MICROFLOW Module.ConfigureMF
	ErrorHandlingMicroflow string // MICROFLOW Module.HandleErrorMF

	// Proxy constant references
	ProxyHost     string
	ProxyPort     string
	ProxyUsername string
	ProxyPassword string

	// Custom HTTP headers
	Headers []HeaderDef
}

// HeaderDef represents a custom HTTP header entry.
type HeaderDef struct {
	Key   string
	Value string // Mendix expression
}

func (s *CreateODataClientStmt) isStatement() {}

// AlterODataClientStmt represents: ALTER ODATA CLIENT Module.Name SET key = value
type AlterODataClientStmt struct {
	Name    QualifiedName
	Changes map[string]any // property name -> new value
}

func (s *AlterODataClientStmt) isStatement() {}

// DropODataClientStmt represents: DROP ODATA CLIENT Module.Name
type DropODataClientStmt struct {
	Name QualifiedName
}

func (s *DropODataClientStmt) isStatement() {}

// CreateODataServiceStmt represents: CREATE ODATA SERVICE Module.Name (...) AUTHENTICATION ... { ... }
type CreateODataServiceStmt struct {
	Name                QualifiedName
	Path                string
	Version             string
	ODataVersion        string
	Namespace           string
	ServiceName         string
	Summary             string
	Description         string
	Documentation       string
	Folder              string // Folder path within module (e.g., "Integration/APIs")
	PublishAssociations bool
	AuthenticationTypes []string
	Entities            []*PublishedEntityDef
	CreateOrModify      bool // True if CREATE OR MODIFY was used
}

func (s *CreateODataServiceStmt) isStatement() {}

// PublishedEntityDef represents a PUBLISH ENTITY block within an OData service.
type PublishedEntityDef struct {
	Entity      QualifiedName
	ExposedName string
	ReadMode    string
	InsertMode  string
	UpdateMode  string
	DeleteMode  string
	UsePaging   bool
	PageSize    int
	Members     []*PublishedMemberDef
}

// PublishedMemberDef represents an EXPOSE member within a PUBLISH ENTITY block.
type PublishedMemberDef struct {
	Name        string
	ExposedName string
	Filterable  bool
	Sortable    bool
	IsPartOfKey bool
}

// AlterODataServiceStmt represents: ALTER ODATA SERVICE Module.Name SET key = value
type AlterODataServiceStmt struct {
	Name    QualifiedName
	Changes map[string]any // property name -> new value
}

func (s *AlterODataServiceStmt) isStatement() {}

// DropODataServiceStmt represents: DROP ODATA SERVICE Module.Name
type DropODataServiceStmt struct {
	Name QualifiedName
}

func (s *DropODataServiceStmt) isStatement() {}

// CreateExternalEntityStmt represents: CREATE [OR MODIFY] EXTERNAL ENTITY Module.Name FROM ODATA CLIENT Module.Service (...) (attrs);
type CreateExternalEntityStmt struct {
	Name                     QualifiedName
	ServiceRef               QualifiedName // FROM ODATA CLIENT ...
	EntitySet                string
	RemoteName               string
	Countable                bool
	Creatable                bool
	Deletable                bool
	Updatable                bool
	AllowCreateChangeLocally bool        // "Allow creating and changing locally" flag
	Attributes               []Attribute // reuse from ast_entity.go
	Documentation            string
	CreateOrModify           bool
}

func (s *CreateExternalEntityStmt) isStatement() {}

// CreateExternalEntitiesStmt represents: CREATE [OR MODIFY] EXTERNAL ENTITIES FROM Module.Service [INTO Module] [ENTITIES (Name1, Name2)]
type CreateExternalEntitiesStmt struct {
	ServiceRef     QualifiedName // FROM Module.Service
	TargetModule   string        // INTO Module (optional, defaults to service module)
	EntityNames    []string      // ENTITIES (Name1, Name2) filter (optional, imports all if empty)
	CreateOrModify bool          // True if CREATE OR MODIFY was used
}

func (s *CreateExternalEntitiesStmt) isStatement() {}
