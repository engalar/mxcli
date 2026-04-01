// Code generated from mdl/grammar/MDLParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // MDLParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by MDLParser.
type MDLParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by MDLParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by MDLParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by MDLParser#ddlStatement.
	VisitDdlStatement(ctx *DdlStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#updateWidgetsStatement.
	VisitUpdateWidgetsStatement(ctx *UpdateWidgetsStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createStatement.
	VisitCreateStatement(ctx *CreateStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#alterStatement.
	VisitAlterStatement(ctx *AlterStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#alterStylingAction.
	VisitAlterStylingAction(ctx *AlterStylingActionContext) interface{}

	// Visit a parse tree produced by MDLParser#alterStylingAssignment.
	VisitAlterStylingAssignment(ctx *AlterStylingAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageOperation.
	VisitAlterPageOperation(ctx *AlterPageOperationContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageSet.
	VisitAlterPageSet(ctx *AlterPageSetContext) interface{}

	// Visit a parse tree produced by MDLParser#alterLayoutMapping.
	VisitAlterLayoutMapping(ctx *AlterLayoutMappingContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageAssignment.
	VisitAlterPageAssignment(ctx *AlterPageAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageInsert.
	VisitAlterPageInsert(ctx *AlterPageInsertContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageDrop.
	VisitAlterPageDrop(ctx *AlterPageDropContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageReplace.
	VisitAlterPageReplace(ctx *AlterPageReplaceContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageAddVariable.
	VisitAlterPageAddVariable(ctx *AlterPageAddVariableContext) interface{}

	// Visit a parse tree produced by MDLParser#alterPageDropVariable.
	VisitAlterPageDropVariable(ctx *AlterPageDropVariableContext) interface{}

	// Visit a parse tree produced by MDLParser#navigationClause.
	VisitNavigationClause(ctx *NavigationClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#navMenuItemDef.
	VisitNavMenuItemDef(ctx *NavMenuItemDefContext) interface{}

	// Visit a parse tree produced by MDLParser#dropStatement.
	VisitDropStatement(ctx *DropStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#renameStatement.
	VisitRenameStatement(ctx *RenameStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#moveStatement.
	VisitMoveStatement(ctx *MoveStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#securityStatement.
	VisitSecurityStatement(ctx *SecurityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createModuleRoleStatement.
	VisitCreateModuleRoleStatement(ctx *CreateModuleRoleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#dropModuleRoleStatement.
	VisitDropModuleRoleStatement(ctx *DropModuleRoleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createUserRoleStatement.
	VisitCreateUserRoleStatement(ctx *CreateUserRoleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#alterUserRoleStatement.
	VisitAlterUserRoleStatement(ctx *AlterUserRoleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#dropUserRoleStatement.
	VisitDropUserRoleStatement(ctx *DropUserRoleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#grantEntityAccessStatement.
	VisitGrantEntityAccessStatement(ctx *GrantEntityAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#revokeEntityAccessStatement.
	VisitRevokeEntityAccessStatement(ctx *RevokeEntityAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#grantMicroflowAccessStatement.
	VisitGrantMicroflowAccessStatement(ctx *GrantMicroflowAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#revokeMicroflowAccessStatement.
	VisitRevokeMicroflowAccessStatement(ctx *RevokeMicroflowAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#grantPageAccessStatement.
	VisitGrantPageAccessStatement(ctx *GrantPageAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#revokePageAccessStatement.
	VisitRevokePageAccessStatement(ctx *RevokePageAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#grantWorkflowAccessStatement.
	VisitGrantWorkflowAccessStatement(ctx *GrantWorkflowAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#revokeWorkflowAccessStatement.
	VisitRevokeWorkflowAccessStatement(ctx *RevokeWorkflowAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#grantODataServiceAccessStatement.
	VisitGrantODataServiceAccessStatement(ctx *GrantODataServiceAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#revokeODataServiceAccessStatement.
	VisitRevokeODataServiceAccessStatement(ctx *RevokeODataServiceAccessStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#alterProjectSecurityStatement.
	VisitAlterProjectSecurityStatement(ctx *AlterProjectSecurityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createDemoUserStatement.
	VisitCreateDemoUserStatement(ctx *CreateDemoUserStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#dropDemoUserStatement.
	VisitDropDemoUserStatement(ctx *DropDemoUserStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#updateSecurityStatement.
	VisitUpdateSecurityStatement(ctx *UpdateSecurityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#moduleRoleList.
	VisitModuleRoleList(ctx *ModuleRoleListContext) interface{}

	// Visit a parse tree produced by MDLParser#entityAccessRightList.
	VisitEntityAccessRightList(ctx *EntityAccessRightListContext) interface{}

	// Visit a parse tree produced by MDLParser#entityAccessRight.
	VisitEntityAccessRight(ctx *EntityAccessRightContext) interface{}

	// Visit a parse tree produced by MDLParser#createEntityStatement.
	VisitCreateEntityStatement(ctx *CreateEntityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#generalizationClause.
	VisitGeneralizationClause(ctx *GeneralizationClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#entityBody.
	VisitEntityBody(ctx *EntityBodyContext) interface{}

	// Visit a parse tree produced by MDLParser#entityOptions.
	VisitEntityOptions(ctx *EntityOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#entityOption.
	VisitEntityOption(ctx *EntityOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeDefinitionList.
	VisitAttributeDefinitionList(ctx *AttributeDefinitionListContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeDefinition.
	VisitAttributeDefinition(ctx *AttributeDefinitionContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeName.
	VisitAttributeName(ctx *AttributeNameContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeConstraint.
	VisitAttributeConstraint(ctx *AttributeConstraintContext) interface{}

	// Visit a parse tree produced by MDLParser#dataType.
	VisitDataType(ctx *DataTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#templateContext.
	VisitTemplateContext(ctx *TemplateContextContext) interface{}

	// Visit a parse tree produced by MDLParser#nonListDataType.
	VisitNonListDataType(ctx *NonListDataTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#indexDefinition.
	VisitIndexDefinition(ctx *IndexDefinitionContext) interface{}

	// Visit a parse tree produced by MDLParser#indexAttributeList.
	VisitIndexAttributeList(ctx *IndexAttributeListContext) interface{}

	// Visit a parse tree produced by MDLParser#indexAttribute.
	VisitIndexAttribute(ctx *IndexAttributeContext) interface{}

	// Visit a parse tree produced by MDLParser#indexColumnName.
	VisitIndexColumnName(ctx *IndexColumnNameContext) interface{}

	// Visit a parse tree produced by MDLParser#createAssociationStatement.
	VisitCreateAssociationStatement(ctx *CreateAssociationStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#associationOptions.
	VisitAssociationOptions(ctx *AssociationOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#associationOption.
	VisitAssociationOption(ctx *AssociationOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#deleteBehavior.
	VisitDeleteBehavior(ctx *DeleteBehaviorContext) interface{}

	// Visit a parse tree produced by MDLParser#alterEntityAction.
	VisitAlterEntityAction(ctx *AlterEntityActionContext) interface{}

	// Visit a parse tree produced by MDLParser#alterAssociationAction.
	VisitAlterAssociationAction(ctx *AlterAssociationActionContext) interface{}

	// Visit a parse tree produced by MDLParser#alterEnumerationAction.
	VisitAlterEnumerationAction(ctx *AlterEnumerationActionContext) interface{}

	// Visit a parse tree produced by MDLParser#alterNotebookAction.
	VisitAlterNotebookAction(ctx *AlterNotebookActionContext) interface{}

	// Visit a parse tree produced by MDLParser#createModuleStatement.
	VisitCreateModuleStatement(ctx *CreateModuleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#moduleOptions.
	VisitModuleOptions(ctx *ModuleOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#moduleOption.
	VisitModuleOption(ctx *ModuleOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#createEnumerationStatement.
	VisitCreateEnumerationStatement(ctx *CreateEnumerationStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#enumerationValueList.
	VisitEnumerationValueList(ctx *EnumerationValueListContext) interface{}

	// Visit a parse tree produced by MDLParser#enumerationValue.
	VisitEnumerationValue(ctx *EnumerationValueContext) interface{}

	// Visit a parse tree produced by MDLParser#enumValueName.
	VisitEnumValueName(ctx *EnumValueNameContext) interface{}

	// Visit a parse tree produced by MDLParser#enumerationOptions.
	VisitEnumerationOptions(ctx *EnumerationOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#enumerationOption.
	VisitEnumerationOption(ctx *EnumerationOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#createImageCollectionStatement.
	VisitCreateImageCollectionStatement(ctx *CreateImageCollectionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#imageCollectionOptions.
	VisitImageCollectionOptions(ctx *ImageCollectionOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#imageCollectionOption.
	VisitImageCollectionOption(ctx *ImageCollectionOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#imageCollectionBody.
	VisitImageCollectionBody(ctx *ImageCollectionBodyContext) interface{}

	// Visit a parse tree produced by MDLParser#imageCollectionItem.
	VisitImageCollectionItem(ctx *ImageCollectionItemContext) interface{}

	// Visit a parse tree produced by MDLParser#imageName.
	VisitImageName(ctx *ImageNameContext) interface{}

	// Visit a parse tree produced by MDLParser#createJsonStructureStatement.
	VisitCreateJsonStructureStatement(ctx *CreateJsonStructureStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#customNameMapping.
	VisitCustomNameMapping(ctx *CustomNameMappingContext) interface{}

	// Visit a parse tree produced by MDLParser#createValidationRuleStatement.
	VisitCreateValidationRuleStatement(ctx *CreateValidationRuleStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#validationRuleBody.
	VisitValidationRuleBody(ctx *ValidationRuleBodyContext) interface{}

	// Visit a parse tree produced by MDLParser#rangeConstraint.
	VisitRangeConstraint(ctx *RangeConstraintContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeReference.
	VisitAttributeReference(ctx *AttributeReferenceContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeReferenceList.
	VisitAttributeReferenceList(ctx *AttributeReferenceListContext) interface{}

	// Visit a parse tree produced by MDLParser#createMicroflowStatement.
	VisitCreateMicroflowStatement(ctx *CreateMicroflowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createJavaActionStatement.
	VisitCreateJavaActionStatement(ctx *CreateJavaActionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#javaActionParameterList.
	VisitJavaActionParameterList(ctx *JavaActionParameterListContext) interface{}

	// Visit a parse tree produced by MDLParser#javaActionParameter.
	VisitJavaActionParameter(ctx *JavaActionParameterContext) interface{}

	// Visit a parse tree produced by MDLParser#javaActionReturnType.
	VisitJavaActionReturnType(ctx *JavaActionReturnTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#javaActionExposedClause.
	VisitJavaActionExposedClause(ctx *JavaActionExposedClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowParameterList.
	VisitMicroflowParameterList(ctx *MicroflowParameterListContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowParameter.
	VisitMicroflowParameter(ctx *MicroflowParameterContext) interface{}

	// Visit a parse tree produced by MDLParser#parameterName.
	VisitParameterName(ctx *ParameterNameContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowReturnType.
	VisitMicroflowReturnType(ctx *MicroflowReturnTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowOptions.
	VisitMicroflowOptions(ctx *MicroflowOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowOption.
	VisitMicroflowOption(ctx *MicroflowOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowBody.
	VisitMicroflowBody(ctx *MicroflowBodyContext) interface{}

	// Visit a parse tree produced by MDLParser#microflowStatement.
	VisitMicroflowStatement(ctx *MicroflowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#declareStatement.
	VisitDeclareStatement(ctx *DeclareStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#setStatement.
	VisitSetStatement(ctx *SetStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createObjectStatement.
	VisitCreateObjectStatement(ctx *CreateObjectStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#changeObjectStatement.
	VisitChangeObjectStatement(ctx *ChangeObjectStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#attributePath.
	VisitAttributePath(ctx *AttributePathContext) interface{}

	// Visit a parse tree produced by MDLParser#commitStatement.
	VisitCommitStatement(ctx *CommitStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#deleteObjectStatement.
	VisitDeleteObjectStatement(ctx *DeleteObjectStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#rollbackStatement.
	VisitRollbackStatement(ctx *RollbackStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#retrieveStatement.
	VisitRetrieveStatement(ctx *RetrieveStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#retrieveSource.
	VisitRetrieveSource(ctx *RetrieveSourceContext) interface{}

	// Visit a parse tree produced by MDLParser#onErrorClause.
	VisitOnErrorClause(ctx *OnErrorClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#loopStatement.
	VisitLoopStatement(ctx *LoopStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#whileStatement.
	VisitWhileStatement(ctx *WhileStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#continueStatement.
	VisitContinueStatement(ctx *ContinueStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#breakStatement.
	VisitBreakStatement(ctx *BreakStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#returnStatement.
	VisitReturnStatement(ctx *ReturnStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#raiseErrorStatement.
	VisitRaiseErrorStatement(ctx *RaiseErrorStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#logStatement.
	VisitLogStatement(ctx *LogStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#logLevel.
	VisitLogLevel(ctx *LogLevelContext) interface{}

	// Visit a parse tree produced by MDLParser#templateParams.
	VisitTemplateParams(ctx *TemplateParamsContext) interface{}

	// Visit a parse tree produced by MDLParser#templateParam.
	VisitTemplateParam(ctx *TemplateParamContext) interface{}

	// Visit a parse tree produced by MDLParser#logTemplateParams.
	VisitLogTemplateParams(ctx *LogTemplateParamsContext) interface{}

	// Visit a parse tree produced by MDLParser#logTemplateParam.
	VisitLogTemplateParam(ctx *LogTemplateParamContext) interface{}

	// Visit a parse tree produced by MDLParser#callMicroflowStatement.
	VisitCallMicroflowStatement(ctx *CallMicroflowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#callJavaActionStatement.
	VisitCallJavaActionStatement(ctx *CallJavaActionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#executeDatabaseQueryStatement.
	VisitExecuteDatabaseQueryStatement(ctx *ExecuteDatabaseQueryStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#callExternalActionStatement.
	VisitCallExternalActionStatement(ctx *CallExternalActionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#callArgumentList.
	VisitCallArgumentList(ctx *CallArgumentListContext) interface{}

	// Visit a parse tree produced by MDLParser#callArgument.
	VisitCallArgument(ctx *CallArgumentContext) interface{}

	// Visit a parse tree produced by MDLParser#showPageStatement.
	VisitShowPageStatement(ctx *ShowPageStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#showPageArgList.
	VisitShowPageArgList(ctx *ShowPageArgListContext) interface{}

	// Visit a parse tree produced by MDLParser#showPageArg.
	VisitShowPageArg(ctx *ShowPageArgContext) interface{}

	// Visit a parse tree produced by MDLParser#closePageStatement.
	VisitClosePageStatement(ctx *ClosePageStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#showHomePageStatement.
	VisitShowHomePageStatement(ctx *ShowHomePageStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#showMessageStatement.
	VisitShowMessageStatement(ctx *ShowMessageStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#throwStatement.
	VisitThrowStatement(ctx *ThrowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#validationFeedbackStatement.
	VisitValidationFeedbackStatement(ctx *ValidationFeedbackStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallStatement.
	VisitRestCallStatement(ctx *RestCallStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#httpMethod.
	VisitHttpMethod(ctx *HttpMethodContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallUrl.
	VisitRestCallUrl(ctx *RestCallUrlContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallUrlParams.
	VisitRestCallUrlParams(ctx *RestCallUrlParamsContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallHeaderClause.
	VisitRestCallHeaderClause(ctx *RestCallHeaderClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallAuthClause.
	VisitRestCallAuthClause(ctx *RestCallAuthClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallBodyClause.
	VisitRestCallBodyClause(ctx *RestCallBodyClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallTimeoutClause.
	VisitRestCallTimeoutClause(ctx *RestCallTimeoutClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#restCallReturnsClause.
	VisitRestCallReturnsClause(ctx *RestCallReturnsClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#sendRestRequestStatement.
	VisitSendRestRequestStatement(ctx *SendRestRequestStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#sendRestRequestBodyClause.
	VisitSendRestRequestBodyClause(ctx *SendRestRequestBodyClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#listOperationStatement.
	VisitListOperationStatement(ctx *ListOperationStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#listOperation.
	VisitListOperation(ctx *ListOperationContext) interface{}

	// Visit a parse tree produced by MDLParser#sortSpecList.
	VisitSortSpecList(ctx *SortSpecListContext) interface{}

	// Visit a parse tree produced by MDLParser#sortSpec.
	VisitSortSpec(ctx *SortSpecContext) interface{}

	// Visit a parse tree produced by MDLParser#aggregateListStatement.
	VisitAggregateListStatement(ctx *AggregateListStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#listAggregateOperation.
	VisitListAggregateOperation(ctx *ListAggregateOperationContext) interface{}

	// Visit a parse tree produced by MDLParser#createListStatement.
	VisitCreateListStatement(ctx *CreateListStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#addToListStatement.
	VisitAddToListStatement(ctx *AddToListStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#removeFromListStatement.
	VisitRemoveFromListStatement(ctx *RemoveFromListStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#memberAssignmentList.
	VisitMemberAssignmentList(ctx *MemberAssignmentListContext) interface{}

	// Visit a parse tree produced by MDLParser#memberAssignment.
	VisitMemberAssignment(ctx *MemberAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#memberAttributeName.
	VisitMemberAttributeName(ctx *MemberAttributeNameContext) interface{}

	// Visit a parse tree produced by MDLParser#changeList.
	VisitChangeList(ctx *ChangeListContext) interface{}

	// Visit a parse tree produced by MDLParser#changeItem.
	VisitChangeItem(ctx *ChangeItemContext) interface{}

	// Visit a parse tree produced by MDLParser#createPageStatement.
	VisitCreatePageStatement(ctx *CreatePageStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createSnippetStatement.
	VisitCreateSnippetStatement(ctx *CreateSnippetStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#snippetOptions.
	VisitSnippetOptions(ctx *SnippetOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#snippetOption.
	VisitSnippetOption(ctx *SnippetOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#pageParameterList.
	VisitPageParameterList(ctx *PageParameterListContext) interface{}

	// Visit a parse tree produced by MDLParser#pageParameter.
	VisitPageParameter(ctx *PageParameterContext) interface{}

	// Visit a parse tree produced by MDLParser#snippetParameterList.
	VisitSnippetParameterList(ctx *SnippetParameterListContext) interface{}

	// Visit a parse tree produced by MDLParser#snippetParameter.
	VisitSnippetParameter(ctx *SnippetParameterContext) interface{}

	// Visit a parse tree produced by MDLParser#variableDeclarationList.
	VisitVariableDeclarationList(ctx *VariableDeclarationListContext) interface{}

	// Visit a parse tree produced by MDLParser#variableDeclaration.
	VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{}

	// Visit a parse tree produced by MDLParser#sortColumn.
	VisitSortColumn(ctx *SortColumnContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathConstraint.
	VisitXpathConstraint(ctx *XpathConstraintContext) interface{}

	// Visit a parse tree produced by MDLParser#andOrXpath.
	VisitAndOrXpath(ctx *AndOrXpathContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathExpr.
	VisitXpathExpr(ctx *XpathExprContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathAndExpr.
	VisitXpathAndExpr(ctx *XpathAndExprContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathNotExpr.
	VisitXpathNotExpr(ctx *XpathNotExprContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathComparisonExpr.
	VisitXpathComparisonExpr(ctx *XpathComparisonExprContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathValueExpr.
	VisitXpathValueExpr(ctx *XpathValueExprContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathPath.
	VisitXpathPath(ctx *XpathPathContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathStep.
	VisitXpathStep(ctx *XpathStepContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathStepValue.
	VisitXpathStepValue(ctx *XpathStepValueContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathQualifiedName.
	VisitXpathQualifiedName(ctx *XpathQualifiedNameContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathWord.
	VisitXpathWord(ctx *XpathWordContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathFunctionCall.
	VisitXpathFunctionCall(ctx *XpathFunctionCallContext) interface{}

	// Visit a parse tree produced by MDLParser#xpathFunctionName.
	VisitXpathFunctionName(ctx *XpathFunctionNameContext) interface{}

	// Visit a parse tree produced by MDLParser#pageHeaderV3.
	VisitPageHeaderV3(ctx *PageHeaderV3Context) interface{}

	// Visit a parse tree produced by MDLParser#pageHeaderPropertyV3.
	VisitPageHeaderPropertyV3(ctx *PageHeaderPropertyV3Context) interface{}

	// Visit a parse tree produced by MDLParser#snippetHeaderV3.
	VisitSnippetHeaderV3(ctx *SnippetHeaderV3Context) interface{}

	// Visit a parse tree produced by MDLParser#snippetHeaderPropertyV3.
	VisitSnippetHeaderPropertyV3(ctx *SnippetHeaderPropertyV3Context) interface{}

	// Visit a parse tree produced by MDLParser#pageBodyV3.
	VisitPageBodyV3(ctx *PageBodyV3Context) interface{}

	// Visit a parse tree produced by MDLParser#useFragmentRef.
	VisitUseFragmentRef(ctx *UseFragmentRefContext) interface{}

	// Visit a parse tree produced by MDLParser#widgetV3.
	VisitWidgetV3(ctx *WidgetV3Context) interface{}

	// Visit a parse tree produced by MDLParser#widgetTypeV3.
	VisitWidgetTypeV3(ctx *WidgetTypeV3Context) interface{}

	// Visit a parse tree produced by MDLParser#widgetPropertiesV3.
	VisitWidgetPropertiesV3(ctx *WidgetPropertiesV3Context) interface{}

	// Visit a parse tree produced by MDLParser#widgetPropertyV3.
	VisitWidgetPropertyV3(ctx *WidgetPropertyV3Context) interface{}

	// Visit a parse tree produced by MDLParser#filterTypeValue.
	VisitFilterTypeValue(ctx *FilterTypeValueContext) interface{}

	// Visit a parse tree produced by MDLParser#attributeListV3.
	VisitAttributeListV3(ctx *AttributeListV3Context) interface{}

	// Visit a parse tree produced by MDLParser#dataSourceExprV3.
	VisitDataSourceExprV3(ctx *DataSourceExprV3Context) interface{}

	// Visit a parse tree produced by MDLParser#actionExprV3.
	VisitActionExprV3(ctx *ActionExprV3Context) interface{}

	// Visit a parse tree produced by MDLParser#microflowArgsV3.
	VisitMicroflowArgsV3(ctx *MicroflowArgsV3Context) interface{}

	// Visit a parse tree produced by MDLParser#microflowArgV3.
	VisitMicroflowArgV3(ctx *MicroflowArgV3Context) interface{}

	// Visit a parse tree produced by MDLParser#attributePathV3.
	VisitAttributePathV3(ctx *AttributePathV3Context) interface{}

	// Visit a parse tree produced by MDLParser#stringExprV3.
	VisitStringExprV3(ctx *StringExprV3Context) interface{}

	// Visit a parse tree produced by MDLParser#paramListV3.
	VisitParamListV3(ctx *ParamListV3Context) interface{}

	// Visit a parse tree produced by MDLParser#paramAssignmentV3.
	VisitParamAssignmentV3(ctx *ParamAssignmentV3Context) interface{}

	// Visit a parse tree produced by MDLParser#renderModeV3.
	VisitRenderModeV3(ctx *RenderModeV3Context) interface{}

	// Visit a parse tree produced by MDLParser#buttonStyleV3.
	VisitButtonStyleV3(ctx *ButtonStyleV3Context) interface{}

	// Visit a parse tree produced by MDLParser#desktopWidthV3.
	VisitDesktopWidthV3(ctx *DesktopWidthV3Context) interface{}

	// Visit a parse tree produced by MDLParser#selectionModeV3.
	VisitSelectionModeV3(ctx *SelectionModeV3Context) interface{}

	// Visit a parse tree produced by MDLParser#propertyValueV3.
	VisitPropertyValueV3(ctx *PropertyValueV3Context) interface{}

	// Visit a parse tree produced by MDLParser#designPropertyListV3.
	VisitDesignPropertyListV3(ctx *DesignPropertyListV3Context) interface{}

	// Visit a parse tree produced by MDLParser#designPropertyEntryV3.
	VisitDesignPropertyEntryV3(ctx *DesignPropertyEntryV3Context) interface{}

	// Visit a parse tree produced by MDLParser#widgetBodyV3.
	VisitWidgetBodyV3(ctx *WidgetBodyV3Context) interface{}

	// Visit a parse tree produced by MDLParser#createNotebookStatement.
	VisitCreateNotebookStatement(ctx *CreateNotebookStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#notebookOptions.
	VisitNotebookOptions(ctx *NotebookOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#notebookOption.
	VisitNotebookOption(ctx *NotebookOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#notebookPage.
	VisitNotebookPage(ctx *NotebookPageContext) interface{}

	// Visit a parse tree produced by MDLParser#createDatabaseConnectionStatement.
	VisitCreateDatabaseConnectionStatement(ctx *CreateDatabaseConnectionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#databaseConnectionOption.
	VisitDatabaseConnectionOption(ctx *DatabaseConnectionOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#databaseQuery.
	VisitDatabaseQuery(ctx *DatabaseQueryContext) interface{}

	// Visit a parse tree produced by MDLParser#databaseQueryMapping.
	VisitDatabaseQueryMapping(ctx *DatabaseQueryMappingContext) interface{}

	// Visit a parse tree produced by MDLParser#createConstantStatement.
	VisitCreateConstantStatement(ctx *CreateConstantStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#constantOptions.
	VisitConstantOptions(ctx *ConstantOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#constantOption.
	VisitConstantOption(ctx *ConstantOptionContext) interface{}

	// Visit a parse tree produced by MDLParser#createConfigurationStatement.
	VisitCreateConfigurationStatement(ctx *CreateConfigurationStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createRestClientStatement.
	VisitCreateRestClientStatement(ctx *CreateRestClientStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#restClientBaseUrl.
	VisitRestClientBaseUrl(ctx *RestClientBaseUrlContext) interface{}

	// Visit a parse tree produced by MDLParser#restClientAuthentication.
	VisitRestClientAuthentication(ctx *RestClientAuthenticationContext) interface{}

	// Visit a parse tree produced by MDLParser#restAuthValue.
	VisitRestAuthValue(ctx *RestAuthValueContext) interface{}

	// Visit a parse tree produced by MDLParser#restOperationDef.
	VisitRestOperationDef(ctx *RestOperationDefContext) interface{}

	// Visit a parse tree produced by MDLParser#restHttpMethod.
	VisitRestHttpMethod(ctx *RestHttpMethodContext) interface{}

	// Visit a parse tree produced by MDLParser#restOperationClause.
	VisitRestOperationClause(ctx *RestOperationClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#restHeaderValue.
	VisitRestHeaderValue(ctx *RestHeaderValueContext) interface{}

	// Visit a parse tree produced by MDLParser#restResponseSpec.
	VisitRestResponseSpec(ctx *RestResponseSpecContext) interface{}

	// Visit a parse tree produced by MDLParser#createIndexStatement.
	VisitCreateIndexStatement(ctx *CreateIndexStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createODataClientStatement.
	VisitCreateODataClientStatement(ctx *CreateODataClientStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createODataServiceStatement.
	VisitCreateODataServiceStatement(ctx *CreateODataServiceStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#odataPropertyValue.
	VisitOdataPropertyValue(ctx *OdataPropertyValueContext) interface{}

	// Visit a parse tree produced by MDLParser#odataPropertyAssignment.
	VisitOdataPropertyAssignment(ctx *OdataPropertyAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#odataAlterAssignment.
	VisitOdataAlterAssignment(ctx *OdataAlterAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#odataAuthenticationClause.
	VisitOdataAuthenticationClause(ctx *OdataAuthenticationClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#odataAuthType.
	VisitOdataAuthType(ctx *OdataAuthTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#publishEntityBlock.
	VisitPublishEntityBlock(ctx *PublishEntityBlockContext) interface{}

	// Visit a parse tree produced by MDLParser#exposeClause.
	VisitExposeClause(ctx *ExposeClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#exposeMember.
	VisitExposeMember(ctx *ExposeMemberContext) interface{}

	// Visit a parse tree produced by MDLParser#exposeMemberOptions.
	VisitExposeMemberOptions(ctx *ExposeMemberOptionsContext) interface{}

	// Visit a parse tree produced by MDLParser#createExternalEntityStatement.
	VisitCreateExternalEntityStatement(ctx *CreateExternalEntityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#createNavigationStatement.
	VisitCreateNavigationStatement(ctx *CreateNavigationStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#odataHeadersClause.
	VisitOdataHeadersClause(ctx *OdataHeadersClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#odataHeaderEntry.
	VisitOdataHeaderEntry(ctx *OdataHeaderEntryContext) interface{}

	// Visit a parse tree produced by MDLParser#createBusinessEventServiceStatement.
	VisitCreateBusinessEventServiceStatement(ctx *CreateBusinessEventServiceStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#businessEventMessageDef.
	VisitBusinessEventMessageDef(ctx *BusinessEventMessageDefContext) interface{}

	// Visit a parse tree produced by MDLParser#businessEventAttrDef.
	VisitBusinessEventAttrDef(ctx *BusinessEventAttrDefContext) interface{}

	// Visit a parse tree produced by MDLParser#createWorkflowStatement.
	VisitCreateWorkflowStatement(ctx *CreateWorkflowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowBody.
	VisitWorkflowBody(ctx *WorkflowBodyContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowActivityStmt.
	VisitWorkflowActivityStmt(ctx *WorkflowActivityStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowUserTaskStmt.
	VisitWorkflowUserTaskStmt(ctx *WorkflowUserTaskStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowBoundaryEventClause.
	VisitWorkflowBoundaryEventClause(ctx *WorkflowBoundaryEventClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowUserTaskOutcome.
	VisitWorkflowUserTaskOutcome(ctx *WorkflowUserTaskOutcomeContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowCallMicroflowStmt.
	VisitWorkflowCallMicroflowStmt(ctx *WorkflowCallMicroflowStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowParameterMapping.
	VisitWorkflowParameterMapping(ctx *WorkflowParameterMappingContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowCallWorkflowStmt.
	VisitWorkflowCallWorkflowStmt(ctx *WorkflowCallWorkflowStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowDecisionStmt.
	VisitWorkflowDecisionStmt(ctx *WorkflowDecisionStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowConditionOutcome.
	VisitWorkflowConditionOutcome(ctx *WorkflowConditionOutcomeContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowParallelSplitStmt.
	VisitWorkflowParallelSplitStmt(ctx *WorkflowParallelSplitStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowParallelPath.
	VisitWorkflowParallelPath(ctx *WorkflowParallelPathContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowJumpToStmt.
	VisitWorkflowJumpToStmt(ctx *WorkflowJumpToStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowWaitForTimerStmt.
	VisitWorkflowWaitForTimerStmt(ctx *WorkflowWaitForTimerStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowWaitForNotificationStmt.
	VisitWorkflowWaitForNotificationStmt(ctx *WorkflowWaitForNotificationStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#workflowAnnotationStmt.
	VisitWorkflowAnnotationStmt(ctx *WorkflowAnnotationStmtContext) interface{}

	// Visit a parse tree produced by MDLParser#alterSettingsClause.
	VisitAlterSettingsClause(ctx *AlterSettingsClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#settingsSection.
	VisitSettingsSection(ctx *SettingsSectionContext) interface{}

	// Visit a parse tree produced by MDLParser#settingsAssignment.
	VisitSettingsAssignment(ctx *SettingsAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#settingsValue.
	VisitSettingsValue(ctx *SettingsValueContext) interface{}

	// Visit a parse tree produced by MDLParser#dqlStatement.
	VisitDqlStatement(ctx *DqlStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#showStatement.
	VisitShowStatement(ctx *ShowStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#showWidgetsFilter.
	VisitShowWidgetsFilter(ctx *ShowWidgetsFilterContext) interface{}

	// Visit a parse tree produced by MDLParser#widgetTypeKeyword.
	VisitWidgetTypeKeyword(ctx *WidgetTypeKeywordContext) interface{}

	// Visit a parse tree produced by MDLParser#widgetCondition.
	VisitWidgetCondition(ctx *WidgetConditionContext) interface{}

	// Visit a parse tree produced by MDLParser#widgetPropertyAssignment.
	VisitWidgetPropertyAssignment(ctx *WidgetPropertyAssignmentContext) interface{}

	// Visit a parse tree produced by MDLParser#widgetPropertyValue.
	VisitWidgetPropertyValue(ctx *WidgetPropertyValueContext) interface{}

	// Visit a parse tree produced by MDLParser#describeStatement.
	VisitDescribeStatement(ctx *DescribeStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#catalogSelectQuery.
	VisitCatalogSelectQuery(ctx *CatalogSelectQueryContext) interface{}

	// Visit a parse tree produced by MDLParser#catalogJoinClause.
	VisitCatalogJoinClause(ctx *CatalogJoinClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#catalogTableName.
	VisitCatalogTableName(ctx *CatalogTableNameContext) interface{}

	// Visit a parse tree produced by MDLParser#oqlQuery.
	VisitOqlQuery(ctx *OqlQueryContext) interface{}

	// Visit a parse tree produced by MDLParser#oqlQueryTerm.
	VisitOqlQueryTerm(ctx *OqlQueryTermContext) interface{}

	// Visit a parse tree produced by MDLParser#selectClause.
	VisitSelectClause(ctx *SelectClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#selectList.
	VisitSelectList(ctx *SelectListContext) interface{}

	// Visit a parse tree produced by MDLParser#selectItem.
	VisitSelectItem(ctx *SelectItemContext) interface{}

	// Visit a parse tree produced by MDLParser#selectAlias.
	VisitSelectAlias(ctx *SelectAliasContext) interface{}

	// Visit a parse tree produced by MDLParser#fromClause.
	VisitFromClause(ctx *FromClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#tableReference.
	VisitTableReference(ctx *TableReferenceContext) interface{}

	// Visit a parse tree produced by MDLParser#joinClause.
	VisitJoinClause(ctx *JoinClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#associationPath.
	VisitAssociationPath(ctx *AssociationPathContext) interface{}

	// Visit a parse tree produced by MDLParser#joinType.
	VisitJoinType(ctx *JoinTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#whereClause.
	VisitWhereClause(ctx *WhereClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#groupByClause.
	VisitGroupByClause(ctx *GroupByClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#havingClause.
	VisitHavingClause(ctx *HavingClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#orderByClause.
	VisitOrderByClause(ctx *OrderByClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#orderByList.
	VisitOrderByList(ctx *OrderByListContext) interface{}

	// Visit a parse tree produced by MDLParser#orderByItem.
	VisitOrderByItem(ctx *OrderByItemContext) interface{}

	// Visit a parse tree produced by MDLParser#groupByList.
	VisitGroupByList(ctx *GroupByListContext) interface{}

	// Visit a parse tree produced by MDLParser#limitOffsetClause.
	VisitLimitOffsetClause(ctx *LimitOffsetClauseContext) interface{}

	// Visit a parse tree produced by MDLParser#utilityStatement.
	VisitUtilityStatement(ctx *UtilityStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#searchStatement.
	VisitSearchStatement(ctx *SearchStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#connectStatement.
	VisitConnectStatement(ctx *ConnectStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#disconnectStatement.
	VisitDisconnectStatement(ctx *DisconnectStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#updateStatement.
	VisitUpdateStatement(ctx *UpdateStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#checkStatement.
	VisitCheckStatement(ctx *CheckStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#buildStatement.
	VisitBuildStatement(ctx *BuildStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#executeScriptStatement.
	VisitExecuteScriptStatement(ctx *ExecuteScriptStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#executeRuntimeStatement.
	VisitExecuteRuntimeStatement(ctx *ExecuteRuntimeStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#lintStatement.
	VisitLintStatement(ctx *LintStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#lintTarget.
	VisitLintTarget(ctx *LintTargetContext) interface{}

	// Visit a parse tree produced by MDLParser#lintFormat.
	VisitLintFormat(ctx *LintFormatContext) interface{}

	// Visit a parse tree produced by MDLParser#useSessionStatement.
	VisitUseSessionStatement(ctx *UseSessionStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#sessionIdList.
	VisitSessionIdList(ctx *SessionIdListContext) interface{}

	// Visit a parse tree produced by MDLParser#sessionId.
	VisitSessionId(ctx *SessionIdContext) interface{}

	// Visit a parse tree produced by MDLParser#introspectApiStatement.
	VisitIntrospectApiStatement(ctx *IntrospectApiStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#debugStatement.
	VisitDebugStatement(ctx *DebugStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlConnect.
	VisitSqlConnect(ctx *SqlConnectContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlDisconnect.
	VisitSqlDisconnect(ctx *SqlDisconnectContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlConnections.
	VisitSqlConnections(ctx *SqlConnectionsContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlShowTables.
	VisitSqlShowTables(ctx *SqlShowTablesContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlDescribeTable.
	VisitSqlDescribeTable(ctx *SqlDescribeTableContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlGenerateConnector.
	VisitSqlGenerateConnector(ctx *SqlGenerateConnectorContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlQuery.
	VisitSqlQuery(ctx *SqlQueryContext) interface{}

	// Visit a parse tree produced by MDLParser#sqlPassthrough.
	VisitSqlPassthrough(ctx *SqlPassthroughContext) interface{}

	// Visit a parse tree produced by MDLParser#importFromQuery.
	VisitImportFromQuery(ctx *ImportFromQueryContext) interface{}

	// Visit a parse tree produced by MDLParser#importMapping.
	VisitImportMapping(ctx *ImportMappingContext) interface{}

	// Visit a parse tree produced by MDLParser#linkLookup.
	VisitLinkLookup(ctx *LinkLookupContext) interface{}

	// Visit a parse tree produced by MDLParser#linkDirect.
	VisitLinkDirect(ctx *LinkDirectContext) interface{}

	// Visit a parse tree produced by MDLParser#helpStatement.
	VisitHelpStatement(ctx *HelpStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#defineFragmentStatement.
	VisitDefineFragmentStatement(ctx *DefineFragmentStatementContext) interface{}

	// Visit a parse tree produced by MDLParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#orExpression.
	VisitOrExpression(ctx *OrExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#andExpression.
	VisitAndExpression(ctx *AndExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#notExpression.
	VisitNotExpression(ctx *NotExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#comparisonExpression.
	VisitComparisonExpression(ctx *ComparisonExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#comparisonOperator.
	VisitComparisonOperator(ctx *ComparisonOperatorContext) interface{}

	// Visit a parse tree produced by MDLParser#additiveExpression.
	VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#multiplicativeExpression.
	VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#unaryExpression.
	VisitUnaryExpression(ctx *UnaryExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#primaryExpression.
	VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#caseExpression.
	VisitCaseExpression(ctx *CaseExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#ifThenElseExpression.
	VisitIfThenElseExpression(ctx *IfThenElseExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#castExpression.
	VisitCastExpression(ctx *CastExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#castDataType.
	VisitCastDataType(ctx *CastDataTypeContext) interface{}

	// Visit a parse tree produced by MDLParser#aggregateFunction.
	VisitAggregateFunction(ctx *AggregateFunctionContext) interface{}

	// Visit a parse tree produced by MDLParser#functionCall.
	VisitFunctionCall(ctx *FunctionCallContext) interface{}

	// Visit a parse tree produced by MDLParser#functionName.
	VisitFunctionName(ctx *FunctionNameContext) interface{}

	// Visit a parse tree produced by MDLParser#argumentList.
	VisitArgumentList(ctx *ArgumentListContext) interface{}

	// Visit a parse tree produced by MDLParser#atomicExpression.
	VisitAtomicExpression(ctx *AtomicExpressionContext) interface{}

	// Visit a parse tree produced by MDLParser#expressionList.
	VisitExpressionList(ctx *ExpressionListContext) interface{}

	// Visit a parse tree produced by MDLParser#qualifiedName.
	VisitQualifiedName(ctx *QualifiedNameContext) interface{}

	// Visit a parse tree produced by MDLParser#identifierOrKeyword.
	VisitIdentifierOrKeyword(ctx *IdentifierOrKeywordContext) interface{}

	// Visit a parse tree produced by MDLParser#literal.
	VisitLiteral(ctx *LiteralContext) interface{}

	// Visit a parse tree produced by MDLParser#arrayLiteral.
	VisitArrayLiteral(ctx *ArrayLiteralContext) interface{}

	// Visit a parse tree produced by MDLParser#booleanLiteral.
	VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{}

	// Visit a parse tree produced by MDLParser#docComment.
	VisitDocComment(ctx *DocCommentContext) interface{}

	// Visit a parse tree produced by MDLParser#annotation.
	VisitAnnotation(ctx *AnnotationContext) interface{}

	// Visit a parse tree produced by MDLParser#annotationName.
	VisitAnnotationName(ctx *AnnotationNameContext) interface{}

	// Visit a parse tree produced by MDLParser#annotationParams.
	VisitAnnotationParams(ctx *AnnotationParamsContext) interface{}

	// Visit a parse tree produced by MDLParser#annotationParam.
	VisitAnnotationParam(ctx *AnnotationParamContext) interface{}

	// Visit a parse tree produced by MDLParser#annotationValue.
	VisitAnnotationValue(ctx *AnnotationValueContext) interface{}

	// Visit a parse tree produced by MDLParser#commonNameKeyword.
	VisitCommonNameKeyword(ctx *CommonNameKeywordContext) interface{}

	// Visit a parse tree produced by MDLParser#keyword.
	VisitKeyword(ctx *KeywordContext) interface{}
}
