package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/nedpals/bugbuddy-proto/server/daemon"
	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/lsp_server"
	"github.com/nedpals/errgoengine"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bugbuddy",
	Short: "BugBuddy is a runtime error analyzer and assistant.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("you must specify a program to run")
		}

		wd, err := os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}

		daemonClient := daemon.Connect(daemon.DEFAULT_PORT, types.MonitorClientType)
		if err := monitorProcess(wd, daemonClient, args[0], args[1:]...); err != nil {
			log.Fatalln(err)
		}
		return nil
	},
}

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Starts a language server to be consumed by LSP-supported editors",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lsp_server.Start()
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
		var errMsg string
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			if len(errMsg) != 0 {
				errMsg += "\n"
			}

			errMsg += scanner.Text()
		}

		if len(errMsg) == 0 {
			os.Exit(1)
		}

		engine := errgoengine.New()
		template, data, err := engine.Analyze(wd, errMsg)
		if err != nil {
			log.Fatalln(err)
		}

		result := engine.Translate(template, data)
		fmt.Println(result)

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
