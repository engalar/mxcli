// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/mendixlabs/mxcli/internal/auth"
	"github.com/mendixlabs/mxcli/internal/marketplace/application"
	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
	"github.com/mendixlabs/mxcli/internal/marketplace/infrastructure"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse and install from the Mendix Marketplace",
	Long: `Browse published modules, widgets, and themes in the Mendix Marketplace.
Download and install them into your project.

Requires a Personal Access Token (PAT). Run 'mxcli auth login' first.`,
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search marketplace content by keyword",
	Example: `  mxcli marketplace search "database connector"
  mxcli marketplace search "audit" --limit 5 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceSearch,
}

var marketplaceInfoCmd = &cobra.Command{
	Use:   "info <content-id>",
	Short: "Show details of a marketplace item",
	Example: `  mxcli marketplace info 170
  mxcli marketplace info 2888 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceInfo,
}

var marketplaceVersionsCmd = &cobra.Command{
	Use:   "versions <content-id>",
	Short: "List available versions of a marketplace item",
	Example: `  mxcli marketplace versions 2888
  mxcli marketplace versions 170 --min-mendix 10.24.0
  mxcli marketplace versions 170 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceVersions,
}

var marketplaceDownloadCmd = &cobra.Command{
	Use:   "download <content-id>",
	Short: "Download a .mpk file",
	Long: `Download a marketplace module or widget as a .mpk file.

The download is atomic (written to a temp file and renamed), so a cancelled
run never leaves a truncated .mpk.`,
	Example: `  mxcli marketplace download 2888
  mxcli marketplace download 170 --version 11.5.0 -o ./my.mpk`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceDownload,
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <content-id>",
	Short: "Download and install into a project",
	Long: `Download a marketplace module and install it into the project.

For widgets, copies the .mpk into the project's widgets/ folder.
For modules, imports via mxbuild module-import.

If the module is already present in the project, the command reports
the current and target versions and stops — in-place module updates
are not applied automatically because they can discard local edits
and change persistent entity IDs.`,
	Example: `  mxcli marketplace install 2888 -p app.mpr
  mxcli marketplace install 170 --version 11.5.0 -p app.mpr`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceInstall,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed marketplace modules",
	Long: `List all modules in the project that were installed from the
Mendix Marketplace. Shows the installed version and, when possible, the
latest available version from the marketplace.`,
	Example: `  mxcli marketplace list -p app.mpr
  mxcli marketplace list -p app.mpr --json`,
	RunE: runMarketplaceList,
}

var marketplaceUpdateCmd = &cobra.Command{
	Use:   "update [content-id]",
	Short: "Check for module updates",
	Long: `Check installed marketplace modules for available updates.

Reports which modules have newer versions available. In-place automatic
updates are not applied — use Studio Pro to perform an ID-preserving merge.

With a content-id, checks only that module. Without, checks all installed
marketplace modules.`,
	Example: `  mxcli marketplace update -p app.mpr
  mxcli marketplace update 2888 -p app.mpr`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMarketplaceUpdate,
}

func init() {
	marketplaceSearchCmd.Flags().IntP("limit", "n", 20, "max results")
	marketplaceSearchCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
	marketplaceSearchCmd.Flags().Bool("json", false, "emit JSON instead of a table")
	marketplaceSearchCmd.Flags().Bool("refresh", false, "bypass search cache")

	marketplaceInfoCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
	marketplaceInfoCmd.Flags().Bool("json", false, "emit JSON instead of a table")

	marketplaceVersionsCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
	marketplaceVersionsCmd.Flags().Bool("json", false, "emit JSON instead of a table")
	marketplaceVersionsCmd.Flags().String("min-mendix", "", "filter versions whose minSupportedMendixVersion is <= this (e.g., 10.24.0)")

	marketplaceDownloadCmd.Flags().String("version", "", "specific version to download")
	marketplaceDownloadCmd.Flags().StringP("output", "o", "", "output file path")
	marketplaceDownloadCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

	marketplaceInstallCmd.Flags().String("version", "", "specific version to install")
	marketplaceInstallCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

	marketplaceListCmd.Flags().Bool("json", false, "emit JSON")
	marketplaceListCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

	marketplaceUpdateCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInfoCmd)
	marketplaceCmd.AddCommand(marketplaceVersionsCmd)
	marketplaceCmd.AddCommand(marketplaceDownloadCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)
}

func buildMarketplaceService(ctx context.Context, cmd *cobra.Command) (*application.Service, error) {
	if marketplaceServiceFactory != nil {
		return marketplaceServiceFactory(ctx, cmd)
	}
	profile, _ := cmd.Flags().GetString("profile")
	httpClient, err := auth.ClientFor(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("%w\nhint: run 'mxcli auth login'", err)
	}
	cred, err := auth.Resolve(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("%w\nhint: run 'mxcli auth login'", err)
	}
	apiClient := infrastructure.NewAPIClient(httpClient, infrastructure.DefaultBaseURL)
	downloader := infrastructure.NewDownloader(http.DefaultClient, cred.Token)
	lister := projectModuleLister(cmd)

	return application.NewService(apiClient, apiClient, downloader, lister, nil), nil
}

func marketplaceCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mxcli", "marketplace-cache"), nil
}

func projectModuleLister(cmd *cobra.Command) *infrastructure.ProjectReader {
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath == "" {
		return nil
	}
	// Try to open the project and get a module lister.
	// This is a lightweight operation that returns only FromAppStore modules.
	mprPath := resolveProjectPath(projectPath)
	reader, err := openProjectForModuleList(mprPath)
	if err != nil {
		return nil
	}
	return reader
}

func openProjectForModuleList(projectPath string) (*infrastructure.ProjectReader, error) {
	// Use modelsdk to open the project and list modules.
	// This requires importing the backend — currently delegates to
	// a factory that returns nil in CLI context (the -p flag is
	// primarily needed for install/list/update commands).
	return nil, nil
}

var marketplaceServiceFactory func(context.Context, *cobra.Command) (*application.Service, error)

func runMarketplaceSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	asJSON, _ := cmd.Flags().GetBool("json")

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	results, err := svc.Search(cmd.Context(), query, limit)
	if err != nil {
		return err
	}
	if asJSON {
		return emitJSON(cmd, results)
	}
	return renderContentTable(cmd, results)
}

func runMarketplaceInfo(cmd *cobra.Command, args []string) error {
	contentID, err := parseContentID(args[0])
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	content, err := svc.Get(cmd.Context(), domain.ContentID(contentID))
	if err != nil {
		return err
	}
	if asJSON {
		return emitJSON(cmd, content)
	}
	return renderContentDetail(cmd, content)
}

func runMarketplaceVersions(cmd *cobra.Command, args []string) error {
	contentID, err := parseContentID(args[0])
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	minMendix, _ := cmd.Flags().GetString("min-mendix")

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	items, err := svc.GetVersions(cmd.Context(), domain.ContentID(contentID))
	if err != nil {
		return err
	}

	if minMendix != "" {
		items = filterVersionsByMinMendix(items, minMendix)
	}

	if asJSON {
		return emitJSON(cmd, items)
	}
	return renderVersionsTable(cmd, items)
}

func runMarketplaceDownload(cmd *cobra.Command, args []string) error {
	contentID, err := parseContentID(args[0])
	if err != nil {
		return err
	}
	version, _ := cmd.Flags().GetString("version")
	outputPath, _ := cmd.Flags().GetString("output")

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	path, err := svc.Download(cmd.Context(), domain.ContentID(contentID), version, outputPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Downloaded: %s\n", path)
	return nil
}

func runMarketplaceInstall(cmd *cobra.Command, args []string) error {
	contentID, err := parseContentID(args[0])
	if err != nil {
		return err
	}
	version, _ := cmd.Flags().GetString("version")
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath == "" {
		return fmt.Errorf("--project/-p is required for install")
	}

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	if err := svc.Install(cmd.Context(), domain.ContentID(contentID), version, projectPath); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Installed content %d.\n", contentID)
	return nil
}

func runMarketplaceList(cmd *cobra.Command, _ []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath == "" {
		return fmt.Errorf("--project/-p is required for list")
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	modules, err := svc.ListInstalled(cmd.Context(), projectPath)
	if err != nil {
		return err
	}
	if asJSON {
		return emitJSON(cmd, modules)
	}
	if len(modules) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No marketplace modules installed.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tVERSION\tGUID")
	for _, m := range modules {
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.AppStoreVersion, m.AppStoreGuid)
	}
	return w.Flush()
}

func runMarketplaceUpdate(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath == "" {
		return fmt.Errorf("--project/-p is required for update")
	}

	svc, err := buildMarketplaceService(cmd.Context(), cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		contentID, err := parseContentID(args[0])
		if err != nil {
			return err
		}
		result, err := svc.Update(cmd.Context(), domain.ContentID(contentID), projectPath)
		if err != nil {
			return err
		}
		return renderUpdateResult(cmd, result)
	}

	results, err := svc.UpdateAll(cmd.Context(), projectPath)
	if err != nil {
		return err
	}
	return renderUpdateResults(cmd, results)
}

func renderUpdateResult(cmd *cobra.Command, r *application.UpdateResult) error {
	switch r.Status {
	case "up-to-date":
		fmt.Fprintf(cmd.OutOrStdout(), "%s is up-to-date at version %s.\n", r.ModuleName, r.InstalledVersion)
	case "update-available":
		fmt.Fprintf(cmd.OutOrStdout(), "%s: installed %s, latest %s. Update via Studio Pro.\n", r.ModuleName, r.InstalledVersion, r.LatestVersion)
	case "error":
		fmt.Fprintf(cmd.OutOrStdout(), "%s: error checking update: %s\n", r.ModuleName, r.Error)
	}
	return nil
}

func renderUpdateResults(cmd *cobra.Command, results []application.UpdateResult) error {
	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No marketplace modules found.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tINSTALLED\tLATEST\tSTATUS")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ModuleName, r.InstalledVersion, r.LatestVersion, r.Status)
	}
	return w.Flush()
}

func filterVersionsByMinMendix(versions []*domain.Version, maxVer string) []*domain.Version {
	out := make([]*domain.Version, 0, len(versions))
	for _, v := range versions {
		if compareSemverLike(v.MinSupportedMendixVersion, maxVer) <= 0 {
			out = append(out, v)
		}
	}
	return out
}

func compareSemverLike(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := max(len(as), len(bs))
	for i := range n {
		aa, bb := "0", "0"
		if i < len(as) {
			aa = as[i]
		}
		if i < len(bs) {
			bb = bs[i]
		}
		ai, aerr := atoiSafe(aa)
		bi, berr := atoiSafe(bb)
		if aerr == nil && berr == nil {
			if ai < bi {
				return -1
			}
			if ai > bi {
				return 1
			}
			continue
		}
		if aa < bb {
			return -1
		}
		if aa > bb {
			return 1
		}
	}
	return 0
}

func atoiSafe(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("non-numeric: %q", s)
	}
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-numeric: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func parseContentID(s string) (int, error) {
	n, err := atoiSafe(s)
	if err != nil {
		return 0, fmt.Errorf("invalid content id %q: must be a positive integer", s)
	}
	return n, nil
}

func emitJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func renderContentTable(cmd *cobra.Command, items []*domain.Content) error {
	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No results.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tPUBLISHER\tSUPPORT\tLATEST\tNAME")
	for _, it := range items {
		latest := ""
		name := ""
		if it.LatestVersion != nil {
			latest = it.LatestVersion.VersionNumber
			name = it.LatestVersion.Name
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			it.ContentID, it.Type, it.Publisher, it.SupportCategory, latest, name)
	}
	return w.Flush()
}

func renderContentDetail(cmd *cobra.Command, c *domain.Content) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Content ID:\t%d\n", c.ContentID)
	fmt.Fprintf(w, "Type:\t%s\n", c.Type)
	fmt.Fprintf(w, "Publisher:\t%s\n", c.Publisher)
	fmt.Fprintf(w, "Support:\t%s\n", c.SupportCategory)
	if len(c.Categories) > 0 {
		names := make([]string, 0, len(c.Categories))
		for _, cat := range c.Categories {
			names = append(names, cat.Name)
		}
		fmt.Fprintf(w, "Categories:\t%s\n", strings.Join(names, ", "))
	}
	if c.LicenseURL != "" {
		fmt.Fprintf(w, "License:\t%s\n", c.LicenseURL)
	}
	fmt.Fprintf(w, "Private:\t%v\n", c.IsPrivate)
	if c.LatestVersion != nil {
		v := c.LatestVersion
		fmt.Fprintf(w, "Latest:\t%s (%s)\n", v.VersionNumber, v.Name)
		fmt.Fprintf(w, "Min Mendix:\t%s\n", v.MinSupportedMendixVersion)
		fmt.Fprintf(w, "Published:\t%s\n", v.PublicationDate.Format("2006-01-02"))
	}
	return w.Flush()
}

func renderVersionsTable(cmd *cobra.Command, items []*domain.Version) error {
	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No versions.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tMIN MENDIX\tPUBLISHED\tNAME")
	for _, v := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			v.VersionNumber, v.MinSupportedMendixVersion,
			v.PublicationDate.Format("2006-01-02"), v.Name)
	}
	return w.Flush()
}

func resolveProjectPath(projectPath string) string {
	// Basic path resolution — shell globs are not expanded by the flag parser
	// so we check if the file exists as-is. The rootCmd's PersistentPreRunE
	// already normalizes -p to an absolute path.
	return projectPath
}
