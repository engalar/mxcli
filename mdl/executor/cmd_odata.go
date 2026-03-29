// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// outputJavadoc writes a javadoc-style comment block.
func outputJavadoc(w io.Writer, text string) {
	outputJavadocIndented(w, text, "")
}

// outputJavadocIndented writes a javadoc-style comment block with an indent prefix.
func outputJavadocIndented(w io.Writer, text string, indent string) {
	lines := strings.Split(text, "\n")
	fmt.Fprintf(w, "%s/**\n", indent)
	for _, line := range lines {
		fmt.Fprintf(w, "%s * %s\n", indent, line)
	}
	fmt.Fprintf(w, "%s */\n", indent)
}

// showODataClients handles SHOW ODATA CLIENTS [IN module] command.
func (e *Executor) showODataClients(moduleName string) error {
	services, err := e.reader.ListConsumedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	type row struct {
		module        string
		qualifiedName string
		version       string
		odataVer      string
		url           string
		validated     string
	}
	var rows []row
	modWidth := len("Module")
	qnWidth := len("QualifiedName")
	verWidth := len("Version")
	odataWidth := len("OData")
	urlWidth := len("MetadataUrl")
	valWidth := len("Validated")

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}

		validated := "No"
		if svc.Validated {
			validated = "Yes"
		}

		url := svc.MetadataUrl
		if len(url) > 60 {
			url = url[:57] + "..."
		}

		qn := modName + "." + svc.Name
		rows = append(rows, row{modName, qn, svc.Version, svc.ODataVersion, url, validated})
		if len(modName) > modWidth {
			modWidth = len(modName)
		}
		if len(qn) > qnWidth {
			qnWidth = len(qn)
		}
		if len(svc.Version) > verWidth {
			verWidth = len(svc.Version)
		}
		if len(svc.ODataVersion) > odataWidth {
			odataWidth = len(svc.ODataVersion)
		}
		if len(url) > urlWidth {
			urlWidth = len(url)
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(e.output, "No consumed OData services found.")
		return nil
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		modWidth, "Module", qnWidth, "QualifiedName", verWidth, "Version", odataWidth, "OData", urlWidth, "MetadataUrl", valWidth, "Validated")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", modWidth), strings.Repeat("-", qnWidth), strings.Repeat("-", verWidth),
		strings.Repeat("-", odataWidth), strings.Repeat("-", urlWidth), strings.Repeat("-", valWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
			modWidth, r.module, qnWidth, r.qualifiedName, verWidth, r.version, odataWidth, r.odataVer, urlWidth, r.url, valWidth, r.validated)
	}
	fmt.Fprintf(e.output, "\n(%d OData clients)\n", len(rows))

	return nil
}

// describeODataClient handles DESCRIBE ODATA CLIENT command.
func (e *Executor) describeODataClient(name ast.QualifiedName) error {
	services, err := e.reader.ListConsumedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, name.Module) && strings.EqualFold(svc.Name, name.Name) {
			folderPath := h.BuildFolderPath(svc.ContainerID)
			return e.outputConsumedODataServiceMDL(svc, modName, folderPath)
		}
	}

	return fmt.Errorf("consumed OData service not found: %s", name)
}

// outputConsumedODataServiceMDL outputs a consumed OData service in MDL format.
func (e *Executor) outputConsumedODataServiceMDL(svc *model.ConsumedODataService, moduleName string, folderPath string) error {
	// Use Description for javadoc (the user-visible API description)
	if svc.Description != "" {
		outputJavadoc(e.output, svc.Description)
	}

	fmt.Fprintf(e.output, "CREATE ODATA CLIENT %s.%s (\n", moduleName, svc.Name)

	var props []string
	if folderPath != "" {
		props = append(props, fmt.Sprintf("  Folder: '%s'", folderPath))
	}
	if svc.Version != "" {
		props = append(props, fmt.Sprintf("  Version: '%s'", svc.Version))
	}
	if svc.ODataVersion != "" {
		props = append(props, fmt.Sprintf("  ODataVersion: %s", svc.ODataVersion))
	}
	if svc.MetadataUrl != "" {
		props = append(props, fmt.Sprintf("  MetadataUrl: '%s'", svc.MetadataUrl))
	}
	if svc.TimeoutExpression != "" {
		props = append(props, fmt.Sprintf("  Timeout: %s", svc.TimeoutExpression))
	}
	if svc.ProxyType != "" && svc.ProxyType != "DefaultProxy" {
		props = append(props, fmt.Sprintf("  ProxyType: %s", svc.ProxyType))
	}

	// HTTP configuration
	if cfg := svc.HttpConfiguration; cfg != nil {
		if cfg.OverrideLocation && cfg.CustomLocation != "" {
			props = append(props, fmt.Sprintf("  ServiceUrl: %s", formatExprValue(cfg.CustomLocation)))
		}
		if cfg.UseAuthentication {
			props = append(props, "  UseAuthentication: Yes")
			if cfg.Username != "" {
				props = append(props, fmt.Sprintf("  HttpUsername: %s", formatExprValue(cfg.Username)))
			}
			if cfg.Password != "" {
				props = append(props, fmt.Sprintf("  HttpPassword: %s", formatExprValue(cfg.Password)))
			}
		}
		if cfg.ClientCertificate != "" {
			props = append(props, fmt.Sprintf("  ClientCertificate: '%s'", cfg.ClientCertificate))
		}
	}

	// Microflow references
	if svc.ConfigurationMicroflow != "" {
		props = append(props, fmt.Sprintf("  ConfigurationMicroflow: MICROFLOW %s", svc.ConfigurationMicroflow))
	}
	if svc.ErrorHandlingMicroflow != "" {
		props = append(props, fmt.Sprintf("  ErrorHandlingMicroflow: MICROFLOW %s", svc.ErrorHandlingMicroflow))
	}

	// Proxy constant references
	if svc.ProxyHost != "" {
		props = append(props, fmt.Sprintf("  ProxyHost: %s", svc.ProxyHost))
	}
	if svc.ProxyPort != "" {
		props = append(props, fmt.Sprintf("  ProxyPort: %s", svc.ProxyPort))
	}
	if svc.ProxyUsername != "" {
		props = append(props, fmt.Sprintf("  ProxyUsername: %s", svc.ProxyUsername))
	}
	if svc.ProxyPassword != "" {
		props = append(props, fmt.Sprintf("  ProxyPassword: %s", svc.ProxyPassword))
	}

	fmt.Fprintln(e.output, strings.Join(props, ",\n"))

	// Custom HTTP headers (between property block close and semicolon)
	if cfg := svc.HttpConfiguration; cfg != nil && len(cfg.HeaderEntries) > 0 {
		fmt.Fprintln(e.output, ")")
		fmt.Fprintln(e.output, "HEADERS (")
		for i, h := range cfg.HeaderEntries {
			comma := ","
			if i == len(cfg.HeaderEntries)-1 {
				comma = ""
			}
			fmt.Fprintf(e.output, "  '%s': %s%s\n", h.Key, formatExprValue(h.Value), comma)
		}
		fmt.Fprintln(e.output, ");")
	} else {
		fmt.Fprintln(e.output, ");")
	}

	fmt.Fprintln(e.output, "/")

	return nil
}

