// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Default ports used by the Mendix runtime inside the container.
// These are fixed regardless of any port offset applied to host-side mappings.
const (
	containerAdminPort = 8090
	containerAppPort   = 8080
)

// M2EEOptions configures connection to the M2EE admin API.
type M2EEOptions struct {
	// Host is the hostname of the Mendix admin API (default: localhost).
	Host string

	// Port is the admin API port (default: 8090).
	Port int

	// Token is the M2EE admin password for authentication.
	Token string

	// ProjectPath is the path to the .mpr file (used to find .docker/.env).
	ProjectPath string

	// Direct bypasses docker exec and connects to the admin API directly.
	Direct bool

	// Timeout is the HTTP client timeout (default: 10s).
	Timeout time.Duration
}

// M2EEResponse is the parsed envelope from the admin API.
type M2EEResponse struct {
	Result      int             `json:"result"`
	Cause       string          `json:"cause"`
	Message     string          `json:"message"`
	RawFeedback json.RawMessage `json:"feedback"`
}

// M2EECaller abstracts the transport for Mendix admin API calls.
type M2EECaller interface {
	Call(action string, params map[string]any) (*M2EEResponse, error)
}

// DirectM2EECaller sends M2EE requests directly via HTTP.
// Zero values: Host → "localhost", Port → 8090, Timeout → 10s.
type DirectM2EECaller struct {
	Host    string
	Port    int
	Token   string
	Timeout time.Duration
}

func (d *DirectM2EECaller) host() string {
	if d.Host == "" {
		return "localhost"
	}
	return d.Host
}

func (d *DirectM2EECaller) port() int {
	if d.Port == 0 {
		return 8090
	}
	return d.Port
}

func (d *DirectM2EECaller) Call(action string, params map[string]any) (*M2EEResponse, error) {
	timeout := d.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	opts := M2EEOptions{
		Host:    d.host(),
		Port:    d.port(),
		Token:   d.Token,
		Timeout: timeout,
		Direct:  true,
	}
	return callM2EEDirect(opts, action, params)
}

// DockerExecM2EECaller sends M2EE requests via docker compose exec.
// Zero value Port → containerAdminPort (8090).
type DockerExecM2EECaller struct {
	DockerDir string
	Token     string
	Port      int
}

func (d *DockerExecM2EECaller) Call(action string, params map[string]any) (*M2EEResponse, error) {
	port := d.Port
	if port == 0 {
		port = containerAdminPort
	}
	opts := M2EEOptions{Token: d.Token, Port: port}
	return callM2EEViaDocker(opts, d.DockerDir, action, params)
}

// NewDockerExecM2EECaller creates a DockerExecM2EECaller with token resolved from:
//  1. explicitToken (if non-empty)
//  2. M2EE_ADMIN_PASS environment variable
//  3. M2EE_ADMIN_PASS in dockerDir/.env file
//
// Returns an error if no token is found.
func NewDockerExecM2EECaller(dockerDir, explicitToken string) (*DockerExecM2EECaller, error) {
	token := explicitToken
	if token == "" {
		token = os.Getenv("M2EE_ADMIN_PASS")
	}
	if token == "" {
		envPath := filepath.Join(dockerDir, ".env")
		if parsed, err := parseEnvFile(envPath); err == nil {
			token = parsed["M2EE_ADMIN_PASS"]
		}
	}
	if token == "" {
		return nil, fmt.Errorf("admin password required: set --token, M2EE_ADMIN_PASS env var, or configure .docker/.env")
	}
	return &DockerExecM2EECaller{DockerDir: dockerDir, Token: token, Port: containerAdminPort}, nil
}

// Feedback decodes the raw feedback JSON into a map.
// Returns nil map if feedback is empty or not a JSON object.
func (r *M2EEResponse) Feedback() map[string]any {
	if len(r.RawFeedback) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(r.RawFeedback, &m); err != nil {
		return nil
	}
	return m
}

// CallM2EE sends an action to the admin API and returns the parsed response.
// It supports both direct HTTP and docker-exec transport modes.
func CallM2EE(opts M2EEOptions, action string, params map[string]any) (*M2EEResponse, error) {
	if err := resolveM2EEDefaults(&opts); err != nil {
		return nil, err
	}

	// Use docker exec mode when we have a project path and Direct is not forced
	if !opts.Direct && opts.ProjectPath != "" {
		dockerDir := filepath.Join(filepath.Dir(opts.ProjectPath), ".docker")
		composePath := filepath.Join(dockerDir, "docker-compose.yml")
		if _, err := os.Stat(composePath); err == nil {
			return callM2EEViaDocker(opts, dockerDir, action, params)
		}
	}

	return callM2EEDirect(opts, action, params)
}

