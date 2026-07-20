// SPDX-License-Identifier: Apache-2.0

// mxcli is a command-line interface for working with Mendix projects using MDL syntax.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/executor/domainmodel"
	"github.com/mendixlabs/mxcli/mdl/executor/microflow"
	"github.com/mendixlabs/mxcli/mdl/executor/misc"
	"github.com/mendixlabs/mxcli/mdl/executor/page"
	"github.com/mendixlabs/mxcli/mdl/executor/query"
	"github.com/mendixlabs/mxcli/mdl/executor/security"
	"github.com/mendixlabs/mxcli/mdl/executor/workflow"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	Version   = ""
	BuildTime = ""
	CommitSHA = ""
)

const warningBanner = "WARNING: This is a vibe-coded PoC, alpha quality, use with caution.\n"

func main() {
	// CPU profiling (dev): set MXCLI_CPU_PROFILE=profile.prof
	var profileFile *os.File
	if cp := os.Getenv("MXCLI_CPU_PROFILE"); cp != "" {
		f, err := os.Create(cp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cpuprofile: %v\n", err)
		} else {
			pprof.StartCPUProfile(f)
			profileFile = f
		}
	}

	code := run()
	if profileFile != nil {
		pprof.StopCPUProfile()
		profileFile.Close()
	}
	os.Exit(code)
}

func run() int {
	// --internal-update mode: spawned by upgrade/rollback to replace the binary
	// after the parent process exits. Must be handled before Cobra parsing.
	if len(os.Args) > 1 && os.Args[1] == "--internal-update" {
		pid, newBin, target, ok := parseInternalUpdateArgs(os.Args[2:])
		if !ok {
			fmt.Fprintln(os.Stderr, "mxcli: invalid --internal-update args")
			return 1
		}
		if err := runInternalUpdate(pid, newBin, target, &RealPIDWaiter{}, 30*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli: update failed: %v\n", err)
			return 1
		}
		return 0
	}

	// Show warning banner unless --quiet, -q, --help, -h, or --version is passed
	if !shouldSuppressWarning() {
		fmt.Fprint(os.Stderr, warningBanner)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// shouldSuppressWarning checks if the warning should be suppressed
func shouldSuppressWarning() bool {
	// Check environment variable first (best for automated/CI usage)
	if os.Getenv("MXCLI_QUIET") != "" {
		return true
	}

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-q", "--quiet", "-h", "--help", "--version":
			return true
		case "help", "version", "changelog", "completion":
			return true
		}
	}
	return false
}

// discoverProjectPath looks for a single .mpr file in the current directory.
// Returns the filename if exactly one is found, otherwise returns "".
func discoverProjectPath() string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}
	var mprFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mpr") {
			mprFiles = append(mprFiles, e.Name())
		}
	}
	if len(mprFiles) == 1 {
		return mprFiles[0]
	}
	return ""
}

var rootCmd = &cobra.Command{
	Use:   "mxcli",
	Short: "Mendix CLI - Work with Mendix projects using MDL syntax",
	Long: `mxcli is a command-line interface for working with Mendix projects.

It supports MDL (Mendix Definition Language), a SQL-like syntax for
reading and modifying Mendix domain models.

Examples:
  # Get started with Claude Code in a Mendix project
  mxcli init /path/to/mendix-project; claude

  # Execute MDL commands directly
  mxcli -c "CONNECT LOCAL 'app.mpr'; SHOW ENTITIES;"

  # Connect to project and show entities
  mxcli -p app.mpr -c "SHOW ENTITIES"

  # Enable trace output for debugging (-v)
  mxcli -v -p app.mpr -c "SHOW ENTITIES"

  # Enable full debug output to stderr (-vv)
  mxcli -vv -p app.mpr -c "SHOW ENTITIES"
`,
	Version: version,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		commands, _ := cmd.Flags().GetString("command")
		projectPath, _ := cmd.Flags().GetString("project")

		if commands != "" {
			// Execute commands from -c flag
			exec, logger := buildExec("batch", cmd.OutOrStdout())
			defer logger.Close()
			defer exec.Close()

			// Suppress status messages when stdout is a pipe so that
			// output can be piped directly to other tools (e.g. mxcli fmt).
			if fi, statErr := os.Stdout.Stat(); statErr == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
				exec.SetQuiet(true)
			}

			// Auto-connect if project specified
			if projectPath != "" {
				commands = fmt.Sprintf("CONNECT LOCAL '%s'; %s", projectPath, commands)
			}

			prog, errs := visitor.Build(commands)
			if len(errs) > 0 {
				for _, err := range errs {
					fmt.Fprintf(cmd.ErrOrStderr(), "Parse error: %v\n", err)
				}
				return fmt.Errorf("parse failed")
			}

			if err := exec.ExecuteProgram(prog); err != nil {
				if errors.Is(err, executor.ErrExit) {
					return nil
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				return err
			}
			return nil
		}
		return fmt.Errorf("no command specified. Use -c to pass an MDL command or exec to run a script")
	},
}

