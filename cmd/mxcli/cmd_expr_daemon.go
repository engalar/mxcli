// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/spf13/cobra"
)

var exprDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage expr daemon background processes",
}

var exprDaemonStartSocket string

var exprDaemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon (usually invoked implicitly by mxcli expr validate)",
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Root().PersistentFlags().GetString("project")
		if mprPath == "" {
			return fmt.Errorf("requires -p project.mpr")
		}

		idleTimeout := 5 * time.Minute
		if s := os.Getenv("MXCLI_DAEMON_TIMEOUT"); s != "" {
			if d, err := time.ParseDuration(s); err == nil {
				idleTimeout = d
			}
		}
		d, err := daemon.NewWithSocket(mprPath, exprDaemonStartSocket, idleTimeout)
		if err != nil {
			return fmt.Errorf("init daemon: %w", err)
		}
		return d.Serve()
	},
}

var exprDaemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List all running expr daemon statuses",
	RunE: func(cmd *cobra.Command, args []string) error {
		statuses, err := daemon.ListRunning()
		if err != nil {
			return err
		}
		if len(statuses) == 0 {
			fmt.Println("No running expr daemons.")
			return nil
		}
		for _, s := range statuses {
			fmt.Printf("● %s  age %s\n  entities: %d  enums: %d\n",
				s.MprPath, s.IndexAge, s.EntityCount, s.EnumCount)
		}
		return nil
	},
}

var exprDaemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop daemon for specified MPR",
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Root().PersistentFlags().GetString("project")
		if mprPath == "" {
			return fmt.Errorf("requires -p project.mpr")
		}
		sp := daemon.SocketPath(mprPath)
		if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Printf("Stopped daemon: %s\n", mprPath)
		return nil
	},
}

func init() {
	exprDaemonStartCmd.Flags().StringVar(&exprDaemonStartSocket, "socket", "",
		"Socket file path (auto-computed by default)")
	exprDaemonCmd.AddCommand(exprDaemonStartCmd, exprDaemonStatusCmd, exprDaemonStopCmd)
	exprCmd.AddCommand(exprDaemonCmd)
}
