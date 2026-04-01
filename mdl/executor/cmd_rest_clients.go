// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

// showRestClients handles SHOW REST CLIENTS [IN module] command.
func (e *Executor) showRestClients(moduleName string) error {
	services, err := e.reader.ListConsumedRestServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed REST services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	type row struct {
		module        string
		qualifiedName string
		baseUrl       string
		auth          string
		ops           int
	}
	var rows []row
	modWidth := len("Module")
	qnWidth := len("QualifiedName")
	urlWidth := len("BaseURL")
	authWidth := len("Auth")
	opsWidth := len("Operations")

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}

		auth := "NONE"
		if svc.Authentication != nil {
			auth = strings.ToUpper(svc.Authentication.Scheme)
		}

		baseUrl := svc.BaseUrl
		if len(baseUrl) > 60 {
			baseUrl = baseUrl[:57] + "..."
		}

		qn := modName + "." + svc.Name
		opsStr := fmt.Sprintf("%d", len(svc.Operations))
		rows = append(rows, row{modName, qn, baseUrl, auth, len(svc.Operations)})
		if len(modName) > modWidth {
			modWidth = len(modName)
		}
		if len(qn) > qnWidth {
			qnWidth = len(qn)
		}
		if len(baseUrl) > urlWidth {
			urlWidth = len(baseUrl)
		}
		if len(auth) > authWidth {
			authWidth = len(auth)
		}
		if len(opsStr) > opsWidth {
			opsWidth = len(opsStr)
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(e.output, "No consumed REST services found.")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		modWidth, "Module", qnWidth, "QualifiedName", urlWidth, "BaseURL", authWidth, "Auth", opsWidth, "Operations")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", modWidth), strings.Repeat("-", qnWidth), strings.Repeat("-", urlWidth),
		strings.Repeat("-", authWidth), strings.Repeat("-", opsWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*d |\n",
			modWidth, r.module, qnWidth, r.qualifiedName, urlWidth, r.baseUrl, authWidth, r.auth, opsWidth, r.ops)
	}
	fmt.Fprintf(e.output, "\n(%d REST clients)\n", len(rows))

	return nil
}

// describeRestClient handles DESCRIBE REST CLIENT command.
func (e *Executor) describeRestClient(name ast.QualifiedName) error {
	services, err := e.reader.ListConsumedRestServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed REST services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, name.Module) && strings.EqualFold(svc.Name, name.Name) {
			return e.outputConsumedRestServiceMDL(svc, modName)
		}
	}

	return fmt.Errorf("consumed REST service not found: %s", name)
}

// outputConsumedRestServiceMDL outputs a consumed REST service in valid CREATE REST CLIENT MDL format.
func (e *Executor) outputConsumedRestServiceMDL(svc *model.ConsumedRestService, moduleName string) error {
	w := e.output

	// Documentation
	if svc.Documentation != "" {
		outputJavadoc(w, svc.Documentation)
	}

	// CREATE REST CLIENT
	fmt.Fprintf(w, "CREATE REST CLIENT %s.%s\n", moduleName, svc.Name)
	fmt.Fprintf(w, "BASE URL '%s'\n", svc.BaseUrl)

	// Authentication
	if svc.Authentication == nil {
		fmt.Fprintln(w, "AUTHENTICATION NONE")
	} else {
		username := formatRestAuthValue(svc.Authentication.Username)
		password := formatRestAuthValue(svc.Authentication.Password)
		fmt.Fprintf(w, "AUTHENTICATION BASIC (USERNAME = %s, PASSWORD = %s)\n", username, password)
	}

	fmt.Fprintln(w, "BEGIN")

	// Operations
	for i, op := range svc.Operations {
		if i > 0 {
			fmt.Fprintln(w)
		}
		outputRestOperation(w, op)
	}

	fmt.Fprintln(w, "END;")
	return nil
}

// outputRestOperation writes a single operation in MDL format.
func outputRestOperation(w io.Writer, op *model.RestClientOperation) {
	// Documentation
	if op.Documentation != "" {
		outputJavadocIndented(w, op.Documentation, "  ")
	}

	fmt.Fprintf(w, "  OPERATION %s\n", op.Name)
	fmt.Fprintf(w, "    METHOD %s\n", op.HttpMethod)
	fmt.Fprintf(w, "    PATH '%s'\n", op.Path)

	// Path parameters
	for _, p := range op.Parameters {
		fmt.Fprintf(w, "    PARAMETER $%s: %s\n", p.Name, p.DataType)
	}

	// Query parameters
	for _, q := range op.QueryParameters {
		fmt.Fprintf(w, "    QUERY $%s: %s\n", q.Name, q.DataType)
	}

	// Headers
	for _, h := range op.Headers {
		fmt.Fprintf(w, "    HEADER '%s' = '%s'\n", h.Name, h.Value)
	}

	// Body
	if op.BodyType != "" {
		fmt.Fprintf(w, "    BODY %s FROM %s\n", op.BodyType, op.BodyVariable)
	}

	// Timeout
	if op.Timeout > 0 {
		fmt.Fprintf(w, "    TIMEOUT %d\n", op.Timeout)
	}

	// Response
	switch op.ResponseType {
	case "NONE":
		fmt.Fprintln(w, "    RESPONSE NONE;")
	case "JSON":
		fmt.Fprintf(w, "    RESPONSE JSON AS %s;\n", op.ResponseVariable)
	case "STRING":
		fmt.Fprintf(w, "    RESPONSE STRING AS %s;\n", op.ResponseVariable)
	case "FILE":
		fmt.Fprintf(w, "    RESPONSE FILE AS %s;\n", op.ResponseVariable)
	case "STATUS":
		fmt.Fprintf(w, "    RESPONSE STATUS AS %s;\n", op.ResponseVariable)
	default:
		fmt.Fprintln(w, "    RESPONSE NONE;")
	}
}