// showODataServices handles SHOW ODATA SERVICES [IN module] command.
func (e *Executor) showODataServices(moduleName string) error {
	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	type row struct {
		module        string
		qualifiedName string
		path          string
		version       string
		odataVer      string
		entitySets    string
		authTypes     string
	}
	var rows []row
	modWidth := len("Module")
	qnWidth := len("QualifiedName")
	pathWidth := len("Path")
	verWidth := len("Version")
	odataWidth := len("OData")
	esWidth := len("EntitySets")
	authWidth := len("AuthTypes")

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}

		esCount := fmt.Sprintf("%d", len(svc.EntitySets))
		authStr := strings.Join(svc.AuthenticationTypes, ", ")
		if len(authStr) > 30 {
			authStr = authStr[:27] + "..."
		}

		qn := modName + "." + svc.Name
		rows = append(rows, row{modName, qn, svc.Path, svc.Version, svc.ODataVersion, esCount, authStr})
		if len(modName) > modWidth {
			modWidth = len(modName)
		}
		if len(qn) > qnWidth {
			qnWidth = len(qn)
		}
		if len(svc.Path) > pathWidth {
			pathWidth = len(svc.Path)
		}
		if len(svc.Version) > verWidth {
			verWidth = len(svc.Version)
		}
		if len(svc.ODataVersion) > odataWidth {
			odataWidth = len(svc.ODataVersion)
		}
		if len(esCount) > esWidth {
			esWidth = len(esCount)
		}
		if len(authStr) > authWidth {
			authWidth = len(authStr)
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(e.output, "No published OData services found.")
		return nil
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		modWidth, "Module", qnWidth, "QualifiedName", pathWidth, "Path", verWidth, "Version", odataWidth, "OData", esWidth, "EntitySets", authWidth, "AuthTypes")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", modWidth), strings.Repeat("-", qnWidth), strings.Repeat("-", pathWidth),
		strings.Repeat("-", verWidth), strings.Repeat("-", odataWidth), strings.Repeat("-", esWidth),
		strings.Repeat("-", authWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
			modWidth, r.module, qnWidth, r.qualifiedName, pathWidth, r.path, verWidth, r.version,
			odataWidth, r.odataVer, esWidth, r.entitySets, authWidth, r.authTypes)
	}
	fmt.Fprintf(e.output, "\n(%d OData services)\n", len(rows))

	return nil
}

// describeODataService handles DESCRIBE ODATA SERVICE command.
func (e *Executor) describeODataService(name ast.QualifiedName) error {
	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, name.Module) && strings.EqualFold(svc.Name, name.Name) {
			folderPath := h.BuildFolderPath(svc.ContainerID)
			return e.outputPublishedODataServiceMDL(svc, modName, folderPath)
		}
	}

	return fmt.Errorf("published OData service not found: %s", name)
}

