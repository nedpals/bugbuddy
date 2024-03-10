package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nedpals/bugbuddy/server/daemon"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/executor"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/bugbuddy/server/logger"
	log_analyzer "github.com/nedpals/bugbuddy/server/logger/analyzer"
	errorquotient "github.com/nedpals/bugbuddy/server/logger/analyzer/error_quotient"
	red "github.com/nedpals/bugbuddy/server/logger/analyzer/repeated_error_density"
	timetosolve "github.com/nedpals/bugbuddy/server/logger/analyzer/time_to_solve"
	"github.com/nedpals/bugbuddy/server/lsp_server"
	"github.com/nedpals/bugbuddy/server/release"
	"github.com/nedpals/errgoengine"
	"github.com/spf13/cobra"
	"github.com/tealeg/xlsx"
)

var rootCmd = &cobra.Command{
	Use:     "bugbuddy",
	Version: release.Version(),
	Short:   "BugBuddy is a runtime error analyzer and assistant.",
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

		var writer io.Writer = io.Discard
		if isVerbose, _ := cmd.Flags().GetBool("verbose"); isVerbose {
			writer = os.Stderr
		}

		err := daemon.Execute(types.MonitorClientType, func(client *daemon.Client) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			fmt.Printf("bugbuddy> listening to %s %s...\n", args[0], strings.Join(args[1:], " "))
			collector := &executor.ClientCollector{
				Logger: log.New(writer, "bugbuddy>", 0),
				Client: client,
			}
			numErrors, errCode, err := executor.Execute(wd, collector, args[0], args[1:]...)
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
		// change data-dir if present
		if dataDir, _ := cmd.Flags().GetString("data-dir"); len(dataDir) != 0 {
			helpers.SetDataDirPath(dataDir)
		}

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

type analyzeResults struct {
	FilePath             string
	ErrorQuotient        float64
	RepeatedErrorDensity float64
	TimeToSolve          time.Duration
}

var analyzeLogCmd = &cobra.Command{
	Use:   "analyze-log",
	Short: "Analyzes a set of log files. The results will be saved to an excel file.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		loggerLoaders := []log_analyzer.LoggerLoader{}

		for _, path := range args {
			matches, err := filepath.Glob(path)
			if err != nil {
				log.Fatalln(err)
			}

			for _, match := range matches {
				// check if match is a directory
				fi, err := os.Stat(match)
				if err != nil {
					log.Fatalln(err)
				}

				if fi.IsDir() {
					log.Fatalln("directories are not supported")
				}

				loggerLoaders = append(loggerLoaders, func() (*logger.Logger, error) {
					return logger.NewLoggerFromPath(match)
				})

				log.Println("loaded", match)
			}
		}

		// TODO: set as configurable flags
		analyzers := []string{"eq", "red", "tts"}

		// map[analyzer]map[participantId]map[filePath]struct{FilePath string, ErrorQuotient float64, RepeatedErrorDensity float64, TimeToSolve time.Duration}
		results := map[string]map[string]*analyzeResults{}

		for _, analyzer := range analyzers {
			switch analyzer {
			case "eq":
				eqa := log_analyzer.New[errorquotient.Analyzer](loggerLoaders...)
				if err := eqa.Analyze(); err != nil {
					log.Fatalln(err)
				}

				for participantId, filePaths := range eqa.ResultsByParticipant {
					if _, ok := results[participantId]; !ok {
						results[participantId] = map[string]*analyzeResults{}
					}

					for filePath, eq := range filePaths {
						if _, ok := results[participantId][filePath]; !ok {
							results[participantId][filePath] = &analyzeResults{FilePath: filePath}
						}

						results[participantId][filePath].ErrorQuotient = eq
					}
				}
			case "red":
				red := log_analyzer.New[red.Analyzer](loggerLoaders...)
				if err := red.Analyze(); err != nil {
					log.Fatalln(err)
				}

				for participantId, filePaths := range red.ResultsByParticipant {
					if _, ok := results[participantId]; !ok {
						results[participantId] = map[string]*analyzeResults{}
					}

					for filePath, red := range filePaths {
						if _, ok := results[participantId][filePath]; !ok {
							results[participantId][filePath] = &analyzeResults{FilePath: filePath}
						}

						results[participantId][filePath].RepeatedErrorDensity = red
					}
				}
			case "tts":
				tts := log_analyzer.New[timetosolve.Analyzer](loggerLoaders...)
				if err := tts.Analyze(); err != nil {
					log.Fatalln(err)
				}

				for participantId, filePaths := range tts.ResultsByParticipant {
					if _, ok := results[participantId]; !ok {
						results[participantId] = map[string]*analyzeResults{}
					}

					for filePath, tts := range filePaths {
						if _, ok := results[participantId][filePath]; !ok {
							results[participantId][filePath] = &analyzeResults{FilePath: filePath}
						}

						results[participantId][filePath].TimeToSolve = tts
					}
				}
			}
		}

		// save the results to an excel file
		wb := xlsx.NewFile()

		for participantId, filePaths := range results {
			sheet, err := wb.AddSheet(participantId)
			if err != nil {
				log.Fatalln(err)
			}

			// write the header
			row := sheet.AddRow()
			row.AddCell().SetValue("File Path")
			row.AddCell().SetValue("Error Quotient")
			row.AddCell().SetValue("Repeated Error Density")
			row.AddCell().SetValue("Time To Solve")

			for _, result := range filePaths {
				row = sheet.AddRow()
				row.AddCell().SetValue(result.FilePath)
				row.AddCell().SetValue(result.ErrorQuotient)
				row.AddCell().SetValue(result.RepeatedErrorDensity)
				row.AddCell().SetValue(result.TimeToSolve.String())
			}
		}

		// TODO: set as configurable flag
		if err := wb.Save("results.xlsx"); err != nil {
			log.Fatalln(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(participantIdCmd)
	rootCmd.AddCommand(runCommandCmd)
	rootCmd.AddCommand(analyzeLogCmd)
	participantIdCmd.PersistentFlags().Bool("generate", false, "generate a new participant ID")
	rootCmd.AddCommand(resetCmd)
	rootCmd.PersistentFlags().IntP("port", "p", daemon.DEFAULT_PORT, "the port to use for the daemon")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose mode")
	daemonCmd.PersistentFlags().String("data-dir", "", "the directory to use for the daemon. To override the default directory, set the BUGBUDDY_DIR environment variable.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
