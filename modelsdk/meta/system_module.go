// SPDX-License-Identifier: Apache-2.0

package meta

// System module constants — deterministic IDs for the virtual System module.
const (
	SystemModuleID      = "00000000-0000-0000-0000-000000000001"
	SystemDomainModelID = "00000000-0000-0000-0000-000000000002"
)

// SystemAttrDef defines an attribute in a System entity.
type SystemAttrDef struct {
	Name   string
	Type   string // "String", "Integer", "Decimal", "Boolean", "DateTime", "Enumeration", "Long", "Binary", "HashedString", "AutoNumber"
	Length int    // for String type
	EnumQN string // for Enumeration type, qualified name
}

// SystemAssocDef defines an association between System entities.
type SystemAssocDef struct {
	Name   string
	Parent string // parent entity name (without module prefix)
	Child  string // child entity name (without module prefix)
	Type   string // "Reference" or "ReferenceSet"
	Owner  string // "Default" or "Both"
}

// SystemEntityDef defines a System entity with name, persistability, and attributes.
type SystemEntityDef struct {
	Name           string
	Persistable    bool
	Generalization string // e.g. "System.FileDocument", "System.Error"
	Attributes     []SystemAttrDef
}

// SystemEntities lists all entities in the System module.
// Extracted from Mendix Studio Pro 11.6.4 via DummySystem module.
var SystemEntities = []SystemEntityDef{
	{Name: "UserRole", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "ModelGUID", Type: "String"},
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
	}},
	{Name: "User", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Password", Type: "HashedString"},
		{Name: "LastLogin", Type: "DateTime"},
		{Name: "Blocked", Type: "Boolean"},
		{Name: "BlockedSince", Type: "DateTime"},
		{Name: "Active", Type: "Boolean"},
		{Name: "FailedLogins", Type: "Integer"},
		{Name: "WebServiceUser", Type: "Boolean"},
		{Name: "IsAnonymous", Type: "Boolean"},
	}},
	{Name: "FileDocument", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "FileID", Type: "AutoNumber"},
		{Name: "Name", Type: "String"},
		{Name: "DeleteAfterDownload", Type: "Boolean"},
		{Name: "Contents", Type: "Binary"},
		{Name: "HasContents", Type: "Boolean"},
		{Name: "Size", Type: "Long"},
	}},
	{Name: "Image", Persistable: true, Generalization: "System.FileDocument", Attributes: []SystemAttrDef{
		{Name: "PublicThumbnailPath", Type: "String"},
		{Name: "EnableCaching", Type: "Boolean"},
	}},
	{Name: "XASInstance", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "XASId", Type: "String"},
		{Name: "LastUpdate", Type: "DateTime"},
		{Name: "AllowedNumberOfConcurrentUsers", Type: "Integer"},
		{Name: "PartnerName", Type: "String"},
		{Name: "CustomerName", Type: "String"},
	}},
	{Name: "Session", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "SessionId", Type: "String"},
		{Name: "CSRFToken", Type: "String"},
		{Name: "LastActive", Type: "DateTime"},
	}},
	{Name: "ScheduledEventInformation", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Status", Type: "Enumeration", EnumQN: "System.EventStatus"},
	}},
	{Name: "Language", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Code", Type: "String"},
		{Name: "Description", Type: "String"},
	}},
	{Name: "TimeZone", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Code", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "RawOffset", Type: "Integer"},
	}},
	{Name: "Error", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "ErrorType", Type: "String"},
		{Name: "Message", Type: "String"},
		{Name: "Stacktrace", Type: "String"},
	}},
	{Name: "SoapFault", Persistable: true, Generalization: "System.Error", Attributes: []SystemAttrDef{
		{Name: "Code", Type: "String"},
		{Name: "Reason", Type: "String"},
		{Name: "Node", Type: "String"},
		{Name: "Role", Type: "String"},
		{Name: "Detail", Type: "String"},
	}},
	{Name: "TokenInformation", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Token", Type: "HashedString"},
		{Name: "ExpiryDate", Type: "DateTime"},
		{Name: "UserAgent", Type: "String"},
	}},
	{Name: "HttpMessage", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "HttpVersion", Type: "String"},
		{Name: "Content", Type: "String"},
	}},
	{Name: "HttpHeader", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "Key", Type: "String"},
		{Name: "Value", Type: "String"},
	}},
	{Name: "UserReportInfo", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "UserType", Type: "Enumeration", EnumQN: "System.UserType"},
		{Name: "Hash", Type: "String"},
	}},
	{Name: "HttpRequest", Persistable: true, Generalization: "System.HttpMessage", Attributes: []SystemAttrDef{
		{Name: "Uri", Type: "String"},
	}},
	{Name: "HttpResponse", Persistable: true, Generalization: "System.HttpMessage", Attributes: []SystemAttrDef{
		{Name: "StatusCode", Type: "Integer"},
		{Name: "ReasonPhrase", Type: "String"},
	}},
	{Name: "Paging", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "PageNumber", Type: "Long"},
		{Name: "IsSortable", Type: "Boolean"},
		{Name: "SortAttribute", Type: "String"},
		{Name: "SortAscending", Type: "Boolean"},
		{Name: "HasMoreData", Type: "Boolean"},
	}},
	{Name: "SynchronizationError", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Reason", Type: "String"},
		{Name: "ObjectId", Type: "String"},
		{Name: "ObjectType", Type: "String"},
		{Name: "ObjectContent", Type: "String"},
	}},
	{Name: "SynchronizationErrorFile", Persistable: true, Generalization: "System.FileDocument"},
	{Name: "ProcessedQueueTask", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Sequence", Type: "Long"},
		{Name: "Status", Type: "Enumeration", EnumQN: "System.QueueTaskStatus"},
		{Name: "QueueId", Type: "String"},
		{Name: "QueueName", Type: "String"},
		{Name: "ContextType", Type: "Enumeration", EnumQN: "System.ContextType"},
		{Name: "ContextData", Type: "String"},
		{Name: "MicroflowName", Type: "String"},
		{Name: "UserActionName", Type: "String"},
		{Name: "Arguments", Type: "String"},
		{Name: "XASId", Type: "String"},
		{Name: "ThreadId", Type: "Long"},
		{Name: "Created", Type: "DateTime"},
		{Name: "StartAt", Type: "DateTime"},
		{Name: "Started", Type: "DateTime"},
		{Name: "Finished", Type: "DateTime"},
		{Name: "Duration", Type: "Long"},
		{Name: "Retried", Type: "Long"},
		{Name: "ErrorMessage", Type: "String"},
		{Name: "ScheduledEventName", Type: "String"},
	}},
	{Name: "QueuedTask", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Sequence", Type: "AutoNumber"},
		{Name: "Status", Type: "Enumeration", EnumQN: "System.QueueTaskStatus"},
		{Name: "QueueId", Type: "String"},
		{Name: "QueueName", Type: "String"},
		{Name: "ContextType", Type: "Enumeration", EnumQN: "System.ContextType"},
		{Name: "ContextData", Type: "String"},
		{Name: "MicroflowName", Type: "String"},
		{Name: "UserActionName", Type: "String"},
		{Name: "Arguments", Type: "String"},
		{Name: "XASId", Type: "String"},
		{Name: "ThreadId", Type: "Long"},
		{Name: "Created", Type: "DateTime"},
		{Name: "StartAt", Type: "DateTime"},
		{Name: "Started", Type: "DateTime"},
		{Name: "Retried", Type: "Long"},
		{Name: "Retry", Type: "String"},
		{Name: "ScheduledEventName", Type: "String"},
	}},
	{Name: "WorkflowDefinition", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Title", Type: "String"},
		{Name: "IsObsolete", Type: "Boolean"},
		{Name: "IsLocked", Type: "Boolean"},
	}},
	{Name: "WorkflowUserTaskDefinition", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "IsObsolete", Type: "Boolean"},
	}},
	{Name: "Workflow", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "DueDate", Type: "DateTime"},
		{Name: "CanBeRestarted", Type: "Boolean"},
		{Name: "CanBeContinued", Type: "Boolean"},
		{Name: "CanApplyJumpTo", Type: "Boolean"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowState"},
		{Name: "Reason", Type: "String"},
	}},
	{Name: "WorkflowUserTask", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "DueDate", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Outcome", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowUserTaskState"},
		{Name: "CompletionType", Type: "Enumeration", EnumQN: "System.WorkflowUserTaskCompletionType"},
	}},
	{Name: "TaskQueueToken", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "QueueName", Type: "String"},
		{Name: "XASId", Type: "String"},
		{Name: "ValidUntil", Type: "DateTime"},
	}},
	{Name: "ODataResponse", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "Count", Type: "Long"},
	}},
	{Name: "WorkflowJumpToDetails", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "Error", Type: "String"},
	}},
	{Name: "WorkflowCurrentActivity", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "Action", Type: "Enumeration", EnumQN: "System.WorkflowCurrentActivityAction"},
	}},
	{Name: "WorkflowActivityDetails", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "ActivityId", Type: "String"},
		{Name: "ActivityCaption", Type: "String"},
		{Name: "ActivityType", Type: "Enumeration", EnumQN: "System.WorkflowActivityType"},
		{Name: "ExistsInCurrentVersion", Type: "Boolean"},
	}},
	{Name: "WorkflowUserTaskOutcome", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Outcome", Type: "String"},
		{Name: "Time", Type: "DateTime"},
	}},
	{Name: "WorkflowRecord", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "WorkflowKey", Type: "String"},
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowState"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "DueDate", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Reason", Type: "String"},
	}},
	{Name: "WorkflowActivityRecord", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "ModelGUID", Type: "String"},
		{Name: "ActivityKey", Type: "String"},
		{Name: "PreviousActivityKey", Type: "String"},
		{Name: "ActivityType", Type: "Enumeration", EnumQN: "System.WorkflowActivityType"},
		{Name: "Caption", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowActivityExecutionState"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Outcome", Type: "String"},
		{Name: "MicroflowName", Type: "String"},
		{Name: "TaskName", Type: "String"},
		{Name: "TaskDescription", Type: "String"},
		{Name: "TaskDueDate", Type: "DateTime"},
		{Name: "TaskCompletionType", Type: "Enumeration", EnumQN: "System.WorkflowUserTaskCompletionType"},
		{Name: "TaskRequiredUsers", Type: "Integer"},
		{Name: "TaskKey", Type: "String"},
		{Name: "Reason", Type: "String"},
	}},
	{Name: "WorkflowEvent", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "EventTime", Type: "DateTime"},
		{Name: "EventType", Type: "Enumeration", EnumQN: "System.WorkflowEventType"},
	}},
	{Name: "ConsumedODataConfiguration", Persistable: false, Attributes: []SystemAttrDef{
		{Name: "ServiceUrl", Type: "String"},
		{Name: "ProxyConfiguration", Type: "Enumeration", EnumQN: "System.ProxyConfiguration"},
		{Name: "ProxyHost", Type: "String"},
		{Name: "ProxyPort", Type: "Integer"},
		{Name: "ProxyUsername", Type: "String"},
		{Name: "ProxyPassword", Type: "String"},
	}},
	{Name: "WorkflowEndedUserTask", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "DueDate", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Outcome", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowUserTaskState"},
		{Name: "CompletionType", Type: "Enumeration", EnumQN: "System.WorkflowUserTaskCompletionType"},
		{Name: "UserTaskKey", Type: "String"},
	}},
	{Name: "WorkflowEndedUserTaskOutcome", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Outcome", Type: "String"},
		{Name: "Time", Type: "DateTime"},
	}},
	{Name: "WorkflowGroup", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Name", Type: "String"},
		{Name: "Description", Type: "String"},
	}},
	// --- Entities below extracted from MDP (Phase 3, 2026-04-24) ---
	{Name: "WorkflowVersion", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "VersionHash", Type: "String"},
		{Name: "ModelJSON", Type: "String"},
	}},
	{Name: "WorkflowActivity", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "ModelGUID", Type: "String"},
		{Name: "ActivityGUID", Type: "String"},
		{Name: "Caption", Type: "String"},
		{Name: "DetailsJson", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.WorkflowActivityState"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "ActionTime", Type: "DateTime"},
		{Name: "Reason", Type: "String"},
		{Name: "ActivityHash", Type: "String"},
		{Name: "IsDerivedActivity", Type: "Boolean"},
		{Name: "Outcome", Type: "String"},
		{Name: "OutcomeModelGUID", Type: "String"},
	}},
	{Name: "WorkflowActivityUserTaskOutcome", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Outcome", Type: "String"},
		{Name: "Time", Type: "DateTime"},
	}},
	{Name: "PrivateFileDocument", Persistable: true, Generalization: "System.FileDocument"},
	{Name: "Thumbnail", Persistable: true, Generalization: "System.Image"},
	{Name: "BackgroundJob", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "JobId", Type: "Long"},
		{Name: "StartTime", Type: "DateTime"},
		{Name: "EndTime", Type: "DateTime"},
		{Name: "Result", Type: "String"},
		{Name: "Successful", Type: "Boolean"},
	}},
	{Name: "AutoCommitEntry", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "SessionId", Type: "String"},
		{Name: "ObjectId", Type: "Long"},
	}},
	{Name: "UnreferencedFile", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "FileKey", Type: "String"},
		{Name: "State", Type: "Enumeration", EnumQN: "System.UnreferencedFileState"},
		{Name: "TransactionId", Type: "String"},
	}},
	{Name: "OfflineCreatedGuids", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "Guid", Type: "String"},
	}},
	{Name: "OfflineSynchronizationHistory", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "SyncId", Type: "String"},
	}},
	{Name: "ChangeHash", Persistable: true, Attributes: []SystemAttrDef{
		{Name: "ObjectId", Type: "Long"},
		{Name: "Attribute", Type: "String"},
		{Name: "Hash", Type: "String"},
	}},
}