// outputPublishedODataServiceMDL outputs a published OData service in MDL format.
func (e *Executor) outputPublishedODataServiceMDL(svc *model.PublishedODataService, moduleName string, folderPath string) error {
	// Use Description for javadoc (the user-visible API description)
	if svc.Description != "" {
		outputJavadoc(e.output, svc.Description)
	}

	fmt.Fprintf(e.output, "CREATE ODATA SERVICE %s.%s (\n", moduleName, svc.Name)

	var props []string
	if folderPath != "" {
		props = append(props, fmt.Sprintf("  Folder: '%s'", folderPath))
	}
	if svc.Path != "" {
		props = append(props, fmt.Sprintf("  Path: '%s'", svc.Path))
	}
	if svc.Version != "" {
		props = append(props, fmt.Sprintf("  Version: '%s'", svc.Version))
	}
	if svc.ODataVersion != "" {
		props = append(props, fmt.Sprintf("  ODataVersion: %s", svc.ODataVersion))
	}
	if svc.Namespace != "" {
		props = append(props, fmt.Sprintf("  Namespace: '%s'", svc.Namespace))
	}
	if svc.ServiceName != "" {
		props = append(props, fmt.Sprintf("  ServiceName: '%s'", svc.ServiceName))
	}
	if svc.Summary != "" {
		props = append(props, fmt.Sprintf("  Summary: '%s'", svc.Summary))
	}
	if svc.PublishAssociations {
		props = append(props, "  PublishAssociations: Yes")
	}
	fmt.Fprintln(e.output, strings.Join(props, ",\n"))

	fmt.Fprintln(e.output, ")")

	// Authentication types
	if len(svc.AuthenticationTypes) > 0 {
		fmt.Fprintf(e.output, "AUTHENTICATION %s\n", strings.Join(svc.AuthenticationTypes, ", "))
	}
	if svc.AuthMicroflow != "" {
		fmt.Fprintf(e.output, "-- Auth Microflow: %s\n", svc.AuthMicroflow)
	}

	// Published entities block
	if len(svc.EntityTypes) > 0 || len(svc.EntitySets) > 0 {
		fmt.Fprintln(e.output, "{")

		// Build entity set lookup by exposed name and entity type name for merging
		entitySetByExposedName := make(map[string]*model.PublishedEntitySet)
		entitySetByEntityName := make(map[string]*model.PublishedEntitySet)
		for _, es := range svc.EntitySets {
			if es.ExposedName != "" {
				entitySetByExposedName[es.ExposedName] = es
			}
			if es.EntityTypeName != "" {
				entitySetByEntityName[es.EntityTypeName] = es
			}
		}

		for _, et := range svc.EntityTypes {
			// Entity-level javadoc from Summary/Description
			if et.Summary != "" || et.Description != "" {
				doc := et.Summary
				if et.Description != "" {
					if doc != "" {
						doc += "\n\n" + et.Description
					} else {
						doc = et.Description
					}
				}
				outputJavadocIndented(e.output, doc, "  ")
			}

			// Find matching entity set (try exposed name first, then entity reference)
			es := entitySetByExposedName[et.ExposedName]
			if es == nil {
				es = entitySetByEntityName[et.Entity]
			}

			// PUBLISH ENTITY line with modes
			fmt.Fprintf(e.output, "  PUBLISH ENTITY %s AS '%s'", et.Entity, et.ExposedName)
			if es != nil {
				var modeProps []string
				if es.ReadMode != "" {
					modeProps = append(modeProps, fmt.Sprintf("ReadMode: %s", es.ReadMode))
				}
				if es.InsertMode != "" {
					modeProps = append(modeProps, fmt.Sprintf("InsertMode: %s", es.InsertMode))
				}
				if es.UpdateMode != "" {
					modeProps = append(modeProps, fmt.Sprintf("UpdateMode: %s", es.UpdateMode))
				}
				if es.DeleteMode != "" {
					modeProps = append(modeProps, fmt.Sprintf("DeleteMode: %s", es.DeleteMode))
				}
				if es.UsePaging {
					modeProps = append(modeProps, "UsePaging: Yes")
					modeProps = append(modeProps, fmt.Sprintf("PageSize: %d", es.PageSize))
				}
				if len(modeProps) > 0 {
					fmt.Fprintf(e.output, " (\n    %s\n  )", strings.Join(modeProps, ",\n    "))
				}
			}
			fmt.Fprintln(e.output)

			// EXPOSE members
			if len(et.Members) > 0 {
				fmt.Fprintln(e.output, "  EXPOSE (")
				for i, m := range et.Members {
					var modifiers []string
					if m.Filterable {
						modifiers = append(modifiers, "Filterable")
					}
					if m.Sortable {
						modifiers = append(modifiers, "Sortable")
					}
					if m.IsPartOfKey {
						modifiers = append(modifiers, "Key")
					}

					line := fmt.Sprintf("    %s AS '%s'", m.Name, m.ExposedName)
					if len(modifiers) > 0 {
						line += fmt.Sprintf(" (%s)", strings.Join(modifiers, ", "))
					}
					if i < len(et.Members)-1 {
						line += ","
					}
					fmt.Fprintln(e.output, line)
				}
				fmt.Fprintln(e.output, "  );")
			}
			fmt.Fprintln(e.output)
		}

		fmt.Fprintln(e.output, "}")
	}

	// Output GRANT statements for allowed module roles
	if len(svc.AllowedModuleRoles) > 0 {
		fmt.Fprintln(e.output)
		fmt.Fprintf(e.output, "GRANT ACCESS ON ODATA SERVICE %s.%s TO %s;\n",
			moduleName, svc.Name, strings.Join(svc.AllowedModuleRoles, ", "))
	}

	fmt.Fprintln(e.output, "/")

	return nil
}

// showExternalEntities handles SHOW EXTERNAL ENTITIES [IN module] command.
func (e *Executor) showExternalEntities(moduleName string) error {
	domainModels, err := e.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("failed to list domain models: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	type row struct {
		module        string
		qualifiedName string
		service       string
		entitySet     string
		remoteName    string
		countable     string
	}
	var rows []row
	modWidth := len("Module")
	qnWidth := len("QualifiedName")
	svcWidth := len("Service")
	esWidth := len("EntitySet")
	remWidth := len("RemoteName")
	cntWidth := len("Countable")

	for _, dm := range domainModels {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}

		for _, entity := range dm.Entities {
			if entity.Source != "Rest$ODataRemoteEntitySource" {
				continue
			}

			countable := "No"
			if entity.Countable {
				countable = "Yes"
			}

			qn := modName + "." + entity.Name
			rows = append(rows, row{modName, qn, entity.RemoteServiceName, entity.RemoteEntitySet, entity.RemoteEntityName, countable})
			if len(modName) > modWidth {
				modWidth = len(modName)
			}
			if len(qn) > qnWidth {
				qnWidth = len(qn)
			}
			if len(entity.RemoteServiceName) > svcWidth {
				svcWidth = len(entity.RemoteServiceName)
			}
			if len(entity.RemoteEntitySet) > esWidth {
				esWidth = len(entity.RemoteEntitySet)
			}
			if len(entity.RemoteEntityName) > remWidth {
				remWidth = len(entity.RemoteEntityName)
			}
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(e.output, "No external entities found.")
		return nil
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		modWidth, "Module", qnWidth, "QualifiedName", svcWidth, "Service", esWidth, "EntitySet", remWidth, "RemoteName", cntWidth, "Countable")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", modWidth), strings.Repeat("-", qnWidth), strings.Repeat("-", svcWidth),
		strings.Repeat("-", esWidth), strings.Repeat("-", remWidth), strings.Repeat("-", cntWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
			modWidth, r.module, qnWidth, r.qualifiedName, svcWidth, r.service, esWidth, r.entitySet, remWidth, r.remoteName, cntWidth, r.countable)
	}
	fmt.Fprintf(e.output, "\n(%d external entities)\n", len(rows))

	return nil
}

