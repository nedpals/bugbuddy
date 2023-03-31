package main

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bugbuddy",
	Short: "BugBuddy is a runtime error analyzer and assistant.",
	Run: func(cmd *cobra.Command, args []string) {
		daemonClient := connectToDaemon(DEFAULT_DAEMON_PORT)
		if err := monitorProcess(daemonClient, args[0], args[1:]...); err != nil {
			log.Fatalln(err)
		}
	},
}

var lspCmd = &cobra.Command{
	Use:   "lsp-server",
	Short: "Starts a language server to be consumed by LSP-supported editors",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startLspServer(":3333")
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Starts a daemon process to collect error messages from programs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startDaemon(DEFAULT_DAEMON_PORT)
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	rootCmd.AddCommand(daemonCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
