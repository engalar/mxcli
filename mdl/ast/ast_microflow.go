// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Microflow Statements
// ============================================================================

// MicroflowStatement represents a statement inside a microflow body.
type MicroflowStatement interface {
	isMicroflowStatement()
}

// ============================================================================
// Error Handling
// ============================================================================

// ErrorHandlingType represents how errors are handled on a microflow activity.
type ErrorHandlingType string

const (
	ErrorHandlingContinue              ErrorHandlingType = "Continue"
	ErrorHandlingRollback              ErrorHandlingType = "Rollback"
	ErrorHandlingCustom                ErrorHandlingType = "Custom"
	ErrorHandlingCustomWithoutRollback ErrorHandlingType = "CustomWithoutRollBack"
)

// ErrorHandlingClause represents an ON ERROR clause on a microflow statement.
type ErrorHandlingClause struct {
	Type ErrorHandlingType
	Body []MicroflowStatement // non-nil for Custom/CustomWithoutRollback
}

// MicroflowParam represents a microflow parameter.
type MicroflowParam struct {
	Name string   // Parameter name (without $ prefix)
	Type DataType // Parameter type
}

// MicroflowReturnType represents a microflow return type.
type MicroflowReturnType struct {
	Type     DataType // Return type
	Variable string   // Variable name for AS $Var clause
}

// CreateMicroflowStmt represents: CREATE MICROFLOW Module.Name (params) RETURNS type BEGIN body END
type CreateMicroflowStmt struct {
	Name           QualifiedName
	Parameters     []MicroflowParam
	ReturnType     *MicroflowReturnType
	Body           []MicroflowStatement
	Documentation  string
	Comment        string
	Folder         string // Folder path within module (e.g., "Resources/Images")
	CreateOrModify bool
	Excluded       bool // @excluded — document excluded from project
}

func (s *CreateMicroflowStmt) isStatement() {}

// DropMicroflowStmt represents: DROP MICROFLOW Module.Name
type DropMicroflowStmt struct {
	Name QualifiedName
}

func (s *DropMicroflowStmt) isStatement() {}

// CreateNanoflowStmt represents: CREATE NANOFLOW Module.Name (params) RETURNS type BEGIN body END
type CreateNanoflowStmt struct {
	Name           QualifiedName
	Parameters     []MicroflowParam
	ReturnType     *MicroflowReturnType
	Body           []MicroflowStatement
	Documentation  string
	Comment        string
	Folder         string // Folder path within module
	CreateOrModify bool
	Excluded       bool // @excluded — document excluded from project
}

func (s *CreateNanoflowStmt) isStatement() {}

// DropNanoflowStmt represents: DROP NANOFLOW Module.Name
type DropNanoflowStmt struct {
	Name QualifiedName
}

func (s *DropNanoflowStmt) isStatement() {}

// ============================================================================
// Microflow Body Statements
// ============================================================================

// DeclareStmt represents: DECLARE $Var Type = expr
type DeclareStmt struct {
	Variable     string               // Variable name (without $ prefix)
	Type         DataType             // Variable type
	InitialValue Expression           // Optional initial value
	Annotations  *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *DeclareStmt) isMicroflowStatement() {}

// EnumSplitCase represents one enumeration branch in an EnumSplit.
type EnumSplitCase struct {
	Value  string // First enumeration value, or "(empty)" for Mendix's empty enum case.
	Values []string
	Body   []MicroflowStatement
}