// showExternalActions handles SHOW EXTERNAL ACTIONS [IN module] command.
// It scans all microflows and nanoflows for CallExternalAction activities
// and displays the unique actions grouped by consumed OData service.
func (e *Executor) showExternalActions(moduleName string) error {
	mfs, err := e.reader.ListMicroflows()
	if err != nil {
		return fmt.Errorf("failed to list microflows: %w", err)
	}
	nfs, err := e.reader.ListNanoflows()
	if err != nil {
		return fmt.Errorf("failed to list nanoflows: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Collect unique actions: key = service + "." + action name
	type actionInfo struct {
		service    string // Consumed OData service qualified name
		actionName string // External action name
		params     []string
		callers    []string // Microflow/nanoflow qualified names that call this action
	}
	actionMap := make(map[string]*actionInfo) // key = service.actionName

	// Helper to extract actions from a microflow object collection
	extractActions := func(oc *microflows.MicroflowObjectCollection, flowModule, flowName string) {
		if oc == nil {
			return
		}
		for _, obj := range oc.Objects {
			act, ok := obj.(*microflows.ActionActivity)
			if !ok || act.Action == nil {
				continue
			}
			cea, ok := act.Action.(*microflows.CallExternalAction)
			if !ok {
				continue
			}

			key := cea.ConsumedODataService + "." + cea.Name
			info, exists := actionMap[key]
			if !exists {
				var params []string
				for _, pm := range cea.ParameterMappings {
					params = append(params, pm.ParameterName)
				}
				info = &actionInfo{
					service:    cea.ConsumedODataService,
					actionName: cea.Name,
					params:     params,
				}
				actionMap[key] = info
			}
			caller := flowModule + "." + flowName
			// Avoid duplicate caller entries
			found := false
			for _, c := range info.callers {
				if c == caller {
					found = true
					break
				}
			}
			if !found {
				info.callers = append(info.callers, caller)
			}
			// Merge parameter names from different call sites
			if len(cea.ParameterMappings) > len(info.params) {
				info.params = nil
				for _, pm := range cea.ParameterMappings {
					info.params = append(info.params, pm.ParameterName)
				}
			}
		}
	}

	for _, mf := range mfs {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}
		extractActions(mf.ObjectCollection, modName, mf.Name)
	}
	for _, nf := range nfs {
		modID := h.FindModuleID(nf.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}
		extractActions(nf.ObjectCollection, modName, nf.Name)
	}

	if len(actionMap) == 0 {
		fmt.Fprintln(e.output, "No external actions found.")
		return nil
	}

	// Collect and sort rows
	type row struct {
		service    string
		actionName string
		params     string
		usedBy     string
	}
	var rows []row
	svcWidth := len("Service")
	actWidth := len("Action")
	paramWidth := len("Parameters")
	usedWidth := len("UsedBy")

	for _, info := range actionMap {
		params := strings.Join(info.params, ", ")
		usedBy := strings.Join(info.callers, ", ")
		rows = append(rows, row{info.service, info.actionName, params, usedBy})
		if len(info.service) > svcWidth {
			svcWidth = len(info.service)
		}
		if len(info.actionName) > actWidth {
			actWidth = len(info.actionName)
		}
		if len(params) > paramWidth {
			paramWidth = len(params)
		}
		if len(usedBy) > usedWidth {
			usedWidth = len(usedBy)
		}
	}

	// Sort by service, then action name
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].service != rows[j].service {
			return strings.ToLower(rows[i].service) < strings.ToLower(rows[j].service)
		}
		return strings.ToLower(rows[i].actionName) < strings.ToLower(rows[j].actionName)
	})

	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s |\n",
		svcWidth, "Service", actWidth, "Action", paramWidth, "Parameters", usedWidth, "UsedBy")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", svcWidth), strings.Repeat("-", actWidth),
		strings.Repeat("-", paramWidth), strings.Repeat("-", usedWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s |\n",
			svcWidth, r.service, actWidth, r.actionName, paramWidth, r.params, usedWidth, r.usedBy)
	}
	fmt.Fprintf(e.output, "\n(%d external actions)\n", len(rows))

	return nil
}

// describeExternalEntity handles DESCRIBE EXTERNAL ENTITY command.
func (e *Executor) describeExternalEntity(name ast.QualifiedName) error {
	domainModels, err := e.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("failed to list domain models: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, dm := range domainModels {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		if !strings.EqualFold(modName, name.Module) {
			continue
		}

		for _, entity := range dm.Entities {
			if !strings.EqualFold(entity.Name, name.Name) {
				continue
			}

			if entity.Source != "Rest$ODataRemoteEntitySource" {
				return fmt.Errorf("%s.%s is not an external entity (source: %s)", modName, entity.Name, entity.Source)
			}

			return e.outputExternalEntityMDL(entity, modName)
		}
	}

	return fmt.Errorf("external entity not found: %s", name)
}

