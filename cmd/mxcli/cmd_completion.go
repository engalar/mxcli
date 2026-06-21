// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Output shell completion code for the specified shell.

The completion code must be sourced to take effect.

  bash:
    source <(mxcli completion bash)
    # or:
    mxcli completion bash > /etc/bash_completion.d/mxcli

  zsh:
    source <(mxcli completion zsh)
    # or:
    mxcli completion zsh > /usr/local/share/zsh/site-functions/_mxcli

  fish:
    mxcli completion fish > ~/.config/fish/completions/mxcli.fish

  powershell:
    mxcli completion powershell > mxcli.ps1
    # then in your PowerShell profile:
    . ./mxcli.ps1
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	ValidArgs:     []string{"bash", "zsh", "fish", "powershell"},
	Args:          cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
		return nil
	},
}
