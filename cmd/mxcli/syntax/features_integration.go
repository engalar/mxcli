// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	// ── Integration overview ──────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "integration",
		Summary: "Unified discovery of all external services and integration assets",
		Keywords: []string{
			"integration", "services", "external", "contract",
			"odata", "rest", "business events", "database",
		},
		Syntax: "SHOW ODATA CLIENTS [IN Module];\nSHOW REST CLIENTS [IN Module];\nSHOW PUBLISHED REST SERVICES [IN Module];\nSHOW BUSINESS EVENT SERVICES [IN Module];\nSHOW DATABASE CONNECTIONS [IN Module];\nSHOW EXTERNAL ENTITIES [IN Module];\nSHOW EXTERNAL ACTIONS [IN Module];",
		Example: "SHOW ODATA CLIENTS;\nSHOW REST CLIENTS IN MyModule;\nSHOW EXTERNAL ENTITIES;\nSELECT * FROM CATALOG.REST_CLIENTS;",
		SeeAlso: []string{"odata", "rest", "sql", "business-events"},
	})

	// ── OData ─────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "odata",
		Summary: "OData clients, services, and external entities",
		Keywords: []string{
			"odata", "consumed odata", "published odata",
			"external entity", "external entities", "metadata",
		},
		Syntax: "SHOW ODATA CLIENTS [IN Module];\nSHOW ODATA SERVICES [IN Module];\nSHOW EXTERNAL ENTITIES [IN Module];\nSHOW EXTERNAL ACTIONS [IN Module];\nDESCRIBE ODATA CLIENT Module.Name;\nDESCRIBE ODATA SERVICE Module.Name;",
		Example: "SHOW ODATA CLIENTS;\nDESCRIBE ODATA CLIENT MyModule.ExternalAPI;\nSHOW EXTERNAL ENTITIES IN MyModule;",
		SeeAlso: []string{"odata.consume", "odata.show", "integration"},
	})

	Register(SyntaxFeature{
		Path:    "odata.consume",
		Summary: "Create consumed OData services and external entities",
		Keywords: []string{
			"create odata client", "consume odata", "external entity",
			"metadata url", "odata4", "headers", "proxy",
		},
		Syntax: "CREATE ODATA CLIENT Module.Name (\n  Version: '1.0',\n  ODataVersion: OData4,\n  MetadataUrl: 'https://.../$metadata',\n  Timeout: 300\n)\n[HEADERS ('Key': 'Value')];\n\nCREATE EXTERNAL ENTITY Module.Name\n  FROM ODATA CLIENT Module.Client\n  (EntitySet: 'Name', RemoteName: 'Name')\n  (Attr: Type, ...);\n\nCREATE EXTERNAL ENTITIES FROM Module.Client\n  [INTO Module] [ENTITIES (Name1, Name2)];",
		Example: "CREATE ODATA CLIENT MyModule.SalesforceAPI (\n  Version: '1.0',\n  ODataVersion: OData4,\n  MetadataUrl: 'https://api.example.com/odata/$metadata',\n  Timeout: 300\n);\n\nCREATE EXTERNAL ENTITIES FROM MyModule.SalesforceAPI INTO Integration;",
		SeeAlso: []string{"odata", "odata.show"},
	})

	Register(SyntaxFeature{
		Path:    "odata.show",
		Summary: "Browse cached OData contracts — entities, actions, properties",
		Keywords: []string{
			"contract", "show contract", "describe contract",
			"metadata", "entity type", "action", "navigation property",
		},
		Syntax: "SHOW CONTRACT ENTITIES FROM Module.Client;\nSHOW CONTRACT ACTIONS FROM Module.Client;\nDESCRIBE CONTRACT ENTITY Module.Client.EntityName;\nDESCRIBE CONTRACT ENTITY Module.Client.EntityName FORMAT mdl;\nDESCRIBE CONTRACT ACTION Module.Client.ActionName;",
		Example: "SHOW CONTRACT ENTITIES FROM MyModule.SalesforceAPI;\nDESCRIBE CONTRACT ENTITY MyModule.SalesforceAPI.Product;\nDESCRIBE CONTRACT ENTITY MyModule.SalesforceAPI.Product FORMAT mdl;",
		SeeAlso: []string{"odata", "odata.consume"},
	})

	// ── REST ──────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "rest",
		Summary: "Consumed and published REST services",
		Keywords: []string{
			"rest", "rest client", "rest service",
			"published rest", "api", "http",
		},
		Syntax: "SHOW REST CLIENTS [IN Module];\nSHOW PUBLISHED REST SERVICES [IN Module];\nDESCRIBE REST CLIENT Module.Name;\nDESCRIBE PUBLISHED REST SERVICE Module.Name;",
		Example: "SHOW REST CLIENTS;\nDESCRIBE REST CLIENT MyModule.PetStoreAPI;\nSHOW PUBLISHED REST SERVICES IN MyModule;",
		SeeAlso: []string{"rest.consumed", "rest.published", "integration"},
	})

	Register(SyntaxFeature{
		Path:    "rest.consumed",
		Summary: "Create consumed REST clients with operations, mappings, and authentication",
		Keywords: []string{
			"create rest client", "consume rest", "rest operation",
			"get", "post", "put", "delete", "patch",
			"body", "response", "mapping", "authentication",
			"json structure", "import mapping", "export mapping",
		},
		Syntax: "CREATE [OR MODIFY] REST CLIENT Module.Name (\n  BaseUrl: 'https://...',\n  Authentication: NONE | BASIC (...)\n)\n{\n  OPERATION Name {\n    Method: GET|POST|PUT|DELETE|PATCH,\n    Path: '/path/{param}',\n    Parameters: ($param: Type),\n    Headers: ('Key' = 'Value'),\n    Body: JSON FROM $var | MAPPING Entity { ... },\n    Response: JSON AS $var | MAPPING Entity { ... }\n  }\n};",
		Example: "CREATE REST CLIENT Module.PetStore (\n  BaseUrl: 'https://petstore.example.com/api',\n  Authentication: NONE\n)\n{\n  OPERATION GetPet {\n    Method: GET,\n    Path: '/pets/{id}',\n    Parameters: ($id: String),\n    Response: JSON AS $Result\n  }\n};",
		SeeAlso: []string{"rest", "rest.published"},
	})

	Register(SyntaxFeature{
		Path:    "rest.published",
		Summary: "Create and manage published REST services with resources and operations",
		Keywords: []string{
			"create published rest", "publish rest", "rest resource",
			"rest operation", "microflow", "path parameter",
			"grant access", "revoke access",
		},
		Syntax: "CREATE [OR REPLACE] PUBLISHED REST SERVICE Module.Name (\n  Path: 'rest/api/v1',\n  Version: '1.0.0',\n  ServiceName: 'My API'\n)\n{\n  RESOURCE 'name' {\n    GET '' MICROFLOW Module.GetAll;\n    GET '{id}' MICROFLOW Module.GetById;\n    POST '' MICROFLOW Module.Create;\n  }\n};\n\nALTER PUBLISHED REST SERVICE Module.Name SET Version = '2.0.0';\nALTER PUBLISHED REST SERVICE Module.Name ADD RESOURCE 'items' { ... };\nALTER PUBLISHED REST SERVICE Module.Name DROP RESOURCE 'legacy';\nDROP PUBLISHED REST SERVICE Module.Name;",
		Example: "CREATE PUBLISHED REST SERVICE Module.OrderAPI (\n  Path: 'rest/orders/v1',\n  Version: '1.0.0',\n  ServiceName: 'Order API'\n)\n{\n  RESOURCE 'orders' {\n    GET '' MICROFLOW Module.GetAllOrders;\n    GET '{id}' MICROFLOW Module.GetOrderById;\n    POST '' MICROFLOW Module.CreateOrder;\n    DELETE '{id}' MICROFLOW Module.DeleteOrder;\n  }\n};\n\nGRANT ACCESS ON PUBLISHED REST SERVICE Module.OrderAPI\n  TO Module.User, Module.Admin;",
		SeeAlso: []string{"rest", "rest.consumed"},
	})

	// ── SQL ───────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "sql",
		Summary: "External SQL queries against PostgreSQL, Oracle, SQL Server",
		Keywords: []string{
			"sql", "external sql", "database", "postgres",
			"oracle", "sqlserver", "mssql", "query",
		},
		Syntax: "SQL CONNECT <driver> '<dsn>' AS <alias>;\nSQL <alias> SHOW TABLES;\nSQL <alias> SELECT ...;\nSQL CONNECTIONS;\nSQL DISCONNECT <alias>;",
		Example: "SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;\nSQL source SHOW TABLES;\nSQL source SELECT * FROM users WHERE active = true LIMIT 10;\nSQL DISCONNECT source;",
		SeeAlso: []string{"sql.connect", "sql.query", "sql.import", "sql.generate"},
	})

	Register(SyntaxFeature{
		Path:    "sql.connect",
		Summary: "Connect to external databases with credential isolation",
		Keywords: []string{
			"sql connect", "database connect", "dsn",
			"postgres", "oracle", "sqlserver", "driver",
			"connections.yaml", "credential",
		},
		Syntax: "SQL CONNECT <driver> '<dsn>' AS <alias>;\nSQL CONNECT postgres '<dsn>' AS <alias>;\nSQL CONNECT oracle '<dsn>' AS <alias>;\nSQL CONNECT sqlserver '<dsn>' AS <alias>;\nSQL CONNECTIONS;\nSQL DISCONNECT <alias>;",
		Example: "SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;\nSQL CONNECTIONS;\nSQL DISCONNECT source;",
		SeeAlso: []string{"sql", "sql.query"},
	})

	Register(SyntaxFeature{
		Path:    "sql.query",
		Summary: "Execute SQL queries and browse schema of external databases",
		Keywords: []string{
			"sql query", "sql select", "show tables",
			"describe table", "sql insert", "sql execute",
		},
		Syntax: "SQL <alias> SHOW TABLES;\nSQL <alias> SHOW VIEWS;\nSQL <alias> SHOW FUNCTIONS;\nSQL <alias> DESCRIBE <table>;\nSQL <alias> SELECT ...;\nSQL <alias> INSERT ...;",
		Example: "SQL source SHOW TABLES;\nSQL source DESCRIBE users;\nSQL source SELECT * FROM users WHERE active = true LIMIT 10;",
		SeeAlso: []string{"sql.connect", "sql.import"},
	})

	Register(SyntaxFeature{
		Path:    "sql.import",
		Summary: "Import rows from external DB into Mendix app database",
		Keywords: []string{
			"import", "import from", "import into", "map",
			"batch", "link", "association", "data migration",
		},
		Syntax: "IMPORT FROM <alias> QUERY '<sql>'\n  INTO Module.Entity\n  MAP (col AS Attr, ...)\n  [LINK (col TO Assoc ON Attr, ...)]\n  [BATCH n]\n  [LIMIT n];",
		Example: "IMPORT FROM source QUERY 'SELECT name, email FROM employees'\n  INTO HR.Employee\n  MAP (name AS Name, email AS Email);\n\nIMPORT FROM source QUERY 'SELECT name, dept_name FROM employees'\n  INTO HR.Employee\n  MAP (name AS Name)\n  LINK (dept_name TO Employee_Department ON Name)\n  BATCH 500\n  LIMIT 10000;",
		SeeAlso: []string{"sql", "sql.connect"},
	})

	Register(SyntaxFeature{
		Path:    "sql.generate",
		Summary: "Auto-generate Database Connector MDL from external database schema",
		Keywords: []string{
			"generate connector", "database connector",
			"constants", "non-persistent entity", "jdbc",
		},
		Syntax: "SQL <alias> GENERATE CONNECTOR INTO Module\n  [TABLES (t1, t2)]\n  [VIEWS (v1, v2)]\n  [EXEC];",
		Example: "SQL source GENERATE CONNECTOR INTO HRModule;\nSQL source GENERATE CONNECTOR INTO HRModule TABLES (employees, departments) EXEC;",
		SeeAlso: []string{"sql", "sql.connect"},
	})

	// ── OQL ───────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "oql",
		Summary: "OQL query execution against a running Mendix runtime",
		Keywords: []string{
			"oql", "query", "runtime", "m2ee",
			"aggregate", "view entity", "mxcli oql",
		},
		Syntax: "mxcli oql -p app.mpr \"SELECT ...\";\nmxcli oql -p app.mpr --json \"SELECT ...\";\nmxcli oql --direct --host localhost --port 8090 --token 'pass' \"SELECT ...\";",
		Example: "mxcli oql -p app.mpr \"SELECT Name, Email FROM MyModule.Customer\";\nmxcli oql -p app.mpr --json \"SELECT count(c.ID) FROM MyModule.Order AS c\" | jq '.[0]';",
		SeeAlso: []string{"sql"},
	})

	// ── Business Events ───────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "business-events",
		Summary: "Business event services — publish and subscribe to events via Kafka",
		Keywords: []string{
			"business event", "business events", "kafka",
			"publish", "subscribe", "message", "event channel",
		},
		Syntax: "SHOW BUSINESS EVENT SERVICES [IN Module];\nSHOW BUSINESS EVENTS [IN Module];\nDESCRIBE BUSINESS EVENT SERVICE Module.Name;\nCREATE [OR REPLACE] BUSINESS EVENT SERVICE Module.Name (...) { ... };\nDROP BUSINESS EVENT SERVICE Module.Name;",
		Example: "SHOW BUSINESS EVENT SERVICES;\nDESCRIBE BUSINESS EVENT SERVICE Module.CustomerEventsApi;",
		SeeAlso: []string{"business-events.create", "integration"},
	})

	Register(SyntaxFeature{
		Path:    "business-events.create",
		Summary: "Create and drop business event service definitions with messages",
		Keywords: []string{
			"create business event", "drop business event",
			"message", "publish", "subscribe", "entity",
			"event name prefix",
		},
		Syntax: "CREATE [OR REPLACE] BUSINESS EVENT SERVICE Module.Name\n(\n  ServiceName: 'Name',\n  EventNamePrefix: ''\n)\n{\n  MESSAGE EventName (Attr: Type, ...) PUBLISH|SUBSCRIBE\n    ENTITY Module.PBE_Entity;\n};\n\nDROP BUSINESS EVENT SERVICE Module.Name;",
		Example: "CREATE BUSINESS EVENT SERVICE Module.CustomerEventsApi\n(\n  ServiceName: 'CustomerEventsApi',\n  EventNamePrefix: ''\n)\n{\n  MESSAGE CustomerChangedEvent (CustomerId: Long) PUBLISH\n    ENTITY Module.PBE_CustomerChangedEvent;\n};\n\nDROP BUSINESS EVENT SERVICE Module.CustomerEventsApi;",
		SeeAlso: []string{"business-events"},
	})

	// ── XPath ─────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "xpath",
		Summary: "XPath constraint syntax for filtering data in RETRIEVE, pages, and security",
		Keywords: []string{
			"xpath", "constraint", "where", "predicate",
			"filter", "retrieve", "association path",
		},
		Syntax: "WHERE [condition]\nWHERE [cond1][cond2]          -- implicit AND\nWHERE [cond1] AND [cond2]\nWHERE [cond1] OR [cond2]",
		Example: "RETRIEVE $Orders FROM Module.Order\n  WHERE [State = 'Completed'][IsPaid = true]\n  SORT BY OrderDate DESC;\n\n-- Association path traversal\nWHERE [Module.Order_Customer/Module.Customer/Name = $Name]\n\n-- Mendix tokens\nWHERE [System.owner = '[%CurrentUser%]']",
		SeeAlso: []string{"xpath.functions"},
	})

	Register(SyntaxFeature{
		Path:    "xpath.functions",
		Summary: "XPath operators and functions — contains, starts-with, not, boolean literals",
		Keywords: []string{
			"xpath functions", "contains", "starts-with",
			"not", "true", "false", "operators",
			"comparison", "boolean",
		},
		Syntax: "=, !=, <, >, <=, >=          Comparison\nand, or                      Boolean (lowercase)\nnot(expr)                    Negation\ncontains(attr, 'text')       String contains\nstarts-with(attr, 'text')    String starts-with\ntrue(), false()              Boolean literals",
		Example: "WHERE [not(IsArchived)]\nWHERE [contains(Name, 'Corp')]\nWHERE [starts-with(Code, 'PRD')]\nWHERE [State = 'Ready' and Priority > 5]",
		SeeAlso: []string{"xpath"},
	})

	// ── Java Action ───────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "java-action",
		Summary: "Java actions — custom Java code callable from microflows",
		Keywords: []string{
			"java action", "java", "call java",
			"type parameter", "exposed as", "javaaction",
		},
		Syntax: "SHOW JAVA ACTIONS [IN Module];\nDESCRIBE JAVA ACTION Module.Name;\nCREATE JAVA ACTION Module.Name(...) RETURNS Type AS $$ ... $$;\nDROP JAVA ACTION Module.Name;",
		Example: "SHOW JAVA ACTIONS;\nDESCRIBE JAVA ACTION Utils.FormatCurrency;",
		SeeAlso: []string{"java-action.create"},
	})

	Register(SyntaxFeature{
		Path:    "java-action.create",
		Summary: "Create Java actions with type parameters, EXPOSED AS, and inline code",
		Keywords: []string{
			"create java action", "type parameter", "entity parameter",
			"exposed as", "returns", "generics", "drop java action",
		},
		Syntax: "CREATE JAVA ACTION Module.Name(\n  Param: Type [NOT NULL],\n  EntityType: ENTITY <pEntity> NOT NULL,\n  Obj: pEntity\n) RETURNS ReturnType\n[EXPOSED AS 'Label' IN 'Category']\nAS $$\n// Java code\n$$;",
		Example: "CREATE JAVA ACTION Utils.FormatCurrency(\n  Amount: Decimal NOT NULL\n) RETURNS String\nEXPOSED AS 'Format Currency' IN 'Formatting'\nAS $$\nreturn String.format(\"%.2f\", Amount);\n$$;\n\n-- Generic entity validator with type parameter\nCREATE JAVA ACTION Utils.IsValid(\n  EntityType: ENTITY <pEntity> NOT NULL,\n  Obj: pEntity NOT NULL\n) RETURNS Boolean\nAS $$\nreturn Obj != null;\n$$;",
		SeeAlso: []string{"java-action"},
	})
}