// outputExternalEntityMDL outputs an external entity in MDL format.
func (e *Executor) outputExternalEntityMDL(entity *domainmodel.Entity, moduleName string) error {
	if entity.Documentation != "" {
		outputJavadoc(e.output, entity.Documentation)
	}

	fmt.Fprintf(e.output, "CREATE EXTERNAL ENTITY %s.%s\n", moduleName, entity.Name)
	fmt.Fprintf(e.output, "FROM ODATA CLIENT %s\n", entity.RemoteServiceName)
	fmt.Fprintln(e.output, "(")

	var props []string
	if entity.RemoteEntitySet != "" {
		props = append(props, fmt.Sprintf("  EntitySet: '%s'", entity.RemoteEntitySet))
	}
	if entity.RemoteEntityName != "" {
		props = append(props, fmt.Sprintf("  RemoteName: '%s'", entity.RemoteEntityName))
	}
	boolStr := func(b bool) string {
		if b {
			return "Yes"
		}
		return "No"
	}
	props = append(props, fmt.Sprintf("  Countable: %s", boolStr(entity.Countable)))
	props = append(props, fmt.Sprintf("  Creatable: %s", boolStr(entity.Creatable)))
	props = append(props, fmt.Sprintf("  Deletable: %s", boolStr(entity.Deletable)))
	props = append(props, fmt.Sprintf("  Updatable: %s", boolStr(entity.Updatable)))
	fmt.Fprintln(e.output, strings.Join(props, ",\n"))

	fmt.Fprintln(e.output, ")")

	// Output attributes
	if len(entity.Attributes) > 0 {
		fmt.Fprintln(e.output, "(")
		for i, attr := range entity.Attributes {
			typeName := "Unknown"
			if attr.Type != nil {
				typeName = attr.Type.GetTypeName()
			}
			comma := ","
			if i == len(entity.Attributes)-1 {
				comma = ""
			}
			fmt.Fprintf(e.output, "  %s: %s%s\n", attr.Name, typeName, comma)
		}
		fmt.Fprintln(e.output, ");")
	}

	fmt.Fprintln(e.output, "/")

	return nil
}

// ============================================================================
// CREATE EXTERNAL ENTITY
// ============================================================================

