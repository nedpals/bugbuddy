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

	"github.com/lithammer/fuzzysearch/fuzzy"
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
	"github.com/nedpals/bugbuddy/server/runner"
	"github.com/nedpals/errgoengine"
	"github.com/spf13/cobra"
	"github.com/tealeg/xlsx"
	"golang.org/x/exp/maps"
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
		runCmd, err := runner.GetCommand(languageId, path)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(runCmd)
		return nil
	},
}

type analyzerResult map[string]*analyzerResultEntry

func (a analyzerResult) Write(name string, pid string, filePath string, value any) {
	if _, ok := a[pid]; !ok {
		a[pid] = &analyzerResultEntry{
			ParticipantId:    pid,
			FilenameAliases:  map[string]string{},
			FilenamesIndices: map[string]int{},
			Filenames:        []string{},

			ErrorQuotient:        map[int]float64{},
			RepeatedErrorDensity: map[int]float64{},
			TimeToSolve:          map[int]time.Duration{},
		}
	}

	a[pid].Write(name, filePath, value)
}

type analyzerResultEntry struct {
	ParticipantId    string
	FilenameAliases  map[string]string
	FilenamesIndices map[string]int
	Filenames        []string

	// map[index of file]type
	ErrorQuotient        map[int]float64
	RepeatedErrorDensity map[int]float64
	TimeToSolve          map[int]time.Duration
}

func (a *analyzerResultEntry) Write(name string, filePath string, value any) {
	if _, ok := a.FilenamesIndices[filePath]; !ok {
		// check if the filePath is already an alias
		if alias, ok := a.FilenameAliases[filePath]; ok {
			filePath = alias
		} else if found := fuzzy.RankFindFold(filePath, a.Filenames); len(found) != 0 {
			// find the closest file name first before adding the value
			foundPath := found[0].Target

			// check if the found path is a prefix of the file path
			if found[0].Distance <= 5 && len(filePath) > len(foundPath) && strings.HasPrefix(filePath, foundPath) {
				// if it is, replace the found path with the file path
				a.FilenameAliases[foundPath] = filePath
				a.FilenamesIndices[filePath] = a.FilenamesIndices[foundPath]
				delete(a.FilenamesIndices, foundPath)

				// replace the found path with the file path
				a.Filenames[a.FilenamesIndices[filePath]] = filePath
			} else {
				// if it is not, add the file path
				a.FilenamesIndices[filePath] = len(a.Filenames)
				a.Filenames = append(a.Filenames, filePath)
			}
		} else {
			// if it is not, add the file path
			a.FilenamesIndices[filePath] = len(a.Filenames)
			a.Filenames = append(a.Filenames, filePath)
		}
	}

	// fmt.Printf("error_quotient: Merging %s into %s\n", filePath, found[0].Target)
	index := a.FilenamesIndices[filePath]

	switch name {
	case errorquotient.KEY:
		a.ErrorQuotient[index] = value.(float64)
	case red.KEY:
		a.RepeatedErrorDensity[index] = value.(float64)
	case timetosolve.KEY:
		a.TimeToSolve[index] = value.(time.Duration)
	}
}

var supportedAnalyzers = map[string]log_analyzer.LoggerAnalyzer{
	"eq":  log_analyzer.New[*errorquotient.Analyzer](),
	"red": log_analyzer.New[*red.Analyzer](),
	"tts": log_analyzer.New[*timetosolve.Analyzer](),
}

var analyzerCellNames = map[string]string{
	"eq":  "Error Quotient",
	"red": "Repeated Error Density",
	"tts": "Time To Solve",
}

var analyzeLogCmd = &cobra.Command{
	Use:   "analyze-log",
	Short: "Analyzes a set of log files. The results will be saved to an excel file.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selectedAnalyzers, _ := cmd.Flags().GetStringSlice("metrics")

		for _, analyzerName := range selectedAnalyzers {
			if _, ok := supportedAnalyzers[analyzerName]; !ok {
				return fmt.Errorf("invalid analyzer: %s. only %s were allowed", analyzerName, strings.Join(maps.Keys(supportedAnalyzers), ", "))
			}
		}

		loggerLoaders := []log_analyzer.LoggerLoader{}
		outputPath, _ := cmd.Flags().GetString("output")

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
				} else if filepath.Ext(match) != ".db" {
					log.Fatalln("only .db files are supported")
				}

				realPath, err := filepath.Abs(match)
				if err != nil {
					log.Fatalln(err)
				}

				loggerLoaders = append(loggerLoaders, func() (*logger.Logger, error) {
					return logger.NewLoggerFromPath(realPath)
				})

				log.Println("loaded", realPath)
			}
		}

		results := analyzerResult{}

		for _, lgLoader := range loggerLoaders {
			lg, err := lgLoader()
			if err != nil {
				log.Fatalln(err)
			}

			loader := func() (*logger.Logger, error) {
				return lg, nil
			}

			for _, analyzerName := range selectedAnalyzers {
				analyzer := supportedAnalyzers[analyzerName]
				if err := analyzer.Analyze(results, loader); err != nil {
					log.Fatalf("error(%T): %s", analyzer, err)
				}
			}

			if err := lg.Close(); err != nil {
				log.Fatalln(err)
			}
		}

		// save the results to an excel file
		wb := xlsx.NewFile()

		for participantId, result := range results {
			sheet, err := wb.AddSheet(participantId)
			if err != nil {
				log.Fatalln(err)
			}

			// write the header
			row := sheet.AddRow()
			row.AddCell().SetValue("File Path")

			// analyzer locations
			analyzerCellLocations := map[string]int{}

			for aCellRow, analyzerName := range selectedAnalyzers {
				name := analyzerCellNames[analyzerName]
				aCell := row.AddCell()
				aCell.SetValue(name)
				analyzerCellLocations[analyzerName] = aCellRow + 1
			}

			for fileIdx, filePath := range result.Filenames {
				row := sheet.Row(fileIdx + 1)
				row.AddCell().SetValue(filePath)

				for _, analyzerName := range selectedAnalyzers {
					cell := sheet.Cell(fileIdx+1, analyzerCellLocations[analyzerName])

					switch analyzerName {
					case "eq":
						cell.SetValue(result.ErrorQuotient[fileIdx])
					case "red":
						cell.SetValue(result.RepeatedErrorDensity[fileIdx])
					case "tts":
						cell.SetValue(formatDuration(result.TimeToSolve[fileIdx]))
					}
				}
			}
		}

		if err := wb.Save(outputPath); err != nil {
			log.Fatalln(err)
		}

		return nil
	},
}

func formatDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
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
	analyzeLogCmd.PersistentFlags().StringP("output", "o", "results.xlsx", "the output file to save the results")
	analyzeLogCmd.PersistentFlags().StringSliceP("metrics", "m", []string{"eq", "red", "tts"}, "the analyzers to use")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
