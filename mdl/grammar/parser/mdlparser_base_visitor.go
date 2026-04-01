// Code generated from mdl/grammar/MDLParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // MDLParser
import "github.com/antlr4-go/antlr/v4"

type BaseMDLParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseMDLParserVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDdlStatement(ctx *DdlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUpdateWidgetsStatement(ctx *UpdateWidgetsStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateStatement(ctx *CreateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterStatement(ctx *AlterStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterStylingAction(ctx *AlterStylingActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterStylingAssignment(ctx *AlterStylingAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageOperation(ctx *AlterPageOperationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageSet(ctx *AlterPageSetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterLayoutMapping(ctx *AlterLayoutMappingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageAssignment(ctx *AlterPageAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageInsert(ctx *AlterPageInsertContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageDrop(ctx *AlterPageDropContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageReplace(ctx *AlterPageReplaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageAddVariable(ctx *AlterPageAddVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterPageDropVariable(ctx *AlterPageDropVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNavigationClause(ctx *NavigationClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNavMenuItemDef(ctx *NavMenuItemDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDropStatement(ctx *DropStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRenameStatement(ctx *RenameStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMoveStatement(ctx *MoveStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSecurityStatement(ctx *SecurityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateModuleRoleStatement(ctx *CreateModuleRoleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDropModuleRoleStatement(ctx *DropModuleRoleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateUserRoleStatement(ctx *CreateUserRoleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterUserRoleStatement(ctx *AlterUserRoleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDropUserRoleStatement(ctx *DropUserRoleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGrantEntityAccessStatement(ctx *GrantEntityAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRevokeEntityAccessStatement(ctx *RevokeEntityAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGrantMicroflowAccessStatement(ctx *GrantMicroflowAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRevokeMicroflowAccessStatement(ctx *RevokeMicroflowAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGrantPageAccessStatement(ctx *GrantPageAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRevokePageAccessStatement(ctx *RevokePageAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGrantWorkflowAccessStatement(ctx *GrantWorkflowAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRevokeWorkflowAccessStatement(ctx *RevokeWorkflowAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGrantODataServiceAccessStatement(ctx *GrantODataServiceAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRevokeODataServiceAccessStatement(ctx *RevokeODataServiceAccessStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterProjectSecurityStatement(ctx *AlterProjectSecurityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateDemoUserStatement(ctx *CreateDemoUserStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDropDemoUserStatement(ctx *DropDemoUserStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUpdateSecurityStatement(ctx *UpdateSecurityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitModuleRoleList(ctx *ModuleRoleListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEntityAccessRightList(ctx *EntityAccessRightListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEntityAccessRight(ctx *EntityAccessRightContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateEntityStatement(ctx *CreateEntityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGeneralizationClause(ctx *GeneralizationClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEntityBody(ctx *EntityBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEntityOptions(ctx *EntityOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEntityOption(ctx *EntityOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeDefinitionList(ctx *AttributeDefinitionListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeDefinition(ctx *AttributeDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeName(ctx *AttributeNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeConstraint(ctx *AttributeConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDataType(ctx *DataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitTemplateContext(ctx *TemplateContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNonListDataType(ctx *NonListDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIndexDefinition(ctx *IndexDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIndexAttributeList(ctx *IndexAttributeListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIndexAttribute(ctx *IndexAttributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIndexColumnName(ctx *IndexColumnNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateAssociationStatement(ctx *CreateAssociationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAssociationOptions(ctx *AssociationOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAssociationOption(ctx *AssociationOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDeleteBehavior(ctx *DeleteBehaviorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterEntityAction(ctx *AlterEntityActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterAssociationAction(ctx *AlterAssociationActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterEnumerationAction(ctx *AlterEnumerationActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterNotebookAction(ctx *AlterNotebookActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateModuleStatement(ctx *CreateModuleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitModuleOptions(ctx *ModuleOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitModuleOption(ctx *ModuleOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateEnumerationStatement(ctx *CreateEnumerationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEnumerationValueList(ctx *EnumerationValueListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEnumerationValue(ctx *EnumerationValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEnumValueName(ctx *EnumValueNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEnumerationOptions(ctx *EnumerationOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitEnumerationOption(ctx *EnumerationOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateImageCollectionStatement(ctx *CreateImageCollectionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImageCollectionOptions(ctx *ImageCollectionOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImageCollectionOption(ctx *ImageCollectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImageCollectionBody(ctx *ImageCollectionBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImageCollectionItem(ctx *ImageCollectionItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImageName(ctx *ImageNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateJsonStructureStatement(ctx *CreateJsonStructureStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCustomNameMapping(ctx *CustomNameMappingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateValidationRuleStatement(ctx *CreateValidationRuleStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitValidationRuleBody(ctx *ValidationRuleBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRangeConstraint(ctx *RangeConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeReference(ctx *AttributeReferenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeReferenceList(ctx *AttributeReferenceListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateMicroflowStatement(ctx *CreateMicroflowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateJavaActionStatement(ctx *CreateJavaActionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJavaActionParameterList(ctx *JavaActionParameterListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJavaActionParameter(ctx *JavaActionParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJavaActionReturnType(ctx *JavaActionReturnTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJavaActionExposedClause(ctx *JavaActionExposedClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowParameterList(ctx *MicroflowParameterListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowParameter(ctx *MicroflowParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitParameterName(ctx *ParameterNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowReturnType(ctx *MicroflowReturnTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowOptions(ctx *MicroflowOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowOption(ctx *MicroflowOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowBody(ctx *MicroflowBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowStatement(ctx *MicroflowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDeclareStatement(ctx *DeclareStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSetStatement(ctx *SetStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateObjectStatement(ctx *CreateObjectStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitChangeObjectStatement(ctx *ChangeObjectStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributePath(ctx *AttributePathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCommitStatement(ctx *CommitStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDeleteObjectStatement(ctx *DeleteObjectStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRollbackStatement(ctx *RollbackStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRetrieveStatement(ctx *RetrieveStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRetrieveSource(ctx *RetrieveSourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOnErrorClause(ctx *OnErrorClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIfStatement(ctx *IfStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLoopStatement(ctx *LoopStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWhileStatement(ctx *WhileStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitContinueStatement(ctx *ContinueStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitBreakStatement(ctx *BreakStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitReturnStatement(ctx *ReturnStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRaiseErrorStatement(ctx *RaiseErrorStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLogStatement(ctx *LogStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLogLevel(ctx *LogLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitTemplateParams(ctx *TemplateParamsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitTemplateParam(ctx *TemplateParamContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLogTemplateParams(ctx *LogTemplateParamsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLogTemplateParam(ctx *LogTemplateParamContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCallMicroflowStatement(ctx *CallMicroflowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCallJavaActionStatement(ctx *CallJavaActionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExecuteDatabaseQueryStatement(ctx *ExecuteDatabaseQueryStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCallExternalActionStatement(ctx *CallExternalActionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCallArgumentList(ctx *CallArgumentListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCallArgument(ctx *CallArgumentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowPageStatement(ctx *ShowPageStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowPageArgList(ctx *ShowPageArgListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowPageArg(ctx *ShowPageArgContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitClosePageStatement(ctx *ClosePageStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowHomePageStatement(ctx *ShowHomePageStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowMessageStatement(ctx *ShowMessageStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitThrowStatement(ctx *ThrowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitValidationFeedbackStatement(ctx *ValidationFeedbackStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallStatement(ctx *RestCallStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitHttpMethod(ctx *HttpMethodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallUrl(ctx *RestCallUrlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallUrlParams(ctx *RestCallUrlParamsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallHeaderClause(ctx *RestCallHeaderClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallAuthClause(ctx *RestCallAuthClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallBodyClause(ctx *RestCallBodyClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallTimeoutClause(ctx *RestCallTimeoutClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestCallReturnsClause(ctx *RestCallReturnsClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSendRestRequestStatement(ctx *SendRestRequestStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSendRestRequestBodyClause(ctx *SendRestRequestBodyClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitListOperationStatement(ctx *ListOperationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitListOperation(ctx *ListOperationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSortSpecList(ctx *SortSpecListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSortSpec(ctx *SortSpecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAggregateListStatement(ctx *AggregateListStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitListAggregateOperation(ctx *ListAggregateOperationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateListStatement(ctx *CreateListStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAddToListStatement(ctx *AddToListStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRemoveFromListStatement(ctx *RemoveFromListStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMemberAssignmentList(ctx *MemberAssignmentListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMemberAssignment(ctx *MemberAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMemberAttributeName(ctx *MemberAttributeNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitChangeList(ctx *ChangeListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitChangeItem(ctx *ChangeItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreatePageStatement(ctx *CreatePageStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateSnippetStatement(ctx *CreateSnippetStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetOptions(ctx *SnippetOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetOption(ctx *SnippetOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPageParameterList(ctx *PageParameterListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPageParameter(ctx *PageParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetParameterList(ctx *SnippetParameterListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetParameter(ctx *SnippetParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitVariableDeclarationList(ctx *VariableDeclarationListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSortColumn(ctx *SortColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathConstraint(ctx *XpathConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAndOrXpath(ctx *AndOrXpathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathExpr(ctx *XpathExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathAndExpr(ctx *XpathAndExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathNotExpr(ctx *XpathNotExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathComparisonExpr(ctx *XpathComparisonExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathValueExpr(ctx *XpathValueExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathPath(ctx *XpathPathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathStep(ctx *XpathStepContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathStepValue(ctx *XpathStepValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathQualifiedName(ctx *XpathQualifiedNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathWord(ctx *XpathWordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathFunctionCall(ctx *XpathFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitXpathFunctionName(ctx *XpathFunctionNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPageHeaderV3(ctx *PageHeaderV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPageHeaderPropertyV3(ctx *PageHeaderPropertyV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetHeaderV3(ctx *SnippetHeaderV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSnippetHeaderPropertyV3(ctx *SnippetHeaderPropertyV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPageBodyV3(ctx *PageBodyV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUseFragmentRef(ctx *UseFragmentRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetV3(ctx *WidgetV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetTypeV3(ctx *WidgetTypeV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetPropertiesV3(ctx *WidgetPropertiesV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetPropertyV3(ctx *WidgetPropertyV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitFilterTypeValue(ctx *FilterTypeValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributeListV3(ctx *AttributeListV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDataSourceExprV3(ctx *DataSourceExprV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitActionExprV3(ctx *ActionExprV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowArgsV3(ctx *MicroflowArgsV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMicroflowArgV3(ctx *MicroflowArgV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAttributePathV3(ctx *AttributePathV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitStringExprV3(ctx *StringExprV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitParamListV3(ctx *ParamListV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitParamAssignmentV3(ctx *ParamAssignmentV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRenderModeV3(ctx *RenderModeV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitButtonStyleV3(ctx *ButtonStyleV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDesktopWidthV3(ctx *DesktopWidthV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSelectionModeV3(ctx *SelectionModeV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPropertyValueV3(ctx *PropertyValueV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDesignPropertyListV3(ctx *DesignPropertyListV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDesignPropertyEntryV3(ctx *DesignPropertyEntryV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetBodyV3(ctx *WidgetBodyV3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateNotebookStatement(ctx *CreateNotebookStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNotebookOptions(ctx *NotebookOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNotebookOption(ctx *NotebookOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNotebookPage(ctx *NotebookPageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateDatabaseConnectionStatement(ctx *CreateDatabaseConnectionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDatabaseConnectionOption(ctx *DatabaseConnectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDatabaseQuery(ctx *DatabaseQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDatabaseQueryMapping(ctx *DatabaseQueryMappingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateConstantStatement(ctx *CreateConstantStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitConstantOptions(ctx *ConstantOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitConstantOption(ctx *ConstantOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateConfigurationStatement(ctx *CreateConfigurationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateRestClientStatement(ctx *CreateRestClientStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestClientBaseUrl(ctx *RestClientBaseUrlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestClientAuthentication(ctx *RestClientAuthenticationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestAuthValue(ctx *RestAuthValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestOperationDef(ctx *RestOperationDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestHttpMethod(ctx *RestHttpMethodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestOperationClause(ctx *RestOperationClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestHeaderValue(ctx *RestHeaderValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitRestResponseSpec(ctx *RestResponseSpecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateIndexStatement(ctx *CreateIndexStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateODataClientStatement(ctx *CreateODataClientStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateODataServiceStatement(ctx *CreateODataServiceStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataPropertyValue(ctx *OdataPropertyValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataPropertyAssignment(ctx *OdataPropertyAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataAlterAssignment(ctx *OdataAlterAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataAuthenticationClause(ctx *OdataAuthenticationClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataAuthType(ctx *OdataAuthTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPublishEntityBlock(ctx *PublishEntityBlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExposeClause(ctx *ExposeClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExposeMember(ctx *ExposeMemberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExposeMemberOptions(ctx *ExposeMemberOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateExternalEntityStatement(ctx *CreateExternalEntityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateNavigationStatement(ctx *CreateNavigationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataHeadersClause(ctx *OdataHeadersClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOdataHeaderEntry(ctx *OdataHeaderEntryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateBusinessEventServiceStatement(ctx *CreateBusinessEventServiceStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitBusinessEventMessageDef(ctx *BusinessEventMessageDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitBusinessEventAttrDef(ctx *BusinessEventAttrDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCreateWorkflowStatement(ctx *CreateWorkflowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowBody(ctx *WorkflowBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowActivityStmt(ctx *WorkflowActivityStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowUserTaskStmt(ctx *WorkflowUserTaskStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowBoundaryEventClause(ctx *WorkflowBoundaryEventClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowUserTaskOutcome(ctx *WorkflowUserTaskOutcomeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowCallMicroflowStmt(ctx *WorkflowCallMicroflowStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowParameterMapping(ctx *WorkflowParameterMappingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowCallWorkflowStmt(ctx *WorkflowCallWorkflowStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowDecisionStmt(ctx *WorkflowDecisionStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowConditionOutcome(ctx *WorkflowConditionOutcomeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowParallelSplitStmt(ctx *WorkflowParallelSplitStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowParallelPath(ctx *WorkflowParallelPathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowJumpToStmt(ctx *WorkflowJumpToStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowWaitForTimerStmt(ctx *WorkflowWaitForTimerStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowWaitForNotificationStmt(ctx *WorkflowWaitForNotificationStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWorkflowAnnotationStmt(ctx *WorkflowAnnotationStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAlterSettingsClause(ctx *AlterSettingsClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSettingsSection(ctx *SettingsSectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSettingsAssignment(ctx *SettingsAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSettingsValue(ctx *SettingsValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDqlStatement(ctx *DqlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowStatement(ctx *ShowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitShowWidgetsFilter(ctx *ShowWidgetsFilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetTypeKeyword(ctx *WidgetTypeKeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetCondition(ctx *WidgetConditionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetPropertyAssignment(ctx *WidgetPropertyAssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWidgetPropertyValue(ctx *WidgetPropertyValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDescribeStatement(ctx *DescribeStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCatalogSelectQuery(ctx *CatalogSelectQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCatalogJoinClause(ctx *CatalogJoinClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCatalogTableName(ctx *CatalogTableNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOqlQuery(ctx *OqlQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOqlQueryTerm(ctx *OqlQueryTermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSelectClause(ctx *SelectClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSelectList(ctx *SelectListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSelectItem(ctx *SelectItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSelectAlias(ctx *SelectAliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitFromClause(ctx *FromClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitTableReference(ctx *TableReferenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJoinClause(ctx *JoinClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAssociationPath(ctx *AssociationPathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitJoinType(ctx *JoinTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitWhereClause(ctx *WhereClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGroupByClause(ctx *GroupByClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitHavingClause(ctx *HavingClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOrderByClause(ctx *OrderByClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOrderByList(ctx *OrderByListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOrderByItem(ctx *OrderByItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitGroupByList(ctx *GroupByListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLimitOffsetClause(ctx *LimitOffsetClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUtilityStatement(ctx *UtilityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSearchStatement(ctx *SearchStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitConnectStatement(ctx *ConnectStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDisconnectStatement(ctx *DisconnectStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUpdateStatement(ctx *UpdateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCheckStatement(ctx *CheckStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitBuildStatement(ctx *BuildStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExecuteScriptStatement(ctx *ExecuteScriptStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExecuteRuntimeStatement(ctx *ExecuteRuntimeStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLintStatement(ctx *LintStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLintTarget(ctx *LintTargetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLintFormat(ctx *LintFormatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUseSessionStatement(ctx *UseSessionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSessionIdList(ctx *SessionIdListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSessionId(ctx *SessionIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIntrospectApiStatement(ctx *IntrospectApiStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDebugStatement(ctx *DebugStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlConnect(ctx *SqlConnectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlDisconnect(ctx *SqlDisconnectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlConnections(ctx *SqlConnectionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlShowTables(ctx *SqlShowTablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlDescribeTable(ctx *SqlDescribeTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlGenerateConnector(ctx *SqlGenerateConnectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlQuery(ctx *SqlQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitSqlPassthrough(ctx *SqlPassthroughContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImportFromQuery(ctx *ImportFromQueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitImportMapping(ctx *ImportMappingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLinkLookup(ctx *LinkLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLinkDirect(ctx *LinkDirectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitHelpStatement(ctx *HelpStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDefineFragmentStatement(ctx *DefineFragmentStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitOrExpression(ctx *OrExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAndExpression(ctx *AndExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitNotExpression(ctx *NotExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitComparisonExpression(ctx *ComparisonExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitComparisonOperator(ctx *ComparisonOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitUnaryExpression(ctx *UnaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCaseExpression(ctx *CaseExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIfThenElseExpression(ctx *IfThenElseExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCastExpression(ctx *CastExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCastDataType(ctx *CastDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAggregateFunction(ctx *AggregateFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitFunctionCall(ctx *FunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitFunctionName(ctx *FunctionNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitArgumentList(ctx *ArgumentListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAtomicExpression(ctx *AtomicExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitExpressionList(ctx *ExpressionListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitQualifiedName(ctx *QualifiedNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitIdentifierOrKeyword(ctx *IdentifierOrKeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitLiteral(ctx *LiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitArrayLiteral(ctx *ArrayLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitDocComment(ctx *DocCommentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAnnotation(ctx *AnnotationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAnnotationName(ctx *AnnotationNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAnnotationParams(ctx *AnnotationParamsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAnnotationParam(ctx *AnnotationParamContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitAnnotationValue(ctx *AnnotationValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitCommonNameKeyword(ctx *CommonNameKeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMDLParserVisitor) VisitKeyword(ctx *KeywordContext) interface{} {
	return v.VisitChildren(ctx)
}
