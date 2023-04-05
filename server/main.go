package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bugbuddy",
	Short: "BugBuddy is a runtime error analyzer and assistant.",
	Run: func(cmd *cobra.Command, args []string) {
		daemonClient := connectToDaemon(DEFAULT_DAEMON_PORT, CLIENT_TYPE_MONITOR)
		if err := monitorProcess(daemonClient, args[0], args[1:]...); err != nil {
			log.Fatalln(err)
		}
	},
}

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Starts a language server to be consumed by LSP-supported editors",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startLspServer(DEFAULT_LSP_PORT)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Starts a daemon process to collect error messages from programs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startDaemon(DEFAULT_DAEMON_PORT)
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyzes a specific error message and returns the suggestion. For testing purposes only",
	RunE: func(cmd *cobra.Command, args []string) error {
		output := defaultErrorAnalyzer.analyze(args[0])
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
