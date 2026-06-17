package build

import (
	"context"
)

// Builder builds a widget project into an .mpk file.
type Builder interface {
	// Name returns the builder name for display.
	Name() string
	// Available returns true if this builder's toolchain is installed.
	Available() bool
	// Build runs the build and returns the path to the generated .mpk.
	Build(ctx context.Context, projectDir string) (mpkPath string, err error)
}