// execCreateExternalEntity handles CREATE [OR MODIFY] EXTERNAL ENTITY statements.
func (e *Executor) execCreateExternalEntity(s *ast.CreateExternalEntityStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	if s.Name.Module == "" {
		return fmt.Errorf("module name required: use CREATE EXTERNAL ENTITY Module.Name FROM ODATA CLIENT ...")
	}

	// Find module
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Get domain model
	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Check if entity already exists
	var existingEntity *domainmodel.Entity
	for _, entity := range dm.Entities {
		if entity.Name == s.Name.Name {
			existingEntity = entity
			break
		}
	}

	if existingEntity != nil && !s.CreateOrModify {
		return fmt.Errorf("entity already exists: %s.%s (use CREATE OR MODIFY to update)", s.Name.Module, s.Name.Name)
	}

	// Build attributes
	var attrs []*domainmodel.Attribute
	for _, a := range s.Attributes {
		attr := &domainmodel.Attribute{
			Name: a.Name,
			Type: convertDataType(a.Type),
		}
		attr.ID = model.ID(mpr.GenerateID())
		attrs = append(attrs, attr)
	}

	// Service reference as qualified name
	serviceRef := s.ServiceRef.String()

	if existingEntity != nil {
		// Update existing entity
		existingEntity.Source = "Rest$ODataRemoteEntitySource"
		existingEntity.RemoteServiceName = serviceRef
		existingEntity.RemoteEntitySet = s.EntitySet
		existingEntity.RemoteEntityName = s.RemoteName
		existingEntity.Countable = s.Countable
		existingEntity.Creatable = s.Creatable
		existingEntity.Deletable = s.Deletable
		existingEntity.Updatable = s.Updatable
		if len(attrs) > 0 {
			existingEntity.Attributes = attrs
		}
		if s.Documentation != "" {
			existingEntity.Documentation = s.Documentation
		}
		if err := e.writer.UpdateEntity(dm.ID, existingEntity); err != nil {
			return fmt.Errorf("failed to update external entity: %w", err)
		}
		fmt.Fprintf(e.output, "Modified external entity: %s.%s\n", s.Name.Module, s.Name.Name)
		return nil
	}

	// Auto-position based on existing entities
	location := model.Point{X: 100 + len(dm.Entities)*150, Y: 100}

	newEntity := &domainmodel.Entity{
		Name:              s.Name.Name,
		Documentation:     s.Documentation,
		Persistable:       false, // External entities are not persistable
		Location:          location,
		Attributes:        attrs,
		Source:            "Rest$ODataRemoteEntitySource",
		RemoteServiceName: serviceRef,
		RemoteEntitySet:   s.EntitySet,
		RemoteEntityName:  s.RemoteName,
		Countable:         s.Countable,
		Creatable:         s.Creatable,
		Deletable:         s.Deletable,
		Updatable:         s.Updatable,
	}
	newEntity.ID = model.ID(mpr.GenerateID())

	if err := e.writer.CreateEntity(dm.ID, newEntity); err != nil {
		return fmt.Errorf("failed to create external entity: %w", err)
	}
	fmt.Fprintf(e.output, "Created external entity: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// ============================================================================
// OData Write Handlers (CREATE / ALTER / DROP)
// ============================================================================

// createODataClient handles CREATE ODATA CLIENT command.
func (e *Executor) createODataClient(stmt *ast.CreateODataClientStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	if stmt.Name.Module == "" {
		return fmt.Errorf("module name required: use CREATE ODATA CLIENT Module.Name (...)")
	}

	module, err := e.findModule(stmt.Name.Module)
	if err != nil {
		return err
	}

	// Check if client already exists
	services, err := e.reader.ListConsumedODataServices()
	if err == nil {
		h, _ := e.getHierarchy()
		for _, svc := range services {
			modID := h.FindModuleID(svc.ContainerID)
			modName := h.GetModuleName(modID)
			if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
				if stmt.CreateOrModify {
					svc.Documentation = stmt.Documentation
					if stmt.Version != "" {
						svc.Version = stmt.Version
					}
					if stmt.ODataVersion != "" {
						svc.ODataVersion = stmt.ODataVersion
					}
					if stmt.MetadataUrl != "" {
						svc.MetadataUrl = stmt.MetadataUrl
					}
					if stmt.TimeoutExpression != "" {
						svc.TimeoutExpression = stmt.TimeoutExpression
					}
					if stmt.ProxyType != "" {
						svc.ProxyType = stmt.ProxyType
					}
					if stmt.Description != "" {
						svc.Description = stmt.Description
					}
					if stmt.ConfigurationMicroflow != "" {
						svc.ConfigurationMicroflow = extractMicroflowRef(stmt.ConfigurationMicroflow)
					}
					if stmt.ErrorHandlingMicroflow != "" {
						svc.ErrorHandlingMicroflow = extractMicroflowRef(stmt.ErrorHandlingMicroflow)
					}
					if stmt.ProxyHost != "" {
						svc.ProxyHost = stmt.ProxyHost
					}
					if stmt.ProxyPort != "" {
						svc.ProxyPort = stmt.ProxyPort
					}
					if stmt.ProxyUsername != "" {
						svc.ProxyUsername = stmt.ProxyUsername
					}
					if stmt.ProxyPassword != "" {
						svc.ProxyPassword = stmt.ProxyPassword
					}
					// Update HTTP configuration
					if stmt.ServiceUrl != "" || stmt.UseAuthentication || stmt.HttpUsername != "" ||
						stmt.HttpPassword != "" || stmt.ClientCertificate != "" || len(stmt.Headers) > 0 {
						if svc.HttpConfiguration == nil {
							svc.HttpConfiguration = &model.HttpConfiguration{}
						}
						if stmt.ServiceUrl != "" {
							svc.HttpConfiguration.OverrideLocation = true
							svc.HttpConfiguration.CustomLocation = stmt.ServiceUrl
						}
						svc.HttpConfiguration.UseAuthentication = stmt.UseAuthentication
						if stmt.HttpUsername != "" {
							svc.HttpConfiguration.Username = stmt.HttpUsername
						}
						if stmt.HttpPassword != "" {
							svc.HttpConfiguration.Password = stmt.HttpPassword
						}
						if stmt.ClientCertificate != "" {
							svc.HttpConfiguration.ClientCertificate = stmt.ClientCertificate
						}
						if len(stmt.Headers) > 0 {
							svc.HttpConfiguration.HeaderEntries = nil
							for _, h := range stmt.Headers {
								svc.HttpConfiguration.HeaderEntries = append(svc.HttpConfiguration.HeaderEntries, &model.HttpHeaderEntry{
									Key:   h.Key,
									Value: h.Value,
								})
							}
						}
					}
					if err := e.writer.UpdateConsumedODataService(svc); err != nil {
						return fmt.Errorf("failed to update OData client: %w", err)
					}
					e.invalidateHierarchy()
					fmt.Fprintf(e.output, "Modified OData client: %s.%s\n", modName, svc.Name)
					return nil
				}
				return fmt.Errorf("OData client already exists: %s.%s (use CREATE OR MODIFY to update)", modName, svc.Name)
			}
		}
	}

	// Resolve folder if specified
	containerID := module.ID
	if stmt.Folder != "" {
		folderID, err := e.resolveFolder(module.ID, stmt.Folder)
		if err != nil {
			return fmt.Errorf("failed to resolve folder %s: %w", stmt.Folder, err)
		}
		containerID = folderID
	}

	newSvc := &model.ConsumedODataService{
		ContainerID:            containerID,
		Name:                   stmt.Name.Name,
		ServiceName:            stmt.Name.Name, // Default ServiceName to document name (CE0339)
		Documentation:          stmt.Documentation,
		Version:                stmt.Version,
		ODataVersion:           stmt.ODataVersion,
		MetadataUrl:            stmt.MetadataUrl,
		TimeoutExpression:      stmt.TimeoutExpression,
		ProxyType:              stmt.ProxyType,
		Description:            stmt.Description,
		ConfigurationMicroflow: extractMicroflowRef(stmt.ConfigurationMicroflow),
		ErrorHandlingMicroflow: extractMicroflowRef(stmt.ErrorHandlingMicroflow),
		ProxyHost:              stmt.ProxyHost,
		ProxyPort:              stmt.ProxyPort,
		ProxyUsername:          stmt.ProxyUsername,
		ProxyPassword:          stmt.ProxyPassword,
	}

	// Build HTTP configuration if any HTTP-level properties are set
	if stmt.ServiceUrl != "" || stmt.UseAuthentication || stmt.HttpUsername != "" ||
		stmt.HttpPassword != "" || stmt.ClientCertificate != "" || len(stmt.Headers) > 0 {
		cfg := &model.HttpConfiguration{
			UseAuthentication: stmt.UseAuthentication,
			Username:          stmt.HttpUsername,
			Password:          stmt.HttpPassword,
			ClientCertificate: stmt.ClientCertificate,
		}
		if stmt.ServiceUrl != "" {
			cfg.OverrideLocation = true
			cfg.CustomLocation = stmt.ServiceUrl
		}
		for _, h := range stmt.Headers {
			cfg.HeaderEntries = append(cfg.HeaderEntries, &model.HttpHeaderEntry{
				Key:   h.Key,
				Value: h.Value,
			})
		}
		newSvc.HttpConfiguration = cfg
	}

	// Fetch and cache $metadata from the service URL
	if newSvc.MetadataUrl != "" {
		metadata, hash, err := fetchODataMetadata(newSvc.MetadataUrl)
		if err != nil {
			fmt.Fprintf(e.output, "Warning: could not fetch $metadata: %v\n", err)
		} else if metadata != "" {
			newSvc.Metadata = metadata
			newSvc.MetadataHash = hash
			newSvc.Validated = true
		}
	}

	if err := e.writer.CreateConsumedODataService(newSvc); err != nil {
		return fmt.Errorf("failed to create OData client: %w", err)
	}
	e.invalidateHierarchy()
	fmt.Fprintf(e.output, "Created OData client: %s.%s\n", stmt.Name.Module, stmt.Name.Name)
	if newSvc.Metadata != "" {
		// Parse to show summary
		if doc, err := mpr.ParseEdmx(newSvc.Metadata); err == nil {
			entityCount := 0
			actionCount := 0
			for _, s := range doc.Schemas {
				entityCount += len(s.EntityTypes)
			}
			actionCount = len(doc.Actions)
			fmt.Fprintf(e.output, "  Cached $metadata: %d entity types, %d actions\n", entityCount, actionCount)
		}
	}
	return nil
}

// alterODataClient handles ALTER ODATA CLIENT command.
func (e *Executor) alterODataClient(stmt *ast.AlterODataClientStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	services, err := e.reader.ListConsumedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
			for key, val := range stmt.Changes {
				strVal := fmt.Sprintf("%v", val)
				switch strings.ToLower(key) {
				case "version":
					svc.Version = strVal
				case "odataversion":
					svc.ODataVersion = strVal
				case "metadataurl":
					svc.MetadataUrl = strVal
				case "timeout":
					svc.TimeoutExpression = strVal
				case "proxytype":
					svc.ProxyType = strVal
				case "description":
					svc.Description = strVal
				case "serviceurl":
					if svc.HttpConfiguration == nil {
						svc.HttpConfiguration = &model.HttpConfiguration{}
					}
					svc.HttpConfiguration.OverrideLocation = true
					svc.HttpConfiguration.CustomLocation = strVal
				case "useauthentication":
					if svc.HttpConfiguration == nil {
						svc.HttpConfiguration = &model.HttpConfiguration{}
					}
					svc.HttpConfiguration.UseAuthentication = strings.EqualFold(strVal, "true") || strings.EqualFold(strVal, "yes")
				case "httpusername":
					if svc.HttpConfiguration == nil {
						svc.HttpConfiguration = &model.HttpConfiguration{}
					}
					svc.HttpConfiguration.Username = strVal
				case "httppassword":
					if svc.HttpConfiguration == nil {
						svc.HttpConfiguration = &model.HttpConfiguration{}
					}
					svc.HttpConfiguration.Password = strVal
				case "clientcertificate":
					if svc.HttpConfiguration == nil {
						svc.HttpConfiguration = &model.HttpConfiguration{}
					}
					svc.HttpConfiguration.ClientCertificate = strVal
				case "configurationmicroflow":
					svc.ConfigurationMicroflow = extractMicroflowRef(strVal)
				case "errorhandlingmicroflow":
					svc.ErrorHandlingMicroflow = extractMicroflowRef(strVal)
				case "proxyhost":
					svc.ProxyHost = strVal
				case "proxyport":
					svc.ProxyPort = strVal
				case "proxyusername":
					svc.ProxyUsername = strVal
				case "proxypassword":
					svc.ProxyPassword = strVal
				default:
					return fmt.Errorf("unknown OData client property: %s", key)
				}
			}
			if err := e.writer.UpdateConsumedODataService(svc); err != nil {
				return fmt.Errorf("failed to alter OData client: %w", err)
			}
			e.invalidateHierarchy()
			fmt.Fprintf(e.output, "Altered OData client: %s.%s\n", modName, svc.Name)
			return nil
		}
	}

	return fmt.Errorf("OData client not found: %s", stmt.Name)
}