// EnumSplitStmt represents: CASE $Var WHEN value THEN body ... END CASE
type EnumSplitStmt struct {
	Variable    string // Variable or attribute path without $ prefix (e.g. EventType or Event/EventType)
	Cases       []EnumSplitCase
	ElseBody    []MicroflowStatement
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

// InheritanceSplitCase represents one typed branch in an inheritance split.
type InheritanceSplitCase struct {
	Entity QualifiedName
	Body   []MicroflowStatement
}

// InheritanceSplitStmt represents: SPLIT TYPE $Var ... END SPLIT
type InheritanceSplitStmt struct {
	Variable    string // Variable name without $ prefix
	Cases       []InheritanceSplitCase
	ElseBody    []MicroflowStatement
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *EnumSplitStmt) isMicroflowStatement()        {}
func (s *InheritanceSplitStmt) isMicroflowStatement() {}

// CastObjectStmt represents: $Output = CAST $Object
type CastObjectStmt struct {
	OutputVariable string               // Output variable name without $ prefix
	ObjectVariable string               // Source object variable name without $ prefix
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CastObjectStmt) isMicroflowStatement() {}

// MfSetStmt represents: SET $Var = expr or SET $Var/Attr = expr
// (Named MfSetStmt to avoid conflict with existing SetStmt for SET key = value)
type MfSetStmt struct {
	Target      string               // Variable name or attribute path
	Value       Expression           // Value to assign
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *MfSetStmt) isMicroflowStatement() {}

// ReturnStmt represents: RETURN [expr]
type ReturnStmt struct {
	Value       Expression           // Optional return value
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ReturnStmt) isMicroflowStatement() {}

// RaiseErrorStmt represents: RAISE ERROR
// Used in custom error handlers to terminate with an ErrorEvent instead of merging back.
type RaiseErrorStmt struct {
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *RaiseErrorStmt) isMicroflowStatement() {}

// AnchorSide identifies one of the four sides of an activity's visual box
// that a SequenceFlow attaches to.
type AnchorSide int

const (
	AnchorSideUnset  AnchorSide = -1
	AnchorSideTop    AnchorSide = 0
	AnchorSideRight  AnchorSide = 1
	AnchorSideBottom AnchorSide = 2
	AnchorSideLeft   AnchorSide = 3
)

// FlowAnchors captures the origin/destination anchors for a single
// SequenceFlow. Each side is independently optional: Unset means the builder
// should derive the anchor from the visual direction.
type FlowAnchors struct {
	From AnchorSide // OriginConnectionIndex on the outgoing SequenceFlow
	To   AnchorSide // DestinationConnectionIndex on the outgoing SequenceFlow
}

// ActivityAnnotations holds metadata annotations for microflow activities.
// These are emitted as @position, @caption, @color, @annotation, @excluded, @anchor lines in MDL.
type ActivityAnnotations struct {
	Position        *Position    // @position(x, y)
	Caption         string       // @caption 'text'
	Color           string       // @color Green
	AnnotationText  string       // @annotation 'text'
	FreeAnnotations []string     // Multiple free-floating @annotation lines in source order
	Excluded        bool         // @excluded
	Anchor          *FlowAnchors // @anchor(from: X, to: Y) — anchors of the flow leaving this statement

	// Split-specific anchors for IF statements. When the statement is not an
	// IF these remain nil. The grammar accepts them on IfStmt only:
	//   @anchor(true: (from: right, to: left), false: (from: bottom, to: left))
	TrueBranchAnchor  *FlowAnchors
	FalseBranchAnchor *FlowAnchors

	// Loop body anchors for LOOP/WHILE. IteratorAnchor is the flow that
	// enters the loop body from the iterator; BodyTailAnchor is the flow
	// from the last body statement back to the loop boundary. Both are only
	// populated on LoopStmt/WhileStmt.
	IteratorAnchor *FlowAnchors
	BodyTailAnchor *FlowAnchors
}

// ChangeItem represents a single assignment in CREATE/CHANGE: Attr = expr
type ChangeItem struct {
	Attribute string     // Attribute name
	Value     Expression // Value expression
}

// CreateObjectStmt represents: $Var = CREATE Entity (assignments) [ON ERROR ...]
type CreateObjectStmt struct {
	Variable      string               // Variable name (without $ prefix)
	EntityType    QualifiedName        // Entity type
	Changes       []ChangeItem         // SET assignments
	ErrorHandling *ErrorHandlingClause // Optional ON ERROR clause
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CreateObjectStmt) isMicroflowStatement() {}

// ChangeObjectStmt represents: CHANGE $Var (assignments)
type ChangeObjectStmt struct {
	Variable        string               // Variable name
	Changes         []ChangeItem         // SET assignments
	RefreshInClient bool                 // Whether to refresh in client
	Annotations     *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ChangeObjectStmt) isMicroflowStatement() {}

// MfCommitStmt represents: COMMIT $Var [WITH EVENTS] [REFRESH] [ON ERROR ...]
type MfCommitStmt struct {
	Variable        string               // Variable to commit
	WithEvents      bool                 // Whether to trigger events
	RefreshInClient bool                 // Whether to refresh in client
	ErrorHandling   *ErrorHandlingClause // Optional ON ERROR clause
	Annotations     *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *MfCommitStmt) isMicroflowStatement() {}

// DeleteObjectStmt represents: DELETE $Var [ON ERROR ...]
type DeleteObjectStmt struct {
	Variable      string               // Variable to delete
	ErrorHandling *ErrorHandlingClause // Optional ON ERROR clause
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *DeleteObjectStmt) isMicroflowStatement() {}

// RollbackStmt represents: ROLLBACK $Var [REFRESH]
type RollbackStmt struct {
	Variable        string               // Variable to rollback
	RefreshInClient bool                 // Whether to refresh in client
	Annotations     *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *RollbackStmt) isMicroflowStatement() {}

// RetrieveStmt represents: RETRIEVE $Var FROM Entity [WHERE condition] [SORT BY ...] [LIMIT n] [OFFSET n] [ON ERROR ...]
// or: RETRIEVE $Var FROM $Parent/Module.Association (association retrieve)
type RetrieveStmt struct {
	Variable      string               // Output variable
	Source        QualifiedName        // Entity (database) or Association (association retrieve)
	StartVariable string               // Non-empty for association retrieve: the starting variable name
	Where         Expression           // Optional WHERE condition
	SortColumns   []SortColumnDef      // Optional SORT BY columns
	Limit         string               // Optional LIMIT expression (empty = no limit)
	Offset        string               // Optional OFFSET expression (empty = no offset)
	ErrorHandling *ErrorHandlingClause // Optional ON ERROR clause
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *RetrieveStmt) isMicroflowStatement() {}

// IfStmt represents: IF expr THEN body [ELSE body] END IF
type IfStmt struct {
	Condition   Expression           // IF condition
	ThenBody    []MicroflowStatement // THEN branch
	ElseBody    []MicroflowStatement // ELSE branch (optional)
	HasElse     bool                 // true when the source contained ELSE, even if the body is empty
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *IfStmt) isMicroflowStatement() {}

// LoopStmt represents: LOOP $Var IN $List BEGIN body END LOOP
type LoopStmt struct {
	LoopVariable string               // Iterator variable name
	ListVariable string               // List variable name
	Body         []MicroflowStatement // Loop body
	Annotations  *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *LoopStmt) isMicroflowStatement() {}

// WhileStmt represents: WHILE expr BEGIN body END WHILE
type WhileStmt struct {
	Condition   Expression           // WHILE condition expression
	Body        []MicroflowStatement // Loop body
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *WhileStmt) isMicroflowStatement() {}

// LogLevel represents the severity level for LOG statements.
type LogLevel int

const (
	LogTrace LogLevel = iota
	LogDebug
	LogInfo
	LogWarning
	LogError
	LogCritical
)

func (l LogLevel) String() string {
	switch l {
	case LogTrace:
		return "Trace"
	case LogDebug:
		return "Debug"
	case LogInfo:
		return "Info"
	case LogWarning:
		return "Warning"
	case LogError:
		return "Error"
	case LogCritical:
		return "Critical"
	default:
		return "Info"
	}
}

// TemplateParam represents a parameter in string template WITH clause: {1} = expr
// Used by LOG statements (microflows) and CONTENT/captions (pages).
// Supports both simple expressions and data source attribute references ($Widget.Attr).
type TemplateParam struct {
	Index          int        // Placeholder index (1, 2, 3, ...)
	Value          Expression // Value expression (for general expressions)
	DataSourceName string     // Widget name for $WidgetName.Attribute syntax (empty if not a DS ref)
	AttributeName  string     // Attribute name for $WidgetName.Attribute syntax
}

// IsDataSourceRef returns true if this is a data source attribute reference.
func (p *TemplateParam) IsDataSourceRef() bool {
	return p.DataSourceName != ""
}

// LogStmt represents: LOG LEVEL [NODE expr] message [WITH params]
type LogStmt struct {
	Level       LogLevel             // Log level (INFO, WARNING, etc.)
	Node        Expression           // Optional log node expression
	Message     Expression           // Message expression
	Template    []TemplateParam      // Optional WITH template params
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *LogStmt) isMicroflowStatement() {}

// CallArgument represents a parameter in CALL: name = expr
type CallArgument struct {
	Name  string     // Parameter name
	Value Expression // Value expression
}

// CallMicroflowStmt represents: [$Result =] CALL MICROFLOW Name (args) [ON ERROR ...]
type CallMicroflowStmt struct {
	OutputVariable string               // Optional output variable
	MicroflowName  QualifiedName        // Microflow to call
	Arguments      []CallArgument       // Arguments
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallMicroflowStmt) isMicroflowStatement() {}

// CallNanoflowStmt represents: [$Result =] CALL NANOFLOW Name (args) [ON ERROR ...]
type CallNanoflowStmt struct {
	OutputVariable string               // Optional output variable
	NanoflowName   QualifiedName        // Nanoflow to call
	Arguments      []CallArgument       // Arguments
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallNanoflowStmt) isMicroflowStatement() {}

// CallJavaActionStmt represents: CALL JAVA ACTION Name (args) [ON ERROR ...]
type CallJavaActionStmt struct {
	OutputVariable string               // Optional output variable
	ActionName     QualifiedName        // Java action name
	Arguments      []CallArgument       // Arguments
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallJavaActionStmt) isMicroflowStatement() {}

// CallJavaScriptActionStmt represents: CALL JAVASCRIPT ACTION Name (args) [ON ERROR ...]
type CallJavaScriptActionStmt struct {
	OutputVariable string               // Optional output variable
	ActionName     QualifiedName        // JavaScript action name
	Arguments      []CallArgument       // Arguments
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallJavaScriptActionStmt) isMicroflowStatement() {}

// CallWebServiceStmt represents a legacy SOAP web service call.
type CallWebServiceStmt struct {
	OutputVariable   string               // Optional output variable
	RawBSONBase64    string               // Raw Microflows$CallWebServiceAction BSON for lossless roundtrip
	ServiceID        string               // Consumed web service ID or qualified name
	OperationName    string               // Operation name
	SendMappingID    string               // Optional export mapping ID or qualified name
	ReceiveMappingID string               // Optional import mapping ID or qualified name
	Timeout          Expression           // Optional timeout expression
	ErrorHandling    *ErrorHandlingClause // Optional ON ERROR clause
	Annotations      *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallWebServiceStmt) isMicroflowStatement() {}

// ExecuteDatabaseQueryStmt represents: EXECUTE DATABASE QUERY Module.Connection.QueryName ...
type ExecuteDatabaseQueryStmt struct {
	OutputVariable      string               // Optional output variable
	QueryName           string               // Full 3-part identifier: Module.Connection.QueryName
	DynamicQuery        string               // Optional dynamic SQL override
	Arguments           []CallArgument       // Parameter mappings (query parameters)
	ConnectionArguments []CallArgument       // Connection parameter mappings (runtime connection override)
	ErrorHandling       *ErrorHandlingClause // Optional ON ERROR clause
	Annotations         *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ExecuteDatabaseQueryStmt) isMicroflowStatement() {}

// CallExternalActionStmt represents: CALL EXTERNAL ACTION Service.ActionName (args) [ON ERROR ...]
type CallExternalActionStmt struct {
	OutputVariable string               // Optional output variable
	ServiceName    QualifiedName        // Consumed OData service qualified name
	ActionName     string               // External action name
	Arguments      []CallArgument       // Arguments
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CallExternalActionStmt) isMicroflowStatement() {}

// BreakStmt represents: BREAK
type BreakStmt struct {
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *BreakStmt) isMicroflowStatement() {}

// ContinueStmt represents: CONTINUE
type ContinueStmt struct {
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ContinueStmt) isMicroflowStatement() {}

// ============================================================================
// List Operations
// ============================================================================

// ListOperationType represents the type of list operation.
type ListOperationType int

const (
	ListOpHead ListOperationType = iota
	ListOpTail
	ListOpFind
	ListOpFilter
	ListOpSort
	ListOpUnion
	ListOpIntersect
	ListOpSubtract
	ListOpContains
	ListOpEquals
	ListOpRange
)

func (t ListOperationType) String() string {
	switch t {
	case ListOpHead:
		return "HEAD"
	case ListOpTail:
		return "TAIL"
	case ListOpFind:
		return "FIND"
	case ListOpFilter:
		return "FILTER"
	case ListOpSort:
		return "SORT"
	case ListOpUnion:
		return "UNION"
	case ListOpIntersect:
		return "INTERSECT"
	case ListOpSubtract:
		return "SUBTRACT"
	case ListOpContains:
		return "CONTAINS"
	case ListOpEquals:
		return "EQUALS"
	case ListOpRange:
		return "RANGE"
	default:
		return "UNKNOWN"
	}
}

// SortSpec represents a sort specification: attr ASC/DESC
type SortSpec struct {
	Attribute string // Attribute name
	Ascending bool   // True for ASC, false for DESC
}

// ListOperationStmt represents list operations like HEAD, TAIL, FIND, etc.
// $Var = HEAD($List)
// $Var = FIND($List, condition)
// $Var = SORT($List, attr ASC)
// $Var = UNION($List1, $List2)
type ListOperationStmt struct {
	OutputVariable string               // Output variable name
	Operation      ListOperationType    // Operation type
	InputVariable  string               // Input list variable (first operand)
	SecondVariable string               // Second operand for UNION, INTERSECT, SUBTRACT, CONTAINS, EQUALS
	Condition      Expression           // Condition for FIND/FILTER
	SortSpecs      []SortSpec           // Sort specifications for SORT
	OffsetExpr     Expression           // Offset expression for RANGE
	LimitExpr      Expression           // Limit expression for RANGE
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ListOperationStmt) isMicroflowStatement() {}

// AggregateListOperationType represents the type of aggregate operation.
type AggregateListOperationType int

const (
	AggregateCount AggregateListOperationType = iota
	AggregateSum
	AggregateAverage
	AggregateMinimum
	AggregateMaximum
	AggregateReduce
)

func (t AggregateListOperationType) String() string {
	switch t {
	case AggregateCount:
		return "COUNT"
	case AggregateSum:
		return "SUM"
	case AggregateAverage:
		return "AVERAGE"
	case AggregateMinimum:
		return "MINIMUM"
	case AggregateMaximum:
		return "MAXIMUM"
	case AggregateReduce:
		return "REDUCE"
	default:
		return "UNKNOWN"
	}
}

// AggregateListStmt represents aggregate operations: COUNT, SUM, AVERAGE, etc.
// $Count = COUNT($List)
// $Sum = SUM($List/Attr)
// $Sum = SUM($List, $currentObject/Price * 2)  // expression form
type AggregateListStmt struct {
	OutputVariable string                     // Output variable name
	Operation      AggregateListOperationType // Operation type
	InputVariable  string                     // Input list variable
	Attribute      string                     // Attribute name for SUM/AVG/MIN/MAX (empty for COUNT or expression form)
	IsExpression   bool                       // true when Expression is used instead of Attribute
	Expression     Expression                 // Mendix expression (when IsExpression=true)
	Annotations    *ActivityAnnotations       // Optional @position, @caption, @color, @annotation
}

func (s *AggregateListStmt) isMicroflowStatement() {}

// CreateListStmt represents: $Var = CREATE LIST OF Entity
type CreateListStmt struct {
	Variable    string               // Output variable name
	EntityType  QualifiedName        // Entity type for the list
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *CreateListStmt) isMicroflowStatement() {}

// AddToListStmt represents: ADD expr TO $List
type AddToListStmt struct {
	Item        string               // Item variable to add, kept for simple $Var compatibility
	Value       Expression           // Item expression to add
	List        string               // Target list variable
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *AddToListStmt) isMicroflowStatement() {}

// RemoveFromListStmt represents: REMOVE $Item FROM $List
type RemoveFromListStmt struct {
	Item        string               // Item variable to remove
	List        string               // Source list variable
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *RemoveFromListStmt) isMicroflowStatement() {}

// ============================================================================
// Page Actions
// ============================================================================

// ShowPageArg represents a page parameter argument: $Param = $Value
type ShowPageArg struct {
	ParamName string     // Parameter name (without $ prefix)
	Value     Expression // Value expression
}

// ShowPageStmt represents: SHOW PAGE Module.Page($param = $value) [FOR $obj] [WITH (settings)]
type ShowPageStmt struct {
	PageName    QualifiedName        // Page to show
	Arguments   []ShowPageArg        // Page parameter arguments
	ForObject   string               // Optional FOR variable (without $ prefix)
	Title       string               // Optional title override
	Location    string               // Optional location: Content, Popup, Modal (default: Content)
	ModalForm   bool                 // Whether to show as modal
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ShowPageStmt) isMicroflowStatement() {}

// ClosePageStmt represents: CLOSE PAGE
type ClosePageStmt struct {
	NumberOfPages int                  // Number of pages to close (default 1)
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ClosePageStmt) isMicroflowStatement() {}

// ShowHomePageStmt represents: SHOW HOME PAGE
type ShowHomePageStmt struct {
	Annotations *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ShowHomePageStmt) isMicroflowStatement() {}

// ShowMessageStmt represents: SHOW MESSAGE 'text' TYPE Information OBJECTS [$Var1, $Var2];
type ShowMessageStmt struct {
	Message      Expression           // The message text (string template)
	Type         string               // Information, Warning, Error (default: Information)
	TemplateArgs []Expression         // Template arguments for message placeholders {1}, {2}, etc.
	Annotations  *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ShowMessageStmt) isMicroflowStatement() {}

// DownloadFileStmt represents: DOWNLOAD FILE $FileDocument [SHOW IN BROWSER]
type DownloadFileStmt struct {
	FileDocument  string               // File document variable without $ prefix
	ShowInBrowser bool                 // Whether the file opens in the browser
	ErrorHandling *ErrorHandlingClause // Optional ON ERROR clause
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *DownloadFileStmt) isMicroflowStatement() {}

// ValidationFeedbackStmt represents: VALIDATION FEEDBACK $Var/Attr MESSAGE 'message' OBJECTS [$Var1, $Var2];
type ValidationFeedbackStmt struct {
	AttributePath *AttributePathExpr   // The attribute to associate with the feedback
	Message       Expression           // The feedback message (string template)
	TemplateArgs  []Expression         // Template arguments for message placeholders
	Annotations   *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ValidationFeedbackStmt) isMicroflowStatement() {}

// ============================================================================
// REST Call Statements
// ============================================================================

// HttpMethod represents an HTTP method for REST calls.
type HttpMethod string

const (
	HttpMethodGet    HttpMethod = "Get"
	HttpMethodPost   HttpMethod = "Post"
	HttpMethodPut    HttpMethod = "Put"
	HttpMethodPatch  HttpMethod = "Patch"
	HttpMethodDelete HttpMethod = "Delete"
)

// RestHeader represents a custom HTTP header: HEADER name = value
type RestHeader struct {
	Name  string     // Header name (e.g., "Accept", "Content-Type")
	Value Expression // Header value expression
}

// RestAuth represents HTTP authentication configuration.
type RestAuth struct {
	Username Expression // Username expression
	Password Expression // Password expression
}

// RestBodyType represents the type of request body handling.
type RestBodyType int

const (
	RestBodyNone    RestBodyType = iota // No body
	RestBodyCustom                      // Custom body template
	RestBodyMapping                     // Export mapping
)

// RestBody represents the request body configuration.
type RestBody struct {
	Type           RestBodyType    // Body type
	Template       Expression      // Body template (for Custom type)
	TemplateParams []TemplateParam // Template parameters for placeholders
	MappingName    QualifiedName   // Export mapping name (for Mapping type)
	SourceVariable string          // Source variable for mapping
}

// RestResultType represents how the response should be handled.
type RestResultType int

const (
	RestResultString   RestResultType = iota // Return as string
	RestResultResponse                       // Return HttpResponse object
	RestResultMapping                        // Use import mapping
	RestResultNone                           // Ignore response
)

// RestResult represents the response handling configuration.
type RestResult struct {
	Type         RestResultType // Result type
	MappingName  QualifiedName  // Import mapping name (for Mapping type)
	ResultEntity QualifiedName  // Result entity type (for Mapping type)
	// IsList distinguishes `as Module.Entity` (single object) from
	// `as list of Module.Entity` (list). Studio Pro stores this on the
	// microflow's ImportMappingCall (Range.SingleObject /
	// ForceSingleOccurrence), independently of whether the underlying
	// import mapping is list-typed: the same mapping can yield either a
	// single object or a list depending on this flag.
	IsList bool
}

// RestCallStmt represents: $Var = REST CALL METHOD url [HEADER ...] [AUTH ...] [BODY ...] [TIMEOUT ...] RETURNS ...
type RestCallStmt struct {
	OutputVariable string               // Optional output variable
	Method         HttpMethod           // HTTP method (GET, POST, PUT, PATCH, DELETE)
	URL            Expression           // URL expression (string literal or expression)
	URLParams      []TemplateParam      // URL template parameters
	Headers        []RestHeader         // Custom HTTP headers
	Auth           *RestAuth            // Optional authentication
	Body           *RestBody            // Optional request body
	Timeout        Expression           // Optional timeout expression (seconds)
	Result         RestResult           // Response handling
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *RestCallStmt) isMicroflowStatement() {}

// SendRestRequestStmt represents: [$Var =] SEND REST REQUEST Module.Service.Operation [BODY $var] [ON ERROR ...]
// Calls a consumed REST service operation defined via CREATE REST CLIENT.
type SendRestRequestStmt struct {
	OutputVariable string               // Optional output variable (without $)
	Operation      QualifiedName        // Consumed REST service operation (Module.Service.Operation)
	Parameters     []SendRestParamDef   // Parameter bindings from WITH clause
	BodyVariable   string               // Optional body variable name (without $)
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

// SendRestParamDef represents a parameter binding: $paramName = expression
type SendRestParamDef struct {
	Name       string // parameter name (without $)
	Expression string // Mendix expression
}

func (s *SendRestRequestStmt) isMicroflowStatement() {}

// ImportFromMappingStmt represents: [$Var =] IMPORT FROM MAPPING Module.IMM($SourceVar)
type ImportFromMappingStmt struct {
	OutputVariable string               // Optional result variable (without $)
	Mapping        QualifiedName        // Import mapping qualified name
	SourceVariable string               // Input string variable (without $)
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ImportFromMappingStmt) isMicroflowStatement() {}

// ExportToMappingStmt represents: $Var = EXPORT TO MAPPING Module.EMM($SourceVar)
type ExportToMappingStmt struct {
	OutputVariable string               // Result string variable (without $)
	Mapping        QualifiedName        // Export mapping qualified name
	SourceVariable string               // Input entity variable (without $)
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *ExportToMappingStmt) isMicroflowStatement() {}

// TransformJsonStmt represents: $Result = TRANSFORM $Input WITH Module.Transformer
type TransformJsonStmt struct {
	OutputVariable string               // Result string variable (without $)
	InputVariable  string               // Source JSON string variable (without $)
	Transformation QualifiedName        // Data transformer qualified name
	ErrorHandling  *ErrorHandlingClause // Optional ON ERROR clause
	Annotations    *ActivityAnnotations // Optional @position, @caption, @color, @annotation
}

func (s *TransformJsonStmt) isMicroflowStatement() {}
