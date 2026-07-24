// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// BuildOptions configures the build command.
type BuildOptions struct {
	// ProjectPath is the path to the .mpr file.
	ProjectPath string

	// MxBuildPath is an explicit path to the mxbuild executable.
	MxBuildPath string

	// SkipCheck skips the 'mx check' pre-build validation.
	SkipCheck bool

	// Stdout for output messages.
	Stdout io.Writer

	// OnPhase is called at each phase boundary. nil = no callback (CLI mode).
	OnPhase func(name, status string, pct int, msg string)

	// OnOutput is called for each line written to stdout. nil = no callback.
	OnOutput func(line string)

	// OnBuildStart, if set, is called with the mxbuild process PID after it
	// starts. The caller can use this to track or kill the build process.
	OnBuildStart func(pid int)
}

// Build runs MxBuild to deploy the project to deployment/.
func Build(opts BuildOptions) error {
	w := opts.Stdout
	if w == nil {
		w = os.Stdout
	}

	if opts.OnPhase != nil {
		opts.OnPhase("detect", "running", 2, "Detecting project version...")
	}
	fmt.Fprintln(w, "Detecting project version...")
	be, err := mprbackend.NewFromPath(opts.ProjectPath)
	if err != nil {
		return fmt.Errorf("opening project: %w", err)
	}
	pv := be.ProjectVersion()
	be.Disconnect()

	fmt.Fprintf(w, "  Mendix version: %s\n", pv.ProductVersion)

	if !pv.IsAtLeastFull(11, 6, 1) {
		return fmt.Errorf("Mendix >= 11.6.1 required, found %s", pv.ProductVersion)
	}

	// Step 2: Resolve MxBuild
	if opts.OnPhase != nil {
		opts.OnPhase("detect", "completed", 5, "")
		opts.OnPhase("mxbuild", "running", 10, "Resolving MxBuild...")
	}
	fmt.Fprintln(w, "Resolving MxBuild...")
	mxbuildPath, err := resolveMxBuild(opts.MxBuildPath, pv.ProductVersion)
	if err != nil {
		fmt.Fprintln(w, "  MxBuild not found locally, downloading from CDN...")
		mxbuildPath, err = DownloadMxBuild(pv.ProductVersion, w)
		if err != nil {
			return fmt.Errorf("downloading mxbuild: %w", err)
		}
	}
	fmt.Fprintf(w, "  MxBuild: %s\n", mxbuildPath)

	// Step 3: Resolve JDK 21
	if opts.OnPhase != nil {
		opts.OnPhase("mxbuild", "completed", 15, "")
		opts.OnPhase("jdk", "running", 18, "Resolving JDK 21...")
	}
	fmt.Fprintln(w, "Resolving JDK 21...")
	javaHome, err := resolveJDK21()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "  JAVA_HOME: %s\n", javaHome)
	fmt.Fprintf(w, "  Java: %s\n", javaVersionString(javaHome))

	// Step 3b: Resolve Maven JAR dependencies into userlib/
	if opts.OnPhase != nil {
		opts.OnPhase("jdk", "completed", 20, "")
		opts.OnPhase("jars", "running", 22, "Resolving JAR dependencies...")
	}
	if err := resolveJarDependencies(opts.ProjectPath, mxbuildPath, javaHome, w); err != nil {
		return fmt.Errorf("resolving jar dependencies: %w", err)
	}

	// Step 3c: Ensure demo users exist (for --security demo/production)
	if opts.OnPhase != nil {
		opts.OnPhase("jars", "completed", 25, "")
		opts.OnPhase("demousers", "running", 27, "Checking demo users...")
	}
	EnsureDemoUsersIfNeeded(opts.ProjectPath, w)

	// Step 4: Pre-build check
	if opts.OnPhase != nil {
		opts.OnPhase("jars", "completed", 25, "")
		opts.OnPhase("check", "running", 28, "Checking project for errors...")
	}
	if !opts.SkipCheck {
		fmt.Fprintln(w, "Checking project for errors...")
		mxPath, err := ResolveMxForVersion(opts.MxBuildPath, pv.ProductVersion)
		if err != nil {
			fmt.Fprintf(w, "  Skipping check: %v\n", err)
		} else {
			cmd := exec.Command(mxPath, "check", opts.ProjectPath)
			cmd.Stdout = w
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("project has errors (fix them or use --skip-check to bypass): %w", err)
			}
			fmt.Fprintln(w, "  Project check passed.")
		}
	}

	// Step 5: Run MxBuild with --target=deploy.
	if opts.OnPhase != nil {
		opts.OnPhase("check", "completed", 30, "")
		opts.OnPhase("build", "running", 40, "Running MxBuild...")
	}
	projectDir := filepath.Dir(opts.ProjectPath)
	deployDir := filepath.Join(projectDir, "deployment")
	javaExePath := filepath.Join(javaHome, "bin", "java")

	fmt.Fprintf(w, "Running MxBuild (target=deploy)...\n")
	fmt.Fprintf(w, "  Output: %s\n", deployDir)

	cmd := exec.Command(mxbuildPath,
		"--target=deploy",
		fmt.Sprintf("--java-home=%s", javaHome),
		fmt.Sprintf("--java-exe-path=%s", javaExePath),
		opts.ProjectPath,
	)
	CmdWithPdeathsig(cmd)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mxbuild start failed: %w", err)
	}
	if opts.OnBuildStart != nil {
		opts.OnBuildStart(cmd.Process.Pid)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("mxbuild failed: %w", err)
	}

	// Write minimal config.json so isDeployLayout works.
	if err := writeDeployConfigJSON(deployDir); err != nil {
		return fmt.Errorf("writing deployment config.json: %w", err)
	}

	// React client frontend build (when rollup.config.mjs present)
	if RollupConfigExists(deployDir) {
		if err := BuildFrontend(FrontendBuildOptions{
			DeployDir:  deployDir,
			MxBuildDir: filepath.Dir(mxbuildPath),
			Stdout:     w,
		}); err != nil {
			return err
		}
	}

	if opts.OnPhase != nil {
		opts.OnPhase("done", "completed", 100, "Build complete")
	}
	fmt.Fprintln(w, "Build complete.")
	return nil
}

// writeDeployConfigJSON writes a minimal deployment/model/config.json if it does
// not already exist. mxbuild --target=deploy does not generate this file, but
// StartLocal needs it to detect and start from the deploy layout.
func writeDeployConfigJSON(deployDir string) error {
	cfgPath := filepath.Join(deployDir, "model", "config.json")
	if _, err := os.Stat(cfgPath); err == nil {
		return nil // already exists (Studio Pro ran locally)
	}
	cfg := []byte(`{"Configuration":{},"Constants":{},"AdminPassword":""}`)
	return os.WriteFile(cfgPath, cfg, 0644)
}
