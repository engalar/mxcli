// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// resolveJarDependencies materializes the Maven JAR dependencies declared in the
// model (via `ALTER MODULE ... ADD JAR DEPENDENCY (...)`) into the project's
// userlib/ directory so the subsequent mxbuild Java compile can resolve them.
//
// mxbuild only compiles against jars already present in userlib/vendorlib (see
// the generated deployment/build.gradle compile task) and it *regenerates*
// build.gradle on every run, so editing that file to add a Maven repository is
// futile. Instead we run a small, self-contained Gradle build with mxbuild's
// bundled Gradle: it resolves the declared coordinates against Maven Central —
// including transitive dependencies, cached under ~/.gradle — and copies the
// resolved jars into userlib/. This mirrors Studio Pro's dependency management.
//
// Only dependencies flagged "included" (Mendix "Include in deployment") are
// materialized. Resolution failures are non-fatal warnings: if a jar is truly
// required the later mxbuild compile fails with an actionable "cannot find
// symbol" error, and the user can drop the jar into userlib/ manually.
func resolveJarDependencies(projectPath, mxbuildPath, javaHome string, w io.Writer) error {
	deps, err := readIncludedJarDependencies(projectPath)
	if err != nil {
		return err
	}
	if len(deps) == 0 {
		return nil
	}

	gradleBin := bundledGradlePath(mxbuildPath)
	if gradleBin == "" {
		fmt.Fprintf(w, "  WARNING: bundled Gradle not found next to mxbuild; skipping JAR dependency resolution.\n")
		fmt.Fprintf(w, "  Place the %d declared jar(s) manually in %s if the Java compile fails.\n",
			len(deps), filepath.Join(filepath.Dir(projectPath), "userlib"))
		return nil
	}

	userlib := filepath.Join(filepath.Dir(projectPath), "userlib")
	if err := os.MkdirAll(userlib, 0o755); err != nil {
		return fmt.Errorf("creating userlib directory: %w", err)
	}

	workDir, err := os.MkdirTemp("", "mxcli-jardeps-")
	if err != nil {
		return fmt.Errorf("creating temp gradle project: %w", err)
	}
	defer os.RemoveAll(workDir)

	if err := writeJarDepsGradle(workDir, userlib, deps); err != nil {
		return err
	}

	fmt.Fprintf(w, "Resolving %d JAR dependency(ies) into userlib/ (Gradle + Maven Central)...\n", len(deps))
	for _, d := range deps {
		fmt.Fprintf(w, "  • %s\n", d)
	}

	cmd := exec.Command(gradleBin, "-p", workDir, "copyToUserlib",
		"--no-daemon", "--console=plain", "-q")
	cmd.Env = append(os.Environ(), "JAVA_HOME="+javaHome)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		// Non-fatal: mxbuild will surface a precise compile error if a required
		// jar is genuinely missing.
		fmt.Fprintf(w, "  WARNING: Gradle dependency resolution failed: %v\n", err)
		fmt.Fprintf(w, "  If the Java compile fails on a missing symbol, place the jar(s) manually in %s\n", userlib)
		return nil
	}
	fmt.Fprintln(w, "  JAR dependencies resolved into userlib/.")
	return nil
}

// readIncludedJarDependencies opens the MPR and returns the deduplicated
// "group:artifact:version" coordinates of all included jar dependencies across
// user modules (System module excluded).
func readIncludedJarDependencies(projectPath string) ([]string, error) {
	be, err := mprbackend.NewFromPath(projectPath)
	if err != nil {
		return nil, fmt.Errorf("opening project for jar dependencies: %w", err)
	}
	defer be.Disconnect()

	modules, err := be.ListModules()
	if err != nil {
		return nil, fmt.Errorf("listing modules for jar dependencies: %w", err)
	}

	seen := make(map[string]bool)
	var deps []string
	for _, m := range modules {
		if m == nil || m.Name == "System" {
			continue
		}
		ms, err := be.GetModuleSettings(m.ID)
		if err != nil || ms == nil {
			continue
		}
		for _, d := range ms.JarDependencies {
			if d == nil || !d.IsIncluded {
				continue
			}
			if d.GroupID == "" || d.ArtifactID == "" || d.Version == "" {
				continue
			}
			coord := d.GroupID + ":" + d.ArtifactID + ":" + d.Version
			if seen[coord] {
				continue
			}
			seen[coord] = true
			deps = append(deps, coord)
		}
	}
	return deps, nil
}

// writeJarDepsGradle writes a self-contained Gradle project into workDir that
// resolves deps (with transitive dependencies) and copies them into userlib.
func writeJarDepsGradle(workDir, userlib string, deps []string) error {
	var b strings.Builder
	b.WriteString("repositories { mavenCentral() }\n")
	b.WriteString("configurations { mxdeps }\n")
	b.WriteString("dependencies {\n")
	for _, d := range deps {
		b.WriteString("    mxdeps '" + d + "'\n")
	}
	b.WriteString("}\n")
	// Gradle accepts forward slashes in paths on every platform.
	into := strings.ReplaceAll(userlib, "\\", "/")
	b.WriteString("tasks.register('copyToUserlib', Copy) {\n")
	b.WriteString("    from configurations.mxdeps\n")
	b.WriteString("    into '" + into + "'\n")
	b.WriteString("}\n")

	if err := os.WriteFile(filepath.Join(workDir, "build.gradle"), []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing jardeps build.gradle: %w", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "settings.gradle"),
		[]byte("rootProject.name = 'mxcli-jardeps'\n"), 0o644); err != nil {
		return fmt.Errorf("writing jardeps settings.gradle: %w", err)
	}
	return nil
}

// bundledGradlePath returns the path to the Gradle launcher shipped alongside
// mxbuild (modeler/tools/gradle/bin/gradle), or "" if it cannot be found.
func bundledGradlePath(mxbuildPath string) string {
	name := "gradle"
	if runtime.GOOS == "windows" {
		name = "gradle.bat"
	}
	p := filepath.Join(filepath.Dir(mxbuildPath), "tools", "gradle", "bin", name)
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return p
	}
	return ""
}