// dropODataClient handles DROP ODATA CLIENT command.
func (e *Executor) dropODataClient(stmt *ast.DropODataClientStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	services, err := e.reader.ListConsumedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list consumed OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
			if err := e.writer.DeleteConsumedODataService(svc.ID); err != nil {
				return fmt.Errorf("failed to drop OData client: %w", err)
			}
			e.invalidateHierarchy()
			fmt.Fprintf(e.output, "Dropped OData client: %s.%s\n", modName, svc.Name)
			return nil
		}
	}

	return fmt.Errorf("OData client not found: %s", stmt.Name)
}

// createODataService handles CREATE ODATA SERVICE command.
func (e *Executor) createODataService(stmt *ast.CreateODataServiceStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	if stmt.Name.Module == "" {
		return fmt.Errorf("module name required: use CREATE ODATA SERVICE Module.Name (...)")
	}

	module, err := e.findModule(stmt.Name.Module)
	if err != nil {
		return err
	}

	// Check if service already exists
	services, err := e.reader.ListPublishedODataServices()
	if err == nil {
		h, _ := e.getHierarchy()
		for _, svc := range services {
			modID := h.FindModuleID(svc.ContainerID)
			modName := h.GetModuleName(modID)
			if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
				if stmt.CreateOrModify {
					svc.Documentation = stmt.Documentation
					if stmt.Path != "" {
						svc.Path = stmt.Path
					}
					if stmt.Version != "" {
						svc.Version = stmt.Version
					}
					if stmt.ODataVersion != "" {
						svc.ODataVersion = stmt.ODataVersion
					}
					if stmt.Namespace != "" {
						svc.Namespace = stmt.Namespace
					}
					if stmt.ServiceName != "" {
						svc.ServiceName = stmt.ServiceName
					}
					if stmt.Summary != "" {
						svc.Summary = stmt.Summary
					}
					if stmt.Description != "" {
						svc.Description = stmt.Description
					}
					svc.PublishAssociations = stmt.PublishAssociations
					if len(stmt.AuthenticationTypes) > 0 {
						svc.AuthenticationTypes = stmt.AuthenticationTypes
					}
					if err := e.writer.UpdatePublishedODataService(svc); err != nil {
						return fmt.Errorf("failed to update OData service: %w", err)
					}
					e.invalidateHierarchy()
					fmt.Fprintf(e.output, "Modified OData service: %s.%s\n", modName, svc.Name)
					return nil
				}
				return fmt.Errorf("OData service already exists: %s.%s (use CREATE OR MODIFY to update)", modName, svc.Name)
			}
		}
	}

	// Resolve folder if specified
	containerID := module.ID
	if stmt.Folder != "" {
		folderID, err := e.resolveFolder(module.ID, stmt.Folder)
		if err != nil {
			return fmt.Errorf("failed to resolve folder %s: %w", stmt.Folder, err)
		}
		containerID = folderID
	}

	newSvc := &model.PublishedODataService{
		ContainerID:         containerID,
		Name:                stmt.Name.Name,
		Documentation:       stmt.Documentation,
		Path:                stmt.Path,
		Version:             stmt.Version,
		ODataVersion:        stmt.ODataVersion,
		Namespace:           stmt.Namespace,
		ServiceName:         stmt.ServiceName,
		Summary:             stmt.Summary,
		Description:         stmt.Description,
		PublishAssociations: stmt.PublishAssociations,
		AuthenticationTypes: stmt.AuthenticationTypes,
	}

	// Map AST entity definitions to model entity types and entity sets
	for _, entityDef := range stmt.Entities {
		entityType, entitySet := astEntityDefToModel(entityDef)
		newSvc.EntityTypes = append(newSvc.EntityTypes, entityType)
		newSvc.EntitySets = append(newSvc.EntitySets, entitySet)
	}

	if err := e.writer.CreatePublishedODataService(newSvc); err != nil {
		return fmt.Errorf("failed to create OData service: %w", err)
	}
	e.invalidateHierarchy()
	fmt.Fprintf(e.output, "Created OData service: %s.%s\n", stmt.Name.Module, stmt.Name.Name)
	return nil
}