// globalJSONFlag is set by PersistentPreRun when --json is passed.
var globalJSONFlag bool

// globalVerboseLevel is set by PersistentPreRunE when -v or -vv is passed.
// 0 = off, 1 = trace (human-readable text), 2 = debug (JSON).
var globalVerboseLevel int

// resolveFormat returns the effective output format for a command.
// If the global --json flag is set and the command has a --format flag, it returns "json".
// Otherwise it returns the command's --format flag value (or the provided default).
func resolveFormat(cmd *cobra.Command, defaultFormat string) string {
	if globalJSONFlag {
		return "json"
	}
	if cmd.Flags().Lookup("format") != nil {
		f, _ := cmd.Flags().GetString("format")
		return f
	}
	return defaultFormat
}

// buildExec creates an Executor for the given mode and output writer.
// A fresh MprBackend is created per CONNECT statement.
// The caller must call logger.Close() and exec.Close() when done.
func buildExec(mode string, out io.Writer) (*executor.Executor, *diaglog.Logger) {
	logger := diaglog.Init(version, mode, globalVerboseLevel)
	b := executor.Build().Out(out).WithLogger(logger).WithFactory(func() backend.ConnectionBackend {
		return mprbackend.New()
	})
	if globalJSONFlag {
		b = b.Format(executor.FormatJSON)
	}
	exec := b.Create()

	// Register domain-specific handlers from subpackages.
	deps := exec.BuildHandlerDeps()
	microflow.RegisterHandlers(exec.Registry(), deps)
	page.RegisterHandlers(exec.Registry(), deps)
	workflow.RegisterHandlers(exec.Registry(), deps)
	domainmodel.RegisterHandlers(exec.Registry(), deps)
	security.RegisterHandlers(exec.Registry(), deps)
	query.RegisterHandlers(exec.Registry(), deps)
	misc.RegisterHandlers(exec.Registry(), deps)
	exec.AddReregister(func(fresh *executor.HandlerDeps) {
		microflow.RegisterHandlers(exec.Registry(), fresh)
		page.RegisterHandlers(exec.Registry(), fresh)
		workflow.RegisterHandlers(exec.Registry(), fresh)
		domainmodel.RegisterHandlers(exec.Registry(), fresh)
		security.RegisterHandlers(exec.Registry(), fresh)
		query.RegisterHandlers(exec.Registry(), fresh)
		misc.RegisterHandlers(exec.Registry(), fresh)
	})
	return exec, logger
}

// executeMDL is a helper to execute MDL commands with a project.
// out should be cmd.OutOrStdout() so daemon-server mode routes output to the socket.
// Returns an error instead of calling os.Exit so the daemon survives command failures.
func executeMDL(projectPath, mdlCmd string, out io.Writer) error {
	exec, logger := buildExec("subcommand", out)
	defer logger.Close()
	defer exec.Close()

	fullCmd := fmt.Sprintf("CONNECT LOCAL '%s'; %s", projectPath, mdlCmd)
	prog, errs := visitor.Build(fullCmd)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(out, "Parse error: %v\n", err)
		}
		return fmt.Errorf("parse failed")
	}

	if err := exec.ExecuteProgram(prog); err != nil {
		if errors.Is(err, executor.ErrExit) {
			return nil
		}
		return err
	}
	return nil
}

