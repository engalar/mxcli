// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	Register(SyntaxFeature{
		Path:    "microflow",
		Summary: "Programmatic logic — variables, object operations, control flow, and integrations",
		Keywords: []string{
			"microflow", "nanoflow", "logic", "automation",
			"action", "activity", "flow",
		},
		Syntax:  "CREATE MICROFLOW Module.Name ($Param: Type) RETURNS Type AS $Result\nBEGIN\n  <statements>\nEND;",
		Example: "CREATE MICROFLOW MyModule.ACT_CreateOrder ($Code: String)\nRETURNS MyModule.Order AS $NewOrder\nBEGIN\n  $NewOrder = CREATE MyModule.Order (OrderNumber = $Code);\n  COMMIT $NewOrder;\n  RETURN $NewOrder;\nEND;",
		SeeAlso: []string{"microflow.create", "microflow.variables", "microflow.control-flow"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.create",
		Summary: "Create a microflow with parameters, return type, and body",
		Keywords: []string{
			"create microflow", "new microflow", "define microflow",
			"parameters", "returns", "folder",
		},
		Syntax: "CREATE MICROFLOW Module.Name ($P1: String, $P2: Integer)\n  RETURNS Type AS $Result\n  [FOLDER 'FolderPath']\nBEGIN\n  <statements>\nEND;",
		Example: "CREATE MICROFLOW MyModule.ACT_CreateOrder (\n  $CustomerCode: String,\n  $Quantity: Integer\n)\nRETURNS MyModule.Order AS $NewOrder\nFOLDER 'Orders'\nBEGIN\n  $NewOrder = CREATE MyModule.Order (\n    OrderNumber = 'ORD-001',\n    Quantity = $Quantity\n  );\n  COMMIT $NewOrder;\n  RETURN $NewOrder;\nEND;",
		SeeAlso: []string{"microflow.nanoflow", "microflow.variables"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.variables",
		Summary: "Declare variables, assign values, and set object attributes",
		Keywords: []string{
			"declare", "variable", "set", "assign", "change",
			"attribute", "expression",
		},
		Syntax: "DECLARE $Var Type;\nDECLARE $Var Type = expression;\nSET $Var = expression;\nSET $Var/Attribute = expression;",
		Example: "DECLARE $Count Integer = 0;\nDECLARE $Name String;\nSET $Name = 'Hello';\nSET $Order/Status = 'Pending';",
		SeeAlso: []string{"microflow.object-operations"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.retrieve",
		Summary: "Query the database with WHERE, SORT BY, LIMIT, and OFFSET",
		Keywords: []string{
			"retrieve", "query", "database", "where", "sort",
			"limit", "offset", "find", "fetch",
		},
		Syntax: "RETRIEVE $Var FROM Module.Entity\n  [WHERE condition]\n  [SORT BY attr ASC|DESC]\n  [LIMIT n] [OFFSET n];",
		Example: "RETRIEVE $Customer FROM MyModule.Customer\n  WHERE Code = $CustomerCode\n  LIMIT 1;\n\nRETRIEVE $Orders FROM MyModule.Order\n  WHERE Status = 'Pending'\n  SORT BY CreateDate DESC\n  LIMIT 10 OFFSET 0;",
		SeeAlso: []string{"microflow.object-operations", "xpath"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.control-flow",
		Summary: "IF/ELSIF/ELSE, LOOP, WHILE, BREAK, CONTINUE, RETURN",
		Keywords: []string{
			"if", "elsif", "else", "then", "end if",
			"loop", "while", "break", "continue", "return",
			"conditional", "branch", "iterate",
		},
		Syntax: "IF condition THEN\n  ...\nELSIF condition THEN\n  ...\nELSE\n  ...\nEND IF;\n\nLOOP $Item IN $List BEGIN ... END LOOP;\nWHILE condition BEGIN ... END WHILE;\nRETURN $Value;\nRETURN empty;",
		Example: "IF $Customer = empty THEN\n  LOG ERROR NODE 'Svc' 'Not found';\n  RETURN empty;\nELSIF $Customer/Active = false THEN\n  LOG WARNING 'Inactive customer';\nELSE\n  CHANGE $Customer (LastAccess = [%CurrentDateTime%]);\nEND IF;\n\nLOOP $Item IN $OrderLines BEGIN\n  COMMIT $Item;\nEND LOOP;",
		SeeAlso: []string{"microflow.variables", "microflow.error-handling"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.error-handling",
		Summary: "Error handling with ON ERROR, THROW, CONTINUE, ROLLBACK",
		Keywords: []string{
			"error", "error handling", "on error", "continue",
			"rollback", "throw", "exception", "try", "catch",
		},
		Syntax: "COMMIT $Obj ON ERROR CONTINUE;\nCOMMIT $Obj ON ERROR ROLLBACK;\nCOMMIT $Obj ON ERROR { <statements> };\nCOMMIT $Obj ON ERROR WITHOUT ROLLBACK { <statements> };",
		Example: "COMMIT $Order ON ERROR {\n  LOG ERROR 'Failed to save order';\n  RETURN empty;\n};\n\nCOMMIT $Batch ON ERROR WITHOUT ROLLBACK {\n  LOG WARNING 'Batch save failed, continuing';\n};",
		SeeAlso: []string{"microflow.control-flow"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.object-operations",
		Summary: "CREATE, CHANGE, COMMIT, ROLLBACK, and DELETE objects",
		Keywords: []string{
			"create object", "change object", "commit", "rollback",
			"delete", "save", "persist", "modify object",
			"with events", "refresh",
		},
		Syntax: "$Obj = CREATE Module.Entity (Attr = value);\nCHANGE $Obj (Attr = value);\nCOMMIT $Obj;\nCOMMIT $Obj WITH EVENTS;\nCOMMIT $Obj REFRESH;\nCOMMIT $Obj WITH EVENTS REFRESH;\nDELETE $Obj;\nROLLBACK $Obj;",
		Example: "$NewOrder = CREATE MyModule.Order (\n  OrderNumber = 'ORD-001',\n  Quantity = $Quantity,\n  CreateDate = [%CurrentDateTime%]\n);\n\nCHANGE $NewOrder (MyModule.Order_Customer = $Customer);\nCOMMIT $NewOrder WITH EVENTS;\nDELETE $OldOrder;\nROLLBACK $DraftOrder;",
		SeeAlso: []string{"microflow.retrieve", "microflow.variables"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.list-operations",
		Summary: "List manipulation — HEAD, TAIL, FIND, FILTER, SORT, UNION, aggregates",
		Keywords: []string{
			"list", "head", "tail", "find", "filter", "sort",
			"union", "intersect", "subtract", "count", "sum",
			"average", "aggregate", "add to list", "remove from list",
			"create list",
		},
		Syntax: "$List = CREATE LIST OF Module.Entity;\nADD $Item TO $List;\nREMOVE $Item FROM $List;\n$Result = HEAD($List);\n$Result = TAIL($List);\n$Result = FIND($List, condition);\n$Result = FILTER($List, condition);\n$Result = SORT($List, attr ASC);\n$Result = UNION($L1, $L2);\n$Result = INTERSECT($L1, $L2);\n$Result = SUBTRACT($L1, $L2);\n$Count = COUNT($List);\n$Sum = SUM($List.Attr);\n$Avg = AVERAGE($List.Attr);",
		Example: "$AllOrders = CREATE LIST OF MyModule.Order;\nADD $NewOrder TO $AllOrders;\n$First = HEAD($AllOrders);\n$Pending = FILTER($AllOrders, Status = 'Pending');\n$Sorted = SORT($Pending, CreateDate DESC);\n$Total = SUM($AllOrders.Amount);",
		SeeAlso: []string{"microflow.retrieve"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.logging",
		Summary: "LOG statements with level, node, message templates",
		Keywords: []string{
			"log", "logging", "info", "warning", "error", "debug",
			"trace", "critical", "node", "message template",
		},
		Syntax: "LOG LEVEL [NODE 'Name'] 'message';\nLOG LEVEL 'template {1}' WITH ({1} = $value);\n\n-- Levels: INFO, WARNING, ERROR, DEBUG, TRACE, CRITICAL",
		Example: "LOG INFO NODE 'OrderService' 'Order created successfully';\nLOG WARNING 'Customer not found';\nLOG ERROR 'Failed to process {1}' WITH (\n  {1} = $OrderNumber\n);",
	})

	Register(SyntaxFeature{
		Path:    "microflow.show-page",
		Summary: "Open and close pages from microflows",
		Keywords: []string{
			"show page", "open page", "close page", "display page",
			"navigate", "page action",
		},
		Syntax: "SHOW PAGE Module.Page;\nSHOW PAGE Module.Page ($Param = $value);\nCLOSE PAGE;",
		Example: "SHOW PAGE MyModule.OrderDetail ($Order = $NewOrder);\nCLOSE PAGE;",
		SeeAlso: []string{"page"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.call",
		Summary: "Call microflows and Java actions with parameters",
		Keywords: []string{
			"call microflow", "call java action", "invoke",
			"sub-microflow", "java action", "parameter passing",
		},
		Syntax: "$Result = CALL MICROFLOW Module.Name (Param = value);\n$Result = CALL JAVA ACTION Module.Name (Param = value);",
		Example: "$IsValid = CALL MICROFLOW MyModule.ValidateOrder (\n  Order = $NewOrder\n);\n\n$Token = CALL JAVA ACTION MyModule.GenerateToken (\n  UserId = $User/Id\n);",
		SeeAlso: []string{"java-action", "microflow.create"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.nanoflow",
		Summary: "CREATE NANOFLOW — client-side logic, same syntax as microflow",
		Keywords: []string{
			"nanoflow", "create nanoflow", "client-side",
			"offline", "client logic",
		},
		Syntax:  "CREATE NANOFLOW Module.Name ($Param: Type) RETURNS Type AS $Result\nBEGIN\n  <statements>\nEND;",
		Example: "CREATE NANOFLOW MyModule.NF_ValidateInput ($Input: String)\nRETURNS Boolean AS $IsValid\nBEGIN\n  IF $Input = empty THEN\n    VALIDATION FEEDBACK $Input MESSAGE 'Required';\n    RETURN false;\n  END IF;\n  RETURN true;\nEND;",
		SeeAlso: []string{"microflow.create"},
	})

	Register(SyntaxFeature{
		Path:    "microflow.validation",
		Summary: "Show validation feedback on object attributes",
		Keywords: []string{
			"validation", "feedback", "validation feedback",
			"error message", "field error", "form validation",
		},
		Syntax: "VALIDATION FEEDBACK $Obj/Attr MESSAGE 'error text';\nVALIDATION FEEDBACK $Obj/Attr MESSAGE '{1} is invalid'\n  OBJECTS [$Value];",
		Example: "VALIDATION FEEDBACK $Order/Quantity MESSAGE 'Quantity must be positive';\nVALIDATION FEEDBACK $Customer/Email MESSAGE '{1} is not valid'\n  OBJECTS [$Customer/Email];",
		SeeAlso: []string{"microflow.error-handling"},
	})
}