// alterODataService handles ALTER ODATA SERVICE command.
func (e *Executor) alterODataService(stmt *ast.AlterODataServiceStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
			for key, val := range stmt.Changes {
				strVal := fmt.Sprintf("%v", val)
				switch strings.ToLower(key) {
				case "path":
					svc.Path = strVal
				case "version":
					svc.Version = strVal
				case "odataversion":
					svc.ODataVersion = strVal
				case "namespace":
					svc.Namespace = strVal
				case "servicename":
					svc.ServiceName = strVal
				case "summary":
					svc.Summary = strVal
				case "description":
					svc.Description = strVal
				case "publishassociations":
					svc.PublishAssociations = strings.EqualFold(strVal, "true") || strings.EqualFold(strVal, "yes")
				default:
					return fmt.Errorf("unknown OData service property: %s", key)
				}
			}
			if err := e.writer.UpdatePublishedODataService(svc); err != nil {
				return fmt.Errorf("failed to alter OData service: %w", err)
			}
			e.invalidateHierarchy()
			fmt.Fprintf(e.output, "Altered OData service: %s.%s\n", modName, svc.Name)
			return nil
		}
	}

	return fmt.Errorf("OData service not found: %s", stmt.Name)
}

// dropODataService handles DROP ODATA SERVICE command.
func (e *Executor) dropODataService(stmt *ast.DropODataServiceStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected in write mode")
	}

	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, stmt.Name.Module) && strings.EqualFold(svc.Name, stmt.Name.Name) {
			if err := e.writer.DeletePublishedODataService(svc.ID); err != nil {
				return fmt.Errorf("failed to drop OData service: %w", err)
			}
			e.invalidateHierarchy()
			fmt.Fprintf(e.output, "Dropped OData service: %s.%s\n", modName, svc.Name)
			return nil
		}
	}

	return fmt.Errorf("OData service not found: %s", stmt.Name)
}

// formatExprValue formats a Mendix expression value for MDL output.
// If the value is already a quoted string literal (starts/ends with '), it's output as-is.
// Otherwise, it's wrapped in single quotes for round-trip compatibility.
func formatExprValue(val string) string {
	if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
		return val // Already a quoted Mendix expression string literal
	}
	// Wrap in quotes, escaping internal single quotes
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}

// extractMicroflowRef strips "MICROFLOW " prefix from a microflow reference string.
// Both "MICROFLOW Module.Name" and "Module.Name" formats are accepted.
func extractMicroflowRef(ref string) string {
	return strings.TrimPrefix(ref, "MICROFLOW ")
}

// astEntityDefToModel converts an AST PublishedEntityDef to model PublishedEntityType
// and PublishedEntitySet. Each PUBLISH ENTITY block maps to both a type (schema) and
// a set (runtime endpoint with CRUD modes).
func astEntityDefToModel(def *ast.PublishedEntityDef) (*model.PublishedEntityType, *model.PublishedEntitySet) {
	exposedName := def.ExposedName
	if exposedName == "" {
		// Default exposed name from the entity name
		exposedName = def.Entity.Name
	}

	entityType := &model.PublishedEntityType{
		Entity:      def.Entity.String(),
		ExposedName: exposedName,
	}

	// Map AST members to model members
	for _, m := range def.Members {
		member := &model.PublishedMember{
			Kind:        "attribute", // Default kind — cannot be distinguished from MDL syntax alone
			Name:        m.Name,
			ExposedName: m.ExposedName,
			Filterable:  m.Filterable,
			Sortable:    m.Sortable,
			IsPartOfKey: m.IsPartOfKey,
		}
		if member.ExposedName == "" {
			member.ExposedName = member.Name
		}
		entityType.Members = append(entityType.Members, member)
	}

	entitySet := &model.PublishedEntitySet{
		ExposedName:    exposedName,
		EntityTypeName: def.Entity.String(),
		ReadMode:       def.ReadMode,
		InsertMode:     def.InsertMode,
		UpdateMode:     def.UpdateMode,
		DeleteMode:     def.DeleteMode,
		UsePaging:      def.UsePaging,
		PageSize:       def.PageSize,
	}

	return entityType, entitySet
}

// fetchODataMetadata downloads the $metadata document from the service URL.
// Returns the metadata XML and its SHA-256 hash, or empty strings if the fetch fails.
func fetchODataMetadata(metadataUrl string) (metadata string, hash string, err error) {
	if metadataUrl == "" {
		return "", "", nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(metadataUrl)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch $metadata from %s: %w", metadataUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("$metadata fetch returned HTTP %d from %s", resp.StatusCode, metadataUrl)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read $metadata response: %w", err)
	}

	metadata = string(body)
	h := sha256.Sum256(body)
	hash = fmt.Sprintf("%x", h)
	return metadata, hash, nil
}