// SystemAssociations lists all associations in the System module.
// Extracted from Mendix Studio Pro 11.6.4 via DummySystem module.
var SystemAssociations = []SystemAssocDef{
	{Name: "grantableRoles", Parent: "UserRole", Child: "UserRole", Type: "ReferenceSet", Owner: "Default"},
	{Name: "UserRoles", Parent: "User", Child: "UserRole", Type: "ReferenceSet", Owner: "Default"},
	{Name: "Session_User", Parent: "Session", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "User_Language", Parent: "User", Child: "Language", Type: "Reference", Owner: "Default"},
	{Name: "User_TimeZone", Parent: "User", Child: "TimeZone", Type: "Reference", Owner: "Default"},
	{Name: "TokenInformation_User", Parent: "TokenInformation", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "HttpHeaders", Parent: "HttpHeader", Child: "HttpMessage", Type: "Reference", Owner: "Default"},
	{Name: "UserReportInfo_User", Parent: "UserReportInfo", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "ScheduledEventInformation_XASInstance", Parent: "ScheduledEventInformation", Child: "XASInstance", Type: "Reference", Owner: "Default"},
	{Name: "SynchronizationErrorFile_SynchronizationError", Parent: "SynchronizationErrorFile", Child: "SynchronizationError", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowUserTaskDefinition_WorkflowDefinition", Parent: "WorkflowUserTaskDefinition", Child: "WorkflowDefinition", Type: "Reference", Owner: "Default"},
	{Name: "Workflow_WorkflowDefinition", Parent: "Workflow", Child: "WorkflowDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowUserTask_TargetUsers", Parent: "WorkflowUserTask", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowUserTask_Assignees", Parent: "WorkflowUserTask", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowUserTask_Workflow", Parent: "WorkflowUserTask", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowUserTask_WorkflowUserTaskDefinition", Parent: "WorkflowUserTask", Child: "WorkflowUserTaskDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowJumpToDetails_Workflow", Parent: "WorkflowJumpToDetails", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowJumpToDetails_CurrentActivities", Parent: "WorkflowJumpToDetails", Child: "WorkflowCurrentActivity", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowCurrentActivity_ActivityDetails", Parent: "WorkflowCurrentActivity", Child: "WorkflowActivityDetails", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowCurrentActivity_ApplicableTargets", Parent: "WorkflowCurrentActivity", Child: "WorkflowActivityDetails", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowCurrentActivity_JumpToTarget", Parent: "WorkflowCurrentActivity", Child: "WorkflowActivityDetails", Type: "Reference", Owner: "Default"},
	{Name: "Workflow_ParentWorkflow", Parent: "Workflow", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowUserTaskOutcome_WorkflowUserTask", Parent: "WorkflowUserTaskOutcome", Child: "WorkflowUserTask", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowUserTaskOutcome_User", Parent: "WorkflowUserTaskOutcome", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowRecord_Workflow", Parent: "WorkflowRecord", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowRecord_Owner", Parent: "WorkflowRecord", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowRecord_WorkflowDefinition", Parent: "WorkflowRecord", Child: "WorkflowDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_PreviousActivity", Parent: "WorkflowActivityRecord", Child: "WorkflowActivityRecord", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_Actor", Parent: "WorkflowActivityRecord", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_SubWorkflow", Parent: "WorkflowActivityRecord", Child: "WorkflowRecord", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_UserTask", Parent: "WorkflowActivityRecord", Child: "WorkflowUserTask", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_WorkflowUserTaskDefinition", Parent: "WorkflowActivityRecord", Child: "WorkflowUserTaskDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowEvent_Initiator", Parent: "WorkflowEvent", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityRecord_TaskTargetedUsers", Parent: "WorkflowActivityRecord", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowActivityRecord_TaskAssignedUsers", Parent: "WorkflowActivityRecord", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "HttpHeader_ConsumedODataConfiguration", Parent: "HttpHeader", Child: "ConsumedODataConfiguration", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowEndedUserTask_Assignees", Parent: "WorkflowEndedUserTask", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowEndedUserTask_TargetUsers", Parent: "WorkflowEndedUserTask", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowEndedUserTask_WorkflowUserTaskDefinition", Parent: "WorkflowEndedUserTask", Child: "WorkflowUserTaskDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowEndedUserTask_Workflow", Parent: "WorkflowEndedUserTask", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowEndedUserTaskOutcome_User", Parent: "WorkflowEndedUserTaskOutcome", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowEndedUserTaskOutcome_WorkflowEndedUserTask", Parent: "WorkflowEndedUserTaskOutcome", Child: "WorkflowEndedUserTask", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowGroup_User", Parent: "WorkflowGroup", Child: "User", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowUserTask_TargetGroups", Parent: "WorkflowUserTask", Child: "WorkflowGroup", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowEndedUserTask_TargetGroups", Parent: "WorkflowEndedUserTask", Child: "WorkflowGroup", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowActivityRecord_TaskTargetedGroups", Parent: "WorkflowActivityRecord", Child: "WorkflowGroup", Type: "ReferenceSet", Owner: "Default"},
	// --- Associations below extracted from MDP (Phase 3, 2026-04-24) ---
	{Name: "WorkflowVersion_WorkflowDefinition", Parent: "WorkflowVersion", Child: "WorkflowDefinition", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowVersion_WorkflowUserTaskDefinition", Parent: "WorkflowVersion", Child: "WorkflowUserTaskDefinition", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowVersion_PreviousVersion", Parent: "WorkflowVersion", Child: "WorkflowVersion", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowDefinition_CurrentWorkflowVersion", Parent: "WorkflowDefinition", Child: "WorkflowVersion", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivity_Workflow", Parent: "WorkflowActivity", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivity_WorkflowUserTask", Parent: "WorkflowActivity", Child: "WorkflowUserTask", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivity_PreviousActivity", Parent: "WorkflowActivity", Child: "WorkflowActivity", Type: "ReferenceSet", Owner: "Default"},
	{Name: "Workflow_CurrentActivity", Parent: "Workflow", Child: "WorkflowActivity", Type: "ReferenceSet", Owner: "Default"},
	{Name: "WorkflowActivity_WorkflowVersion", Parent: "WorkflowActivity", Child: "WorkflowVersion", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivity_Actor", Parent: "WorkflowActivity", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivity_SubWorkflow", Parent: "WorkflowActivity", Child: "Workflow", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityUserTaskOutcome_WorkflowActivity", Parent: "WorkflowActivityUserTaskOutcome", Child: "WorkflowActivity", Type: "Reference", Owner: "Default"},
	{Name: "WorkflowActivityUserTaskOutcome_User", Parent: "WorkflowActivityUserTaskOutcome", Child: "User", Type: "Reference", Owner: "Default"},
	{Name: "Thumbnail_Image", Parent: "Thumbnail", Child: "Image", Type: "Reference", Owner: "Both"},
	{Name: "BackgroundJob_Session", Parent: "BackgroundJob", Child: "Session", Type: "Reference", Owner: "Default"},
	{Name: "BackgroundJob_XASInstance", Parent: "BackgroundJob", Child: "XASInstance", Type: "Reference", Owner: "Default"},
	{Name: "UnreferencedFile_XASInstance", Parent: "UnreferencedFile", Child: "XASInstance", Type: "Reference", Owner: "Default"},
	{Name: "ChangeHash_Session", Parent: "ChangeHash", Child: "Session", Type: "Reference", Owner: "Default"},
}

// SystemEnumDef defines an enumeration in the System module.
type SystemEnumDef struct {
	Name   string   // qualified name, e.g. "System.EventStatus"
	Values []string // enumeration value names
}

// SystemEnumerations lists all enumerations in the System module.
// Extracted from Mendix mxbuild 11.6.4 MDP output.
var SystemEnumerations = []SystemEnumDef{
	{Name: "System.ContextType", Values: []string{
		"System", "User", "Anonymous", "ScheduledEvent",
	}},
	{Name: "System.EventStatus", Values: []string{
		"Running", "Completed", "Error", "Stopped",
	}},
	{Name: "System.DeviceType", Values: []string{
		"Phone", "Tablet", "Desktop",
	}},
	{Name: "System.UserType", Values: []string{
		"Internal", "External",
	}},
	{Name: "System.ProxyConfiguration", Values: []string{
		"UseAppSettings", "Override", "NoProxy",
	}},
	{Name: "System.QueueTaskStatus", Values: []string{
		"Idle", "Running", "Completed", "Failed", "Retrying", "Aborted", "Incompatible",
	}},
	{Name: "System.WorkflowActivityState", Values: []string{
		"Started", "Suspended", "Finished", "Replaced", "Aborted", "Failed",
	}},
	{Name: "System.UnreferencedFileState", Values: []string{
		"New", "Obsolete", "Deleted",
	}},
	{Name: "System.WorkflowActivityExecutionState", Values: []string{
		"Created", "InProgress", "Completed", "Paused", "Aborted", "Failed",
	}},
	{Name: "System.WorkflowState", Values: []string{
		"InProgress", "Paused", "Completed", "Aborted", "Incompatible", "Failed",
	}},
	{Name: "System.WorkflowUserTaskState", Values: []string{
		"Created", "InProgress", "Completed", "Paused", "Aborted", "Failed",
	}},
	{Name: "System.WorkflowActivityType", Values: []string{
		"Start", "End", "ExclusiveSplit", "ParallelSplit",
		"ParallelSplitBranchStopper", "ParallelSplitMerge",
		"UserTask", "CallMicroflow", "CallWorkflow", "JumpTo",
		"MultiInputUserTask", "WaitForNotification", "WaitForTimer",
		"EndOfBoundaryEventPath", "NonInterruptingTimerEvent", "InterruptingTimerEvent",
	}},
	{Name: "System.WorkflowCurrentActivityAction", Values: []string{
		"DoNothing", "JumpTo",
	}},
	{Name: "System.WorkflowUserTaskCompletionType", Values: []string{
		"Single", "Veto", "Consensus", "Majority", "Threshold", "Microflow",
	}},
	{Name: "System.WorkflowEventType", Values: []string{
		"WorkflowCompleted", "WorkflowInitiated", "WorkflowRestarted",
		"WorkflowFailed", "WorkflowAborted", "WorkflowPaused",
		"WorkflowUnpaused", "WorkflowRetried", "WorkflowUpdated",
		"WorkflowUpgraded", "WorkflowConflicted", "WorkflowResolved",
		"WorkflowJumpToOptionApplied",
		"StartEventExecuted", "EndEventExecuted", "DecisionExecuted",
		"JumpExecuted", "ParallelSplitExecuted", "ParallelMergeExecuted",
		"CallWorkflowStarted", "CallWorkflowEnded",
		"CallMicroflowStarted", "CallMicroflowEnded",
		"WaitForNotificationStarted", "WaitForNotificationEnded",
		"WaitForTimerStarted", "WaitForTimerEnded",
		"UserTaskStarted", "MultiUserTaskOutcomeSelected", "UserTaskEnded",
		"NonInterruptingTimerEventExecuted", "InterruptingTimerEventExecuted",
	}},
}
