// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"os/exec"
)

type Installer struct {
	mxPath string
}

func NewInstaller(mxPath string) *Installer {
	return &Installer{mxPath: mxPath}
}

func (inst *Installer) InstallModule(ctx context.Context, mpkPath, projectPath string) error {
	cmd := exec.CommandContext(ctx, inst.mxPath, "module-import", mpkPath, projectPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mx module-import failed: %w\n%s", err, string(out))
	}
	return nil
}
