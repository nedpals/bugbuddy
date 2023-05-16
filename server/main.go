package main

import (
	"fmt"
	"log"

	"github.com/nedpals/bugbuddy-proto/server/analysis"
	"github.com/nedpals/bugbuddy-proto/server/daemon"
	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/lsp_server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bugbuddy",
	Short: "BugBuddy is a runtime error analyzer and assistant.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("you must specify a program to run")
		}

		daemonClient := daemon.Connect(daemon.DEFAULT_PORT, types.MonitorClientType)
		if err := monitorProcess(daemonClient, args[0], args[1:]...); err != nil {
			log.Fatalln(err)
		}
		return nil
	},
}

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Starts a language server to be consumed by LSP-supported editors",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lsp_server.Start(lsp_server.DEFAULT_PORT)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Starts a daemon process to collect error messages from programs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return daemon.Serve(daemon.DEFAULT_PORT)
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyzes a specific error message and returns the suggestion. For testing purposes only",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO:
		output, _ := analysis.DetectError(args[0])
		fmt.Println(output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(analyzeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
