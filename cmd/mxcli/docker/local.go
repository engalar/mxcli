// cmd/mxcli/docker/local.go
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ProcessStarter abstracts exec.Cmd execution for testing.
type ProcessStarter interface {
	Run(cmd *exec.Cmd) error
}

// RealStarter executes the command for real (used in production).
type RealStarter struct{}

func (r *RealStarter) Run(cmd *exec.Cmd) error { return cmd.Run() }

// LocalRunOptions configures StartLocal.
type LocalRunOptions struct {
	// PadDir is the PAD output directory (.docker/build/ by default).
	// Can also be a deploy-format directory (deployment/) when Studio Pro is installed.
	PadDir string
	// DB is an optional postgres:// URL. Empty = use config defaults (HSQLDB).
	DB string
	// AdminPassword sets ADMIN_ADMINPASSWORD and RUNTIME_ADMINUSER_PASSWORD.
	// Defaults to "Admin123!" if empty.
	AdminPassword string
	// Stdout for runtime log output (defaults to os.Stdout).
	Stdout io.Writer
	// Stderr for runtime error output (defaults to os.Stderr).
	Stderr io.Writer
	// Starter is the process runner. Nil = RealStarter (exec.Cmd.Run).
	Starter ProcessStarter
}

// StartLocal starts the Mendix runtime from a pre-built directory without Docker.
// Supports two layouts:
//   - PAD layout (.docker/build/): execs bin/start.bat (Windows) or bin/start (Unix).
//   - Deploy layout (deployment/): uses Studio Pro runtime with generated config.
func StartLocal(opts LocalRunOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	// Deploy layout: Studio Pro --target=deploy output (no ZIP, no start.bat).
	if isDeployLayout(opts.PadDir) {
		if err := preflightLocal(opts.PadDir, stderr, true); err != nil {
			return err
		}
		return startFromDeployLayout(opts, stdout, stderr)
	}

	// PAD layout: classic portable-app-package output.
	if !hasExtractedPADLayout(opts.PadDir) {
		return fmt.Errorf("no PAD found at %s — run 'mxcli local build -p app.mpr' first", opts.PadDir)
	}
	if err := preflightLocal(opts.PadDir, stderr, false); err != nil {
		return err
	}
	return startFromPADLayout(opts, stdout, stderr)
}

// IsDeployLayout reports whether dir is a Studio Pro deploy-format directory.
// Identified by having model/config.json and model/model.mdp but no bin/start.bat.
func IsDeployLayout(dir string) bool {
	return isDeployLayout(dir)
}

func isDeployLayout(dir string) bool {
	configJSON := filepath.Join(dir, "model", "config.json")
	modelMdp := filepath.Join(dir, "model", "model.mdp")
	startBat := filepath.Join(dir, "bin", "start.bat")
	startSh := filepath.Join(dir, "bin", "start")

	_, hasConfig := os.Stat(configJSON)
	_, hasModel := os.Stat(modelMdp)
	_, hasBat := os.Stat(startBat)
	_, hasSh := os.Stat(startSh)

	return hasConfig == nil && hasModel == nil && hasBat != nil && hasSh != nil
}

// deployConfig mirrors the structure of deployment/model/config.json.
type deployConfig struct {
	Configuration map[string]string `json:"Configuration"`
	Constants     map[string]string `json:"Constants"`
	AdminPassword string            `json:"AdminPassword"`
}

// deployMetadata mirrors the relevant fields of deployment/model/metadata.json.
type deployMetadata struct {
	RuntimeVersion string `json:"RuntimeVersion"`
}

