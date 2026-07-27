// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SCSSWatcherConfig configures the SCSS file watcher.
type SCSSWatcherConfig struct {
	// DeployDir is the deployment/ directory (contains sass/main.scss).
	DeployDir string

	// ProjectDir is the project root directory.
	ProjectDir string

	// SassPath is the path to the Dart Sass binary.
	SassPath string

	// AdminHost is the M2EE admin API host.
	AdminHost string

	// AdminPort is the M2EE admin API port.
	AdminPort int

	// AdminPassword is the M2EE admin password.
	AdminPassword string

	// Stdout for output messages.
	Stdout io.Writer
}

// SCSSWatcher watches SCSS files and auto-recompiles on change.
type SCSSWatcher struct {
	config  SCSSWatcherConfig
	watcher *fsnotify.Watcher
	ready   bool
	done    chan struct{}
}

// NewSCSSWatcher creates a new SCSS watcher.
func NewSCSSWatcher(cfg SCSSWatcherConfig) (*SCSSWatcher, error) {
	sassEntry := filepath.Join(cfg.DeployDir, "sass", "main.scss")
	if _, err := os.Stat(sassEntry); err != nil {
		return nil, fmt.Errorf("SCSS entry point not found at %s (run mxcli build first)", sassEntry)
	}
	themeCSS := filepath.Join(cfg.DeployDir, "web", "theme.compiled.css")
	if _, err := os.Stat(themeCSS); err != nil {
		return nil, fmt.Errorf("theme output not found at %s (run mxcli build first)", themeCSS)
	}
	if cfg.SassPath == "" {
		sassPath, err := resolveSassPath()
		if err != nil {
			return nil, fmt.Errorf("Dart Sass not found: %w", err)
		}
		cfg.SassPath = sassPath
	}
	w := &SCSSWatcher{
		config: cfg,
		done:   make(chan struct{}),
	}
	return w, nil
}

// Start begins watching SCSS files. Blocks until Stop is called or an error occurs.
func (w *SCSSWatcher) Start() error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	w.watcher = fw
	defer fw.Close()

	dirs := w.scssDirs()
	for _, d := range dirs {
		if err := fw.Add(d); err != nil {
			fmt.Fprintf(w.config.Stdout, "[scss] Warning: cannot watch %s: %v\n", d, err)
		} else {
			fmt.Fprintf(w.config.Stdout, "[scss] Watching %s\n", d)
		}
	}

	w.ready = true

	var debounceTimer *time.Timer
	var debounceCh chan time.Time

	for {
		select {
		case event, ok := <-fw.Events:
			if !ok {
				return nil
			}
			if !strings.HasSuffix(event.Name, ".scss") {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			if debounceTimer == nil {
				debounceCh = make(chan time.Time, 1)
				debounceTimer = time.NewTimer(500 * time.Millisecond)
				go func() {
					<-debounceTimer.C
					debounceCh <- time.Now()
				}()
			} else {
				debounceTimer.Reset(500 * time.Millisecond)
			}

		case <-debounceCh:
			debounceTimer = nil
			debounceCh = nil
			w.compileAndReload()

		case err, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(w.config.Stdout, "[scss] Watch error: %v\n", err)
		}
	}
}

// Stop signals the watcher to stop.
func (w *SCSSWatcher) Stop() {
	if w.watcher != nil {
		w.watcher.Close()
	}
}

func (w *SCSSWatcher) scssDirs() []string {
	seen := map[string]bool{}
	var dirs []string

	candidates := []string{
		filepath.Join(w.config.ProjectDir, "theme", "web"),
	}
	themesource := filepath.Join(w.config.ProjectDir, "themesource")
	entries, err := os.ReadDir(themesource)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				d := filepath.Join(themesource, e.Name(), "web")
				if info, statErr := os.Stat(d); statErr == nil && info.IsDir() {
					candidates = append(candidates, d)
				}
			}
		}
	}

	for _, d := range candidates {
		p, err := filepath.EvalSymlinks(d)
		if err != nil {
			p = d
		}
		if !seen[p] {
			seen[p] = true
			dirs = append(dirs, p)
		}
	}
	return dirs
}

func (w *SCSSWatcher) compileAndReload() {
	w.compile()
	w.reloadCSS()
}

func (w *SCSSWatcher) compile() {
	sassEntry := filepath.Join(w.config.DeployDir, "sass", "main.scss")
	cssOut := filepath.Join(w.config.DeployDir, "web", "theme.compiled.css")

	// Find the Dart Sass binary
	sassBin := w.config.SassPath
	if _, err := os.Stat(sassBin); err != nil {
		fmt.Fprintf(w.config.Stdout, "[scss] Sass binary not found at %s\n", sassBin)
		return
	}

	start := time.Now()
	cmd := exec.Command(sassBin, sassEntry+":"+cssOut)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(w.config.Stdout, "[scss] Compile error: %v\n%s\n", err, string(output))
		return
	}
	elapsed := time.Since(start)
	fmt.Fprintf(w.config.Stdout, "[scss] Recompiled theme (%v)\n", elapsed)
}

func (w *SCSSWatcher) reloadCSS() {
	caller := &DirectM2EECaller{
		Host:    w.config.AdminHost,
		Port:    w.config.AdminPort,
		Token:   w.config.AdminPassword,
		Timeout: 10 * time.Second,
	}
	resp, err := caller.Call("update_styling", nil)
	if err != nil {
		fmt.Fprintf(w.config.Stdout, "[scss] Reload error: %v\n", err)
		return
	}
	if errMsg := resp.M2EEError(); errMsg != "" {
		fmt.Fprintf(w.config.Stdout, "[scss] Reload error: %s\n", errMsg)
		return
	}
	fmt.Fprintf(w.config.Stdout, "[scss] Styling updated\n")
}

// resolveSassPath finds the Dart Sass binary in the mxbuild cache.
func resolveSassPath() (string, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".mxcli", "mxbuild")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return "", fmt.Errorf("mxbuild cache not found at %s", cacheDir)
	}

	var latestVersion string
	for _, e := range entries {
		if e.IsDir() && e.Name() > latestVersion {
			latestVersion = e.Name()
		}
	}
	if latestVersion == "" {
		return "", fmt.Errorf("no mxbuild version found in %s", cacheDir)
	}

	candidates := []string{
		filepath.Join(cacheDir, latestVersion, "modeler", "tools", "sass", "linux-x64", "sass"),
		filepath.Join(cacheDir, latestVersion, "tools", "sass", "linux-x64", "sass"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("Dart Sass not found in mxbuild %s", latestVersion)
}
