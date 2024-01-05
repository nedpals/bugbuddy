package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type RunCommand struct {
	Universal string // Command if both OS have same command. Not required if either Windows or Unix OS is specified.
	Unix      string // Command for Unix-based systems (eg. macOS, Linux). Not required if Universal is specified.
	Windows   string // Command for Windows-based systems. Not required if Universal is specified.
}

var defaultRunCommands = map[string]RunCommand{
	"python":       {Universal: "python3 ${file}"},
	"c":            {Universal: "gcc ${file} -o ${fileNoExt} && ./${fileNoExt}"},
	"cpp":          {Universal: "g++ ${file} -o ${fileNoExt} && ./${fileNoExt}"},
	"java":         {Universal: "javac ${file} && java ${fileNoExt}"},
	"rust":         {Universal: "rustc ${file} && ./${fileNoExt}"},
	"go":           {Universal: "go run ${file}"},
	"js":           {Universal: "node ${file}"},
	"typescript":   {Universal: "ts-node ${file}"},
	"php":          {Universal: "php ${file}"},
	"ruby":         {Universal: "ruby ${file}"},
	"perl":         {Universal: "perl ${file}"},
	"bash":         {Universal: "bash ${file}"},
	"sh":           {Universal: "sh ${file}"},
	"zsh":          {Universal: "zsh ${file}"},
	"powershell":   {Universal: "powershell -ExecutionPolicy Bypass -File ${file}"},
	"batch":        {Universal: "cmd /c ${file}"},
	"lua":          {Universal: "lua ${file}"},
	"r":            {Universal: "Rscript ${file}"},
	"dart":         {Universal: "dart ${file}"},
	"elixir":       {Universal: "elixir ${file}"},
	"erlang":       {Universal: "erl -noshell -s ${fileNoExt} main -s init stop"},
	"clojure":      {Universal: "clojure ${file}"},
	"julia":        {Universal: "julia ${file}"},
	"coffeescript": {Universal: "coffee ${file}"},
	"crystal":      {Universal: "crystal ${file}"},
	"nim":          {Universal: "nim c -r ${file}"},
	"ocaml":        {Universal: "ocaml ${file}"},
	"pascal":       {Universal: "fpc ${file} && ./${fileNoExt}"},
	"perl6":        {Universal: "perl6 ${file}"},
	"prolog":       {Universal: "swipl -q -t main -f ${file}"},
	"racket":       {Universal: "racket ${file}"},
	"raku":         {Universal: "raku ${file}"},
	"reason":       {Universal: "refmt ${file} && node ${fileNoExt}.js"},
	"red":          {Universal: "red ${file}"},
	"solidity":     {Universal: "solc ${file}"},
	"swift":        {Universal: "swift ${file}"},
	"v":            {Universal: "v run ${file}"},
	"vb":           {Universal: "vbnc ${file} && mono ${fileNoExt}.exe"},
	"vbnet":        {Universal: "vbnc ${file} && mono ${fileNoExt}.exe"},
	"vbs":          {Universal: "cscript ${file}"},
	"zig":          {Universal: "zig run ${file}"},
}

func initializeRunnerJson() error {
	dirPath, err := GetOrInitializeDir()
	if err != nil {
		return err
	}

	// write runner.json
	runnerJsonPath := filepath.Join(dirPath, "runner.json")
	if _, err := os.Stat(runnerJsonPath); os.IsNotExist(err) {
		runnerJsonFile, err := os.Create(runnerJsonPath)
		if err != nil {
			return err
		}
		defer runnerJsonFile.Close()

		enc := json.NewEncoder(runnerJsonFile)
		enc.SetIndent("", "    ")
		if err := enc.Encode(defaultRunCommands); err != nil {
			return err
		}
	}

	return nil
}

func GetOrInitializeRunnerJson() (map[string]RunCommand, error) {
	dirPath, err := GetOrInitializeDir()
	if err != nil {
		return nil, err
	}

	// read runner.json
	runnerJsonPath := filepath.Join(dirPath, "runner.json")
	runnerJsonFile, err := os.Open(runnerJsonPath)
	if err != nil {
		if err := initializeRunnerJson(); err != nil {
			return nil, err
		}

		// go back to attempt to open runner.json again
		return GetOrInitializeRunnerJson()
	}

	var runnerJson map[string]RunCommand
	if err := json.NewDecoder(runnerJsonFile).Decode(&runnerJson); err != nil {
		return nil, err
	}

	return runnerJson, nil
}

func GetRunCommand(languageId string, filePath string) (string, error) {
	// get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	runCommands, err := GetOrInitializeRunnerJson()
	if err != nil {
		return "", err
	}

	runCommandList, ok := runCommands[languageId]
	if !ok {
		return "", fmt.Errorf("no run command for language id %s", languageId)
	}

	runCommand := runCommandList.Universal
	if len(runCommand) == 0 {
		if runtime.GOOS == "windows" {
			runCommand = runCommandList.Windows
		} else {
			runCommand = runCommandList.Unix
		}

		if len(runCommand) == 0 {
			return "", fmt.Errorf("no run command for language id %s", languageId)
		}
	}

	// replace the named placeholders
	r := strings.NewReplacer(
		"${file}", filePath,
		"${filename}", filepath.Base(filePath),
		"${dir}", filepath.Dir(filePath),
		"${fileNoExt}", strings.TrimSuffix(filePath, filepath.Ext(filePath)),
	)

	runCommand = r.Replace(runCommand)
	return fmt.Sprintf("%s -- %s", executablePath, runCommand), nil
}