// startFromDeployLayout starts the runtime from a Studio Pro deploy-format directory.
// Uses the Mendix installation runtime (MX_INSTALL_PATH) and a generated HOCON config.
func startFromDeployLayout(opts LocalRunOptions, stdout, stderr io.Writer) error {
	// 1. Read config.json for DB config and constants.
	cfgPath := filepath.Join(opts.PadDir, "model", "config.json")
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("reading deployment config: %w", err)
	}
	var dcfg deployConfig
	if err := json.Unmarshal(cfgData, &dcfg); err != nil {
		return fmt.Errorf("parsing deployment config: %w", err)
	}

	// 1b. Read metadata.json to get the required Mendix runtime version.
	var runtimeVersion string
	if metaData, err2 := os.ReadFile(filepath.Join(opts.PadDir, "model", "metadata.json")); err2 == nil {
		var meta deployMetadata
		if json.Unmarshal(metaData, &meta) == nil {
			runtimeVersion = meta.RuntimeVersion
		}
	}

	// 2. Find Mendix runtime launcher for the correct version.
	mxInstall, err := resolveMxInstallPathForVersion(runtimeVersion)
	if err != nil {
		msg := fmt.Sprintf("cannot find Mendix %s runtime: %v\n", runtimeVersion, err)
		if runtimeVersion != "" {
			msg += fmt.Sprintf("Install Mendix Studio Pro %s or set MX_INSTALL_PATH", runtimeVersion)
		} else {
			msg += "Install Studio Pro or set MX_INSTALL_PATH"
		}
		return fmt.Errorf("%s", msg)
	}
	launcherJar := filepath.Join(mxInstall, "runtime", "launcher", "runtimelauncher.jar")
	if _, err := os.Stat(launcherJar); err != nil {
		return fmt.Errorf("runtimelauncher.jar not found at %s", launcherJar)
	}

	// 3. Resolve JDK 21.
	javaHome, err := resolveJDK21()
	if err != nil {
		return err
	}
	javaExe := filepath.Join(javaHome, "bin", "java")

	// 4. Generate HOCON config file in a temp directory.
	tmpDir, err := os.MkdirTemp("", "mxcli-local-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	adminPass := opts.AdminPassword
	if adminPass == "" {
		adminPass = "Admin123!"
	}
	hoconPath := filepath.Join(tmpDir, "local.conf")
	if err := writeDeployHOCON(hoconPath, dcfg, opts.DB, adminPass); err != nil {
		return fmt.Errorf("generating config: %w", err)
	}

	// 5. Build environment.
	env := append(os.Environ(),
		"MX_INSTALL_PATH="+mxInstall,
		"M2EE_ADMIN_PASS="+adminPass,
		"ADMIN_ADMINPASSWORD="+adminPass,
		"RUNTIME_ADMINUSER_PASSWORD="+adminPass,
		"JAVA_HOME="+javaHome,
		"PATH="+filepath.Join(javaHome, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	// 6. Launch.
	cmd := exec.Command(javaExe,
		"-DMX_LOG_LEVEL=INFO",
		"-Dfile.encoding=UTF-8",
		"-jar", launcherJar,
		opts.PadDir,
		hoconPath,
	)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	starter := opts.Starter
	if starter == nil {
		starter = &RealStarter{}
	}
	return starter.Run(cmd)
}

// writeDeployHOCON writes a HOCON config file for a deploy-layout startup.
func writeDeployHOCON(path string, dcfg deployConfig, dbURL, adminPass string) error {
	cfg := dcfg.Configuration
	dbType := cfg["DatabaseType"]
	if dbType == "" {
		dbType = "HSQLDB"
	}
	dbName := cfg["DatabaseName"]
	if dbName == "" {
		dbName = "default"
	}
	appURL := cfg["ApplicationRootUrl"]
	if appURL == "" {
		appURL = "http://localhost:8080/"
	}

	var sb strings.Builder

	// Runtime parameters from config.json.
	sb.WriteString("runtime.params {\n")
	if dbURL != "" {
		// Override with postgres URL — env vars will handle the specifics.
	} else {
		fmt.Fprintf(&sb, "  DatabaseType = %s\n", dbType)
		fmt.Fprintf(&sb, "  DatabaseName = \"%s\"\n", dbName)
		if h := cfg["DatabaseHost"]; h != "" {
			fmt.Fprintf(&sb, "  DatabaseHost = \"%s\"\n", h)
		}
		if u := cfg["DatabaseUserName"]; u != "" {
			fmt.Fprintf(&sb, "  DatabaseUserName = \"%s\"\n", u)
		}
	}
	fmt.Fprintf(&sb, "  ApplicationRootUrl = \"%s\"\n", appURL)
	fmt.Fprintf(&sb, "  HashAlgorithm = \"BCRYPT:12\"\n")
	fmt.Fprintf(&sb, "  DTAPMode = D\n")
	if s := cfg["ScheduledEventExecution"]; s != "" {
		fmt.Fprintf(&sb, "  ScheduledEventExecution = \"%s\"\n", s)
	} else {
		sb.WriteString("  ScheduledEventExecution = NONE\n")
	}
	sb.WriteString("  MyScheduledEvents = \"\"\n")
	sb.WriteString("  CACertificates = \"\"\n")
	sb.WriteString("  ClientCertificates = \"\"\n")
	sb.WriteString("  ClientCertificatePasswords = \"\"\n")
	sb.WriteString("}\n\n")

	// Constants.
	if len(dcfg.Constants) > 0 {
		sb.WriteString("runtime.params.MicroflowConstants {\n")
		for k, v := range dcfg.Constants {
			fmt.Fprintf(&sb, "  %q = %q\n", k, v)
		}
		sb.WriteString("}\n\n")
	}

	// Admin server.
	sb.WriteString("admin {\n")
	sb.WriteString("  port = 8090\n")
	sb.WriteString("  addresses = [ localhost ]\n")
	sb.WriteString("  adminPassword = ${?ADMIN_ADMINPASSWORD}\n")
	sb.WriteString("}\n\n")

	// Runtime HTTP + admin user.
	sb.WriteString("runtime {\n")
	sb.WriteString("  http {\n")
	sb.WriteString("    port = 8080\n")
	sb.WriteString("    addresses = [ \"*\" ]\n")
	sb.WriteString("  }\n")
	sb.WriteString("  adminUser.password = ${?RUNTIME_ADMINUSER_PASSWORD}\n")
	sb.WriteString("}\n\n")

	// Logging subscriber — required: without it the launcher only starts
	// the M2EE admin server and waits for external commands instead of
	// auto-starting the runtime.
	sb.WriteString("logging = [\n")
	sb.WriteString("  {\n")
	sb.WriteString("    name = console\n")
	sb.WriteString("    type = console\n")
	sb.WriteString("    autoSubscribe = INFO\n")
	sb.WriteString("    levels {}\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// resolveMxInstallPathForVersion finds the Mendix runtime installation directory
// for the specified version. If version is "" it returns the newest installed version.
// Checks MX_INSTALL_PATH env first, then Studio Pro installations on Windows.
func resolveMxInstallPathForVersion(version string) (string, error) {
	if p := os.Getenv("MX_INSTALL_PATH"); p != "" {
		return p, nil
	}
	// Try known Studio Pro installations on Windows.
	if runtime.GOOS == "windows" {
		for _, base := range windowsProgramDirs() {
			mendixBase := filepath.Join(base, "Mendix")
			entries, err := os.ReadDir(mendixBase)
			if err != nil {
				continue
			}
			if version != "" {
				// Exact version match.
				candidate := filepath.Join(mendixBase, version)
				launcher := filepath.Join(candidate, "runtime", "launcher", "runtimelauncher.jar")
				if _, err := os.Stat(launcher); err == nil {
					return candidate, nil
				}
			} else {
				// No version required: return the newest installed version.
				var best string
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					launcher := filepath.Join(mendixBase, e.Name(), "runtime", "launcher", "runtimelauncher.jar")
					if _, err := os.Stat(launcher); err == nil {
						if e.Name() > best {
							best = e.Name()
						}
					}
				}
				if best != "" {
					return filepath.Join(mendixBase, best), nil
				}
			}
		}
	}
	// Fallback: CDN-cached mxbuild (~/.mxcli/mxbuild/{version}/).
	// Covers Linux and macOS where Studio Pro is not installed.
	if home, err2 := os.UserHomeDir(); err2 == nil {
		mxbuildRoot := filepath.Join(home, ".mxcli", "mxbuild")
		if version != "" {
			// Exact version match.
			candidate := filepath.Join(mxbuildRoot, version)
			launcher := filepath.Join(candidate, "runtime", "launcher", "runtimelauncher.jar")
			if _, err2 := os.Stat(launcher); err2 == nil {
				return candidate, nil
			}
		} else {
			// No version: return newest cached version.
			if entries, err2 := os.ReadDir(mxbuildRoot); err2 == nil {
				var best string
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					launcher := filepath.Join(mxbuildRoot, e.Name(), "runtime", "launcher", "runtimelauncher.jar")
					if _, err2 := os.Stat(launcher); err2 == nil && e.Name() > best {
						best = e.Name()
					}
				}
				if best != "" {
					return filepath.Join(mxbuildRoot, best), nil
				}
			}
		}
	}
	if version != "" {
		return "", fmt.Errorf("Mendix %s not found", version)
	}
	return "", fmt.Errorf("Mendix runtime not found")
}

// ResolveMxInstallPathForVersion is the exported wrapper for tests.
func ResolveMxInstallPathForVersion(version string) (string, error) {
	return resolveMxInstallPathForVersion(version)
}

// resolveMxInstallPath finds the newest installed Mendix runtime (version-agnostic).
func resolveMxInstallPath() (string, error) {
	return resolveMxInstallPathForVersion("")
}

// startFromPADLayout starts the runtime from a classic PAD output directory.
func startFromPADLayout(opts LocalRunOptions, stdout, stderr io.Writer) error {
	cmdArgs, err := resolveStartScript(opts.PadDir)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = opts.PadDir
	cmd.Env = append(os.Environ(), buildLocalEnv(opts.DB, opts.AdminPassword)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	starter := opts.Starter
	if starter == nil {
		starter = &RealStarter{}
	}
	return starter.Run(cmd)
}

// resolveStartScript returns [binary, args...] for the platform start script.
func resolveStartScript(padDir string) ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		bat := filepath.Join(padDir, "bin", "start.bat")
		if _, err := os.Stat(bat); err == nil {
			return []string{"cmd.exe", "/c", bat}, nil
		}
		ps1 := filepath.Join(padDir, "bin", "start.ps1")
		if _, err := os.Stat(ps1); err == nil {
			return []string{"powershell.exe", "-ExecutionPolicy", "Bypass", "-File", ps1}, nil
		}
		return nil, fmt.Errorf("no Windows start script (start.bat or start.ps1) found in %s/bin/", padDir)
	default:
		sh := filepath.Join(padDir, "bin", "start")
		if _, err := os.Stat(sh); err != nil {
			return nil, fmt.Errorf("start script not found at %s", sh)
		}
		return []string{sh}, nil
	}
}

// buildLocalEnv returns environment variables required by the Mendix runtime (PAD layout).
func buildLocalEnv(dbURL, adminPassword string) []string {
	if adminPassword == "" {
		adminPassword = "Admin123!"
	}
	env := []string{
		"ADMIN_ADMINPASSWORD=" + adminPassword,
		"RUNTIME_ADMINUSER_PASSWORD=" + adminPassword,
	}
	if dbURL != "" {
		if dbEnv, err := parseDBURL(dbURL); err == nil {
			env = append(env, dbEnv...)
		}
	}

	// Inject JAVA_HOME so bin/start can find java even when it is not in
	// the shell PATH (common in Git Bash on Windows).
	if javaHome, err := resolveJDK21(); err == nil {
		env = append(env, "JAVA_HOME="+javaHome)
		javaBin := filepath.Join(javaHome, "bin")
		if currentPath := os.Getenv("PATH"); currentPath != "" {
			env = append(env, "PATH="+javaBin+string(os.PathListSeparator)+currentPath)
		} else {
			env = append(env, "PATH="+javaBin)
		}
	}

	return env
}

// preflightLocal detects a running Mendix instance before starting a new one.
//
//  1. Port check: tries to bind the admin port (8090). If already taken, another
//     runtime is running — return a clear error instead of a confusing startup crash.
//
//  2. HSQLDB stale lock cleanup: when the previous runtime was killed the JVM may
//     leave a .lck file. On most OSes the file is not kept open after JVM exit, so
//     we simply remove it. If removal fails (process still alive), we surface the error.
func preflightLocal(dir string, stderr io.Writer, isDeployDir bool) error {
	// 1. Admin port check (default 8090).
	if ln, err := net.Listen("tcp", "127.0.0.1:8090"); err != nil {
		return fmt.Errorf("Mendix runtime already running on port 8090.\n" +
			"Stop the existing instance (kill the Java process) before starting a new one.")
	} else {
		ln.Close()
	}

	// 2. HSQLDB stale lock cleanup — path differs by layout.
	var dbRoot string
	if isDeployDir {
		dbRoot = filepath.Join(dir, "data", "database", "hsqldb")
	} else {
		dbRoot = filepath.Join(dir, "app", "data", "database", "hsqldb")
	}
	_ = filepath.Walk(dbRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".lck") {
			return nil
		}
		if rmErr := os.Remove(path); rmErr != nil {
			fmt.Fprintf(stderr, "warning: cannot remove HSQLDB lock file %s: %v\n", path, rmErr)
		} else {
			fmt.Fprintf(stderr, "info: removed stale HSQLDB lock file: %s\n", path)
		}
		return nil
	})
	return nil
}

// ParseDBURL converts a postgres:// connection URL to RUNTIME_PARAMS_* env vars.
func ParseDBURL(rawURL string) ([]string, error) {
	return parseDBURL(rawURL)
}

func parseDBURL(rawURL string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DB URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "postgres" && scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported DB scheme %q (only postgres:// is supported)", u.Scheme)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	username := u.User.Username()
	password, _ := u.User.Password()

	jdbcURL := fmt.Sprintf("jdbc:postgresql://%s:%s/%s", host, port, dbName)

	return []string{
		"RUNTIME_PARAMS_DATABASETYPE=PostgreSQL",
		"RUNTIME_PARAMS_DATABASEJDBCURL=" + jdbcURL,
		"RUNTIME_PARAMS_DATABASEUSERNAME=" + username,
		"RUNTIME_PARAMS_DATABASEPASSWORD=" + password,
	}, nil
}