func init() {
	if Version != "" {
		version = Version
	}
	shaSuffix := ""
	if CommitSHA != "" {
		shaSuffix = " commit " + CommitSHA
	}
	if BuildTime != "" {
		rootCmd.Version = version + " (" + BuildTime + ")" + shaSuffix
	} else {
		rootCmd.Version = version + shaSuffix
	}

	// Normalise -p to an absolute path before any subcommand runs, so that
	// daemon socket paths (which are derived from the MPR path) are stable
	// regardless of the caller's working directory.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Read verbose level (CountP: -v=1, -vv=2)
		if v, err := cmd.Root().PersistentFlags().GetCount("verbose"); err == nil {
			globalVerboseLevel = v
		}

		// Auto-discover project if -p not specified
		if p, _ := cmd.Root().PersistentFlags().GetString("project"); p == "" {
			if discovered := discoverProjectPath(); discovered != "" {
				_ = cmd.Root().PersistentFlags().Set("project", discovered)
				fmt.Fprintf(os.Stderr, "Using project: %s\n", discovered)
			}
		}

		// Set global JSON flag
		globalJSONFlag, _ = cmd.Root().PersistentFlags().GetBool("json")

		// Normalise -p to an absolute path.
		// Use forward slashes on Windows: MDL single-quoted strings interpret
		// backslash escape sequences (\t = TAB, \n = LF, etc.), so a Windows
		// path like D:\testdata\corpus would corrupt the CONNECT LOCAL command.
		if p, _ := cmd.Root().PersistentFlags().GetString("project"); p != "" {
			abs, err := filepath.Abs(p)
			if err != nil {
				return fmt.Errorf("resolving -p path: %w", err)
			}
			abs = filepath.ToSlash(abs)
			if err := cmd.Root().PersistentFlags().Set("project", abs); err != nil {
				return err
			}
		}
		return nil
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("project", "p", "", "Path to Mendix project (.mpr file)")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().CountP("verbose", "v", "Enable verbose output (-v trace, -vv debug)")
	rootCmd.Flags().StringP("command", "c", "", "Execute MDL command(s) and exit")

	// Check command flags
	checkCmd.Flags().BoolP("references", "r", false, "Validate references against the project")
	checkCmd.Flags().String("format", "text", "Output format: text, json, sarif")

	// Describe command flags
	describeCmd.Flags().StringP("format", "f", "mdl", "Output format: mdl, json, mermaid, elk")

	// Search command flags
	searchCmd.Flags().StringP("format", "f", "table", "Output format: table, names, json")
	searchCmd.Flags().BoolP("quiet", "q", false, "Suppress connection and status messages (for piping)")

	// Callers/callees command flags
	callersCmd.Flags().BoolP("transitive", "t", false, "Find transitive (indirect) callers")
	calleesCmd.Flags().BoolP("transitive", "t", false, "Find transitive (indirect) callees")

	// Structure command flags
	structureCmd.Flags().IntP("depth", "d", 2, "Detail level: 1=counts, 2=signatures, 3=types")
	structureCmd.Flags().StringP("module", "m", "", "Filter to specific module")
	structureCmd.Flags().Bool("all", false, "Include system/marketplace modules")

	// Context command flags
	contextCmd.Flags().IntP("depth", "d", 2, "Depth for call chain traversal")

	// Lint command flags
	lintCmd.Flags().StringP("format", "f", "text", "Output format: text, json, sarif")
	lintCmd.Flags().BoolP("color", "", false, "Use colored output")
	lintCmd.Flags().BoolP("list-rules", "l", false, "List available lint rules")
	lintCmd.Flags().StringSliceP("exclude", "e", nil, "Modules to exclude from linting")

	// Report command flags
	reportCmd.Flags().StringP("format", "f", "markdown", "Output format: markdown, json, html")
	reportCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	reportCmd.Flags().StringSliceP("exclude", "e", nil, "Modules to exclude from report")

	// Test command flags
	testRunCmd.Flags().BoolP("list", "l", false, "List tests without executing")
	testRunCmd.Flags().StringP("junit", "j", "", "Write JUnit XML results to file")
	testRunCmd.Flags().BoolP("skip-build", "s", false, "Skip build step (reuse existing deployment)")
	testRunCmd.Flags().Bool("verbose", false, "Show all runtime log output")
	testRunCmd.Flags().BoolP("color", "", false, "Use colored output")
	testRunCmd.Flags().StringP("timeout", "t", "5m", "Timeout for runtime startup and test execution")

	// Eval command flags
	evalCheckCmd.Flags().StringP("test", "t", "", "Run only specific test ID")
	evalCheckCmd.Flags().BoolP("skip-mx-check", "", false, "Skip mx check validation")
	evalCheckCmd.Flags().StringP("output", "o", "", "Output directory for reports (default: no file output)")
	evalCheckCmd.Flags().StringP("mxcli-path", "", "", "Path to mxcli binary (default: self)")
	evalCheckCmd.Flags().BoolP("color", "", false, "Use colored output")
	evalCmd.AddCommand(evalCheckCmd)
	evalCmd.AddCommand(evalListCmd)

	// Add subcommands
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(describeCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(diffLocalCmd)
	rootCmd.AddCommand(callersCmd)
	rootCmd.AddCommand(calleesCmd)
	rootCmd.AddCommand(refsCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(structureCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(projectTreeCmd)
	rootCmd.AddCommand(diagCmd)
	rootCmd.AddCommand(testRunCmd)
	rootCmd.AddCommand(playwrightCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(fmtCmd)
	rootCmd.AddCommand(mprPackCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(oqlCmd)
	rootCmd.AddCommand(sqlCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(reloadCmd())
	rootCmd.AddCommand(syntaxCmd)
}


