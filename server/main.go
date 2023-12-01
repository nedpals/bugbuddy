package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/lsp_server"
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
		numErrors, errCode, err := monitorProcess(wd, daemonClient, args[0], args[1:]...)
		if err != nil {
			log.Fatalln(err)
		} else if errCode > 0 {
			os.Stderr.WriteString(fmt.Sprintf("\n\nCatched %d error/s.\n", numErrors))
			os.Exit(errCode)
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

var participantIdCmd = &cobra.Command{
	Use:   "participant-id",
	Short: "Returns the participant ID of the daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		isGenerate, err := cmd.Flags().GetBool("generate")
		if err != nil {
			log.Fatalln(err)
		}

		var participantId string
		daemonClient := daemon.Connect(daemon.DEFAULT_PORT, types.MonitorClientType)

		if isGenerate {
			if participantId, err = daemonClient.GenerateParticipantId(); err != nil {
				log.Fatalln(err)
			}
		} else if participantId, err = daemonClient.RetrieveParticipantId(); err != nil {
			log.Fatalln(err)
		}

		fmt.Println(participantId)
		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Resets the daemon's database",
	RunE: func(cmd *cobra.Command, args []string) error {
		daemonClient := daemon.Connect(daemon.DEFAULT_PORT, types.MonitorClientType)
		if err := daemonClient.ResetLogger(); err != nil {
			log.Fatalln(err)
		}

		if _, err := daemonClient.GenerateParticipantId(); err != nil {
			log.Fatalln(err)
		}

		fmt.Println("ok")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(participantIdCmd)
	participantIdCmd.PersistentFlags().Bool("generate", false, "generate a new participant ID")
	rootCmd.AddCommand(resetCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