// createRestClient handles CREATE REST CLIENT statement.
func (e *Executor) createRestClient(stmt *ast.CreateRestClientStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project (read-only mode)")
	}

	moduleName := stmt.Name.Module
	module, err := e.findModule(moduleName)
	if err != nil {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	// Check for existing service with same name
	existingServices, _ := e.reader.ListConsumedRestServices()
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, existing := range existingServices {
		existModID := h.FindModuleID(existing.ContainerID)
		existModName := h.GetModuleName(existModID)
		if strings.EqualFold(existModName, moduleName) && strings.EqualFold(existing.Name, stmt.Name.Name) {
			if stmt.CreateOrModify {
				// Delete existing and recreate
				if err := e.writer.DeleteConsumedRestService(existing.ID); err != nil {
					return fmt.Errorf("failed to delete existing REST client: %w", err)
				}
			} else {
				return fmt.Errorf("REST client already exists: %s.%s (use CREATE OR MODIFY to overwrite)", moduleName, stmt.Name.Name)
			}
		}
	}

	// Resolve folder if specified
	containerID := module.ID
	if stmt.Folder != "" {
		folderID, err := e.resolveFolder(module.ID, stmt.Folder)
		if err != nil {
			return fmt.Errorf("failed to resolve folder '%s': %w", stmt.Folder, err)
		}
		containerID = folderID
	}

	// Build the model from AST
	svc := &model.ConsumedRestService{
		ContainerID:   containerID,
		Name:          stmt.Name.Name,
		Documentation: stmt.Documentation,
		BaseUrl:       stmt.BaseUrl,
	}

	// Authentication
	if stmt.Authentication != nil {
		svc.Authentication = &model.RestAuthentication{
			Scheme:   stmt.Authentication.Scheme,
			Username: stmt.Authentication.Username,
			Password: stmt.Authentication.Password,
		}
	}

	// Operations
	for _, opDef := range stmt.Operations {
		op := buildRestClientOperation(opDef)
		svc.Operations = append(svc.Operations, op)
	}

	// Write to project
	if err := e.writer.CreateConsumedRestService(svc); err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	fmt.Fprintf(e.output, "Created REST client: %s.%s (%d operations)\n", moduleName, stmt.Name.Name, len(svc.Operations))
	return nil
}

// buildRestClientOperation converts an AST RestOperationDef to a model RestClientOperation.
func buildRestClientOperation(opDef *ast.RestOperationDef) *model.RestClientOperation {
	op := &model.RestClientOperation{
		Name:             opDef.Name,
		Documentation:    opDef.Documentation,
		HttpMethod:       opDef.Method,
		Path:             opDef.Path,
		BodyType:         opDef.BodyType,
		BodyVariable:     opDef.BodyVariable,
		ResponseType:     opDef.ResponseType,
		ResponseVariable: opDef.ResponseVariable,
		Timeout:          opDef.Timeout,
	}

	// Path parameters
	for _, p := range opDef.Parameters {
		name := strings.TrimPrefix(p.Name, "$")
		op.Parameters = append(op.Parameters, &model.RestClientParameter{
			Name:     name,
			DataType: p.DataType,
		})
	}

	// Query parameters
	for _, q := range opDef.QueryParameters {
		name := strings.TrimPrefix(q.Name, "$")
		op.QueryParameters = append(op.QueryParameters, &model.RestClientParameter{
			Name:     name,
			DataType: q.DataType,
		})
	}

	// Headers
	for _, h := range opDef.Headers {
		header := &model.RestClientHeader{
			Name: h.Name,
		}
		if h.Variable != "" {
			// Dynamic headers: store static prefix only.
			// Mendix consumed REST services don't support dynamic header values
			// in the service definition; dynamic values must be set through the
			// calling microflow.
			header.Value = h.Prefix
		} else {
			header.Value = h.Value
		}
		op.Headers = append(op.Headers, header)
	}

	return op
}

// dropRestClient handles DROP REST CLIENT statement.
func (e *Executor) dropRestClient(stmt *ast.DropRestClientStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project (read-only mode)")
	}

	services, err := e.reader.ListConsumedRestServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed REST services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		moduleName := h.GetModuleName(modID)
		if strings.EqualFold(moduleName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
			if err := e.writer.DeleteConsumedRestService(svc.ID); err != nil {
				return fmt.Errorf("failed to delete REST client: %w", err)
			}
			fmt.Fprintf(e.output, "Dropped REST client: %s.%s\n", moduleName, svc.Name)
			return nil
		}
	}

	return fmt.Errorf("REST client not found: %s", stmt.Name)
}

// formatRestAuthValue formats an authentication value for MDL output.
func formatRestAuthValue(value string) string {
	if strings.HasPrefix(value, "$") {
		return value
	}
	return "'" + value + "'"
}
