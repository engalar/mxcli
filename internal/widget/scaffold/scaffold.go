package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

// Run renders all files from the given renderers and writes them to dir.
func Run(dir string, spec Spec, renderers []Renderer) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	for _, r := range renderers {
		files := r.Render(spec)
		for _, f := range files {
			dest := filepath.Join(dir, f.Path)
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("create dir for %s: %w", f.Path, err)
			}
			perm := os.FileMode(0644)
			if f.Binary {
				perm = 0644
			}
			if err := os.WriteFile(dest, f.Content, perm); err != nil {
				return fmt.Errorf("write %s: %w", f.Path, err)
			}
		}
	}
	return nil
}
