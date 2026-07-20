// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// convertMenuItemDef converts an AST NavMenuItemDef to a writer NavMenuItemSpec.
func convertMenuItemDef(def ast.NavMenuItemDef) types.NavMenuItemSpec {
	spec := types.NavMenuItemSpec{
		Caption: def.Caption,
	}
	if def.Page != nil {
		spec.Page = def.Page.String()
	}
	if def.Microflow != nil {
		spec.Microflow = def.Microflow.String()
	}
	for _, sub := range def.Items {
		spec.Items = append(spec.Items, convertMenuItemDef(sub))
	}
	return spec
}

// profileNames returns a comma-separated list of profile names for error messages.
func profileNames(nav *types.NavigationDocument) string {
	names := make([]string, len(nav.Profiles))
	for i, p := range nav.Profiles {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

// listNavigation handles SHOW NAVIGATION command.
func listNavigation(ctx *ExecContext) error {
	return listNavigationFn(ctx, ctx.Deps.Output, ctx.Deps)
}

// listNavigationMenu handles SHOW NAVIGATION MENU [profile] command.
func listNavigationMenu(ctx *ExecContext, profileName *ast.QualifiedName) error {
	return listNavigationMenuFn(ctx, ctx.Deps.Output, ctx.Deps, profileName)
}

// listNavigationHomes handles SHOW NAVIGATION HOMES command.
func listNavigationHomes(ctx *ExecContext) error {
	return listNavigationHomesFn(ctx, ctx.Deps.Output, ctx.Deps)
}

// describeNavigation handles DESCRIBE NAVIGATION [profile] command.
func describeNavigation(ctx *ExecContext, name ast.QualifiedName) error {
	return describeNavigationFn(ctx, ctx.Deps.Output, ctx.Deps, name)
}

// countMenuItems counts the total number of menu items recursively.
func countMenuItems(items []*types.NavMenuItem) int {
	count := len(items)
	for _, item := range items {
		count += countMenuItems(item.Items)
	}
	return count
}

// printMenuTree prints a menu tree with indentation to an io.Writer.
func printMenuTree(w io.Writer, items []*types.NavMenuItem, depth int) {
	indent := strings.Repeat("  ", depth+1)
	for _, item := range items {
		target := menuItemTarget(item)
		fmt.Fprintf(w, "%s%s%s\n", indent, item.Caption, target)
		if len(item.Items) > 0 {
			printMenuTree(w, item.Items, depth+1)
		}
	}
}

// menuItemTarget returns a display string for a menu item's action target.
func menuItemTarget(item *types.NavMenuItem) string {
	if item.Page != "" {
		return " -> " + item.Page
	}
	if item.Microflow != "" {
		return " -> MF:" + item.Microflow
	}
	return ""
}

// printMenuMDL prints menu items in MDL-style format.
func printMenuMDL(w io.Writer, items []*types.NavMenuItem, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, item := range items {
		if len(item.Items) > 0 {
			// Sub-menu container
			fmt.Fprintf(w, "%smenu '%s' (\n", indent, item.Caption)
			printMenuMDL(w, item.Items, depth+1)
			fmt.Fprintf(w, "%s);\n", indent)
		} else if item.Page != "" {
			fmt.Fprintf(w, "%smenu item '%s' page %s;\n", indent, item.Caption, item.Page)
		} else if item.Microflow != "" {
			fmt.Fprintf(w, "%smenu item '%s' microflow %s;\n", indent, item.Caption, item.Microflow)
		} else {
			fmt.Fprintf(w, "%smenu item '%s';\n", indent, item.Caption)
		}
	}
}

// ────────────────────────────────────────────────────────────
// Phase 3d-5g: Fn (HandlerDeps) versions of navigation functions
// ────────────────────────────────────────────────────────────

func execAlterNavigationFn(ctx context.Context, s *ast.AlterNavigationStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	nav, err := deps.NavigationReader.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	profileFound := false
	for _, p := range nav.Profiles {
		if strings.EqualFold(p.Name, s.ProfileName) {
			profileFound = true
			break
		}
	}
	if !profileFound {
		return mdlerrors.NewNotFoundMsg("navigation profile", s.ProfileName,
			fmt.Sprintf("navigation profile not found: %s (available: %s)", s.ProfileName, profileNames(nav)))
	}

	spec := types.NavigationProfileSpec{
		HasMenu: s.HasMenuBlock,
	}

	for _, hp := range s.HomePages {
		hpSpec := types.NavHomePageSpec{
			IsPage: hp.IsPage,
			Target: hp.Target.String(),
		}
		if hp.ForRole != nil {
			hpSpec.ForRole = hp.ForRole.String()
		}
		spec.HomePages = append(spec.HomePages, hpSpec)
	}

	if s.LoginPage != nil {
		spec.LoginPage = s.LoginPage.String()
	}
	if s.NotFoundPage != nil {
		spec.NotFoundPage = s.NotFoundPage.String()
	}

	for _, mi := range s.MenuItems {
		spec.MenuItems = append(spec.MenuItems, convertMenuItemDef(mi))
	}

	if err := deps.NavigationWriter.UpdateNavigationProfile(nav.ID, s.ProfileName, spec); err != nil {
		return mdlerrors.NewBackend("update navigation profile", err)
	}

	fmt.Fprintf(deps.Output, "Navigation profile '%s' updated.\n", s.ProfileName)
	return nil
}

func listNavigationFn(ctx context.Context, output io.Writer, deps *HandlerDeps) error {
	nav, err := deps.NavigationReader.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	if len(nav.Profiles) == 0 {
		fmt.Fprintln(output, "No navigation profiles found.")
		return nil
	}

	type row struct {
		name      string
		kind      string
		homePage  string
		loginPage string
		menuItems int
		roleHomes int
	}
	var rows []row

	for _, p := range nav.Profiles {
		homePage := ""
		if p.HomePage != nil {
			if p.HomePage.Page != "" {
				homePage = p.HomePage.Page
			} else if p.HomePage.Microflow != "" {
				homePage = "MF:" + p.HomePage.Microflow
			}
		}

		loginPage := p.LoginPage
		if loginPage == "" {
			loginPage = "-"
		}

		menuCount := countMenuItems(p.MenuItems)

		kind := p.Kind
		if p.IsNative {
			kind += " (native)"
		}

		rows = append(rows, row{p.Name, kind, homePage, loginPage, menuCount, len(p.RoleBasedHomePages)})
	}

	result := &TableResult{
		Columns: []string{"Profile", "Kind", "HomePage", "LoginPage", "MenuItems", "RoleHomes"},
		Summary: fmt.Sprintf("(%d navigation profiles)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.name, r.kind, r.homePage, r.loginPage, r.menuItems, r.roleHomes})
	}
	return writeResultTo(output, deps.Format, result)
}

func listNavigationMenuFn(ctx context.Context, output io.Writer, deps *HandlerDeps, profileName *ast.QualifiedName) error {
	nav, err := deps.NavigationReader.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	for _, p := range nav.Profiles {
		if profileName != nil && !strings.EqualFold(p.Name, profileName.Name) {
			continue
		}

		fmt.Fprintf(output, "-- Navigation Menu: %s (%s)\n", p.Name, p.Kind)
		if len(p.MenuItems) == 0 {
			fmt.Fprintln(output, "  (no menu items)")
		} else {
			printMenuTree(output, p.MenuItems, 0)
		}
		fmt.Fprintln(output)
	}

	return nil
}

func listNavigationHomesFn(ctx context.Context, output io.Writer, deps *HandlerDeps) error {
	nav, err := deps.NavigationReader.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	for _, p := range nav.Profiles {
		fmt.Fprintf(output, "-- Profile: %s (%s)\n", p.Name, p.Kind)

		if p.HomePage != nil {
			if p.HomePage.Page != "" {
				fmt.Fprintf(output, "  Default Home: page %s\n", p.HomePage.Page)
			} else if p.HomePage.Microflow != "" {
				fmt.Fprintf(output, "  Default Home: microflow %s\n", p.HomePage.Microflow)
			}
		} else {
			fmt.Fprintln(output, "  Default Home: (none)")
		}

		if len(p.RoleBasedHomePages) > 0 {
			fmt.Fprintln(output, "  Role-Based Homes:")
			for _, rh := range p.RoleBasedHomePages {
				target := ""
				if rh.Page != "" {
					target = "page " + rh.Page
				} else if rh.Microflow != "" {
					target = "microflow " + rh.Microflow
				}
				fmt.Fprintf(output, "    %s -> %s\n", rh.UserRole, target)
			}
		}

		fmt.Fprintln(output)
	}

	return nil
}

func describeNavigationFn(ctx context.Context, output io.Writer, deps *HandlerDeps, name ast.QualifiedName) error {
	nav, err := deps.NavigationReader.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	if name.Name == "" {
		for _, p := range nav.Profiles {
			outputNavigationProfileFn(output, p)
		}
		return nil
	}

	for _, p := range nav.Profiles {
		if strings.EqualFold(p.Name, name.Name) {
			outputNavigationProfileFn(output, p)
			return nil
		}
	}

	return mdlerrors.NewNotFound("navigation profile", name.Name)
}

func outputNavigationProfileFn(output io.Writer, p *types.NavigationProfile) {
	fmt.Fprintf(output, "-- navigation PROFILE: %s\n", p.Name)
	fmt.Fprintf(output, "--   Kind: %s\n", p.Kind)
	if p.IsNative {
		fmt.Fprintf(output, "--   Native: Yes\n")
	}

	fmt.Fprintf(output, "create or replace navigation %s\n", p.Name)

	if p.HomePage != nil {
		if p.HomePage.Page != "" {
			fmt.Fprintf(output, "  home page %s\n", p.HomePage.Page)
		} else if p.HomePage.Microflow != "" {
			fmt.Fprintf(output, "  home microflow %s\n", p.HomePage.Microflow)
		}
	}

	for _, rh := range p.RoleBasedHomePages {
		if rh.Page != "" {
			fmt.Fprintf(output, "  home page %s for %s\n", rh.Page, rh.UserRole)
		} else if rh.Microflow != "" {
			fmt.Fprintf(output, "  home microflow %s for %s\n", rh.Microflow, rh.UserRole)
		}
	}

	if p.LoginPage != "" {
		fmt.Fprintf(output, "  login page %s\n", p.LoginPage)
	}

	if p.NotFoundPage != "" {
		fmt.Fprintf(output, "  not found page %s\n", p.NotFoundPage)
	}

	if len(p.MenuItems) > 0 {
		fmt.Fprintln(output, "  menu (")
		printMenuMDL(output, p.MenuItems, 2)
		fmt.Fprintln(output, "  )")
	}

	if len(p.OfflineEntities) > 0 {
		fmt.Fprintln(output, "  -- Offline Entities (not yet modifiable):")
		for _, oe := range p.OfflineEntities {
			constraint := ""
			if oe.Constraint != "" {
				constraint = fmt.Sprintf(" where '%s'", oe.Constraint)
			}
			fmt.Fprintf(output, "  -- SYNC %s MODE %s%s;\n", oe.Entity, oe.SyncMode, constraint)
		}
	}

	fmt.Fprintln(output, ";")
	fmt.Fprintln(output)
}



func listNavigationDeps(ctx context.Context, deps *HandlerDeps) error {
	return listNavigationFuture(ctx, deps.Output, deps.NavigationReader)
}


func listNavigationMenuDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listNavigationMenuFuture(ctx, deps.Output, deps.NavigationReader, name)
}


func listNavigationHomesDeps(ctx context.Context, deps *HandlerDeps) error {
	return listNavigationHomesFuture(ctx, deps.Output, deps.NavigationReader)
}


func describeNavigationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeNavigationFuture(ctx, deps.Output, deps.NavigationReader, name)
}

// ── OData ──

// Note: listODataClientsDeps, listODataServicesDeps, listExternalEntitiesDeps,
// listExternalActionsDeps, describeODataClientDeps, describeODataServiceDeps,
// and describeExternalEntityDeps are defined in cmd_odata.go with additional
// format parameter. Do NOT redeclare here.

// ── Business Events ──


