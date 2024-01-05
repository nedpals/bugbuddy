package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/bugbuddy/server/lsp_server"
	"github.com/nedpals/errgoengine"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bugbuddy",
	Short: "BugBuddy is a runtime error analyzer and assistant.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			log.Fatalln(err)
		}
		daemon.SetDefaultPort(fmt.Sprintf("%d", port))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("you must specify a program to run")
		}

		err := daemon.Execute(types.MonitorClientType, func(client *daemon.Client) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			numErrors, errCode, err := monitorProcess(wd, client, args[0], args[1:]...)
			if err != nil {
				return err
			} else if errCode > 0 {
				os.Stderr.WriteString(fmt.Sprintf("\n\nCatched %d error/s.\n", numErrors))
				os.Exit(errCode)
			}
			return nil
		})

		if err != nil {
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
		return daemon.Serve(daemon.CurrentPort())
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

		_, result := engine.Translate(template, data)
		fmt.Println(result)

		return nil
	},
}

var participantIdCmd = &cobra.Command{
	Use:   "participant-id",
	Short: "Returns the participant ID of the daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := daemon.Execute(types.MonitorClientType, func(client *daemon.Client) error {
			isGenerate, err := cmd.Flags().GetBool("generate")
			if err != nil {
				return err
			}

			var participantId string

			if isGenerate {
				if participantId, err = client.GenerateParticipantId(); err != nil {
					return err
				}
			} else if participantId, err = client.RetrieveParticipantId(); err != nil {
				return err
			}

			fmt.Println(participantId)
			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}

		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Resets the daemon's database",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := daemon.Execute(types.MonitorClientType, func(client *daemon.Client) error {
			if err := client.ResetLogger(); err != nil {
				return err
			}
			_, err := client.GenerateParticipantId()
			if err == nil {
				fmt.Println("ok")
			}
			return err
		})
		if err != nil {
			log.Fatalln(err)
		}
		return nil
	},
}

var runCommandCmd = &cobra.Command{
	Use:   "run-command [language-id] [file-path]",
	Short: "Returns the run command for a specific language",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		languageId := args[0]
		path := args[1]
		runCmd, err := helpers.GetRunCommand(languageId, path)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(runCmd)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(participantIdCmd)
	rootCmd.AddCommand(runCommandCmd)
	participantIdCmd.PersistentFlags().Bool("generate", false, "generate a new participant ID")
	rootCmd.AddCommand(resetCmd)
	rootCmd.PersistentFlags().IntP("port", "p", daemon.DEFAULT_PORT, "the port to use for the daemon")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