// callM2EEDirect sends the request via direct HTTP to the admin API.
func callM2EEDirect(opts M2EEOptions, action string, params map[string]any) (*M2EEResponse, error) {
	url := fmt.Sprintf("http://%s:%d/", opts.Host, opts.Port)

	body := map[string]any{
		"action": action,
	}
	if params != nil {
		body["params"] = params
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-M2EE-Authentication", m2eeAuthHeader(opts.Token))

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Mendix admin API at %s:%d -- is the app running? Start with 'mxcli docker up'", opts.Host, opts.Port)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("authentication failed (HTTP %d) -- check the admin password (--token or M2EE_ADMIN_PASS)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d from admin API", resp.StatusCode)
	}

	var m2eeResp M2EEResponse
	if err := json.NewDecoder(resp.Body).Decode(&m2eeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &m2eeResp, nil
}

// callM2EEViaDocker runs the request inside the Mendix container using
// "docker compose exec". This bypasses DinD network issues where the admin API
// (port 8090) binds to 127.0.0.1 inside the container and is unreachable from
// the devcontainer host.
func callM2EEViaDocker(opts M2EEOptions, dockerDir string, action string, params map[string]any) (*M2EEResponse, error) {
	body := map[string]any{
		"action": action,
	}
	if params != nil {
		body["params"] = params
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	authHeader := m2eeAuthHeader(opts.Token)

	// Use curl inside the container to hit the admin API.
	// Always use the container-internal port (8090), not opts.Port which is the
	// host-side port (may differ when port offset is applied).
	curlCmd := fmt.Sprintf(
		"curl -sf -X POST http://localhost:%d/ -H 'Content-Type: application/json' -H 'X-M2EE-Authentication: %s' -d '%s'",
		containerAdminPort, authHeader, strings.ReplaceAll(string(bodyBytes), "'", "'\\''"),
	)

	composePath := filepath.Join(dockerDir, "docker-compose.yml")
	cmd := exec.Command(ContainerCLI(), "compose", "-f", composePath, "exec", "-T", "mendix", "sh", "-c", curlCmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("docker compose exec failed: %s", stderrStr)
		}
		return nil, fmt.Errorf("cannot execute in Mendix container -- is the app running? Start with 'mxcli docker up'")
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("empty response from Mendix admin API -- is the app running?")
	}

	var m2eeResp M2EEResponse
	if err := json.NewDecoder(&stdout).Decode(&m2eeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &m2eeResp, nil
}

// resolveM2EEDefaults fills missing M2EEOptions fields from env vars and .env file.
// Priority: explicit value > environment variable > .docker/.env file > default.
func resolveM2EEDefaults(opts *M2EEOptions) error {
	// Load .env values if we can find the .docker directory
	envVals := make(map[string]string)
	if opts.ProjectPath != "" {
		dockerDir := filepath.Join(filepath.Dir(opts.ProjectPath), ".docker")
		envPath := filepath.Join(dockerDir, ".env")
		if parsed, err := parseEnvFile(envPath); err == nil {
			envVals = parsed
		}
	}

	// Host: flag > env > default
	if opts.Host == "" {
		if v := os.Getenv("MCP_OQL_MENDIX_HOST"); v != "" {
			opts.Host = v
		} else {
			opts.Host = "localhost"
		}
	}

	// Port: flag > env > .env > default
	if opts.Port == 0 {
		if v := os.Getenv("ADMIN_PORT"); v != "" {
			var port int
			if _, err := fmt.Sscanf(v, "%d", &port); err == nil && port > 0 {
				opts.Port = port
			}
		}
		if opts.Port == 0 {
			if v, ok := envVals["ADMIN_PORT"]; ok {
				var port int
				if _, err := fmt.Sscanf(v, "%d", &port); err == nil && port > 0 {
					opts.Port = port
				}
			}
		}
		if opts.Port == 0 {
			opts.Port = 8090
		}
	}

	// Token: flag > env > .env > error
	if opts.Token == "" {
		if v := os.Getenv("M2EE_ADMIN_PASS"); v != "" {
			opts.Token = v
		} else if v, ok := envVals["M2EE_ADMIN_PASS"]; ok {
			opts.Token = v
		}
	}
	if opts.Token == "" {
		return fmt.Errorf("admin password required: set --token, M2EE_ADMIN_PASS env var, or configure .docker/.env")
	}

	return nil
}

// M2EEError returns the error message from an M2EEResponse, or "" if result==0.
func (r *M2EEResponse) M2EEError() string {
	if r.Result == 0 {
		return ""
	}
	msg := r.Cause
	if msg == "" {
		msg = r.Message
	}
	if msg == "" {
		msg = "unknown error"
	}
	return msg
}

// m2eeAuthHeader returns the base64-encoded M2EE authentication token.
// The M2EE admin API expects X-M2EE-Authentication: base64(password).
func m2eeAuthHeader(password string) string {
	return base64.StdEncoding.EncodeToString([]byte(password))
}

// parseEnvFile reads a KEY=VALUE file, skipping comments and blank lines.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseEnvReader(f)
}

// parseEnvReader reads KEY=VALUE pairs from a reader.
func parseEnvReader(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first =
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}

	return result, scanner.Err()
}
