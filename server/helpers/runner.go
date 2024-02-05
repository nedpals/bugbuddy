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
	"python":       {Universal: "python3 ${filename}"},
	"c":            {Universal: "gcc ${filename} -o ${fileNoExt} && ./${fileNoExt}"},
	"cpp":          {Universal: "g++ ${filename} -o ${fileNoExt} && ./${fileNoExt}"},
	"java":         {Universal: "javac ${filename} && java ${filenameNoExt}"},
	"rust":         {Universal: "rustc ${filename} && ./${fileNoExt}"},
	"go":           {Universal: "go run ${filename}"},
	"js":           {Universal: "node ${filename}"},
	"typescript":   {Universal: "ts-node ${filename}"},
	"php":          {Universal: "php ${filename}"},
	"ruby":         {Universal: "ruby ${filename}"},
	"perl":         {Universal: "perl ${filename}"},
	"bash":         {Universal: "bash ${filename}"},
	"sh":           {Universal: "sh ${filename}"},
	"zsh":          {Universal: "zsh ${filename}"},
	"powershell":   {Universal: "powershell -ExecutionPolicy Bypass -File ${filename}"},
	"batch":        {Universal: "cmd /c ${filename}"},
	"lua":          {Universal: "lua ${filename}"},
	"r":            {Universal: "Rscript ${filename}"},
	"dart":         {Universal: "dart ${filename}"},
	"elixir":       {Universal: "elixir ${filename}"},
	"erlang":       {Universal: "erl -noshell -s ${fileNoExt} main -s init stop"},
	"clojure":      {Universal: "clojure ${filename}"},
	"julia":        {Universal: "julia ${filename}"},
	"coffeescript": {Universal: "coffee ${filename}"},
	"crystal":      {Universal: "crystal ${filename}"},
	"nim":          {Universal: "nim c -r ${filename}"},
	"ocaml":        {Universal: "ocaml ${filename}"},
	"pascal":       {Universal: "fpc ${filename} && ./${fileNoExt}"},
	"perl6":        {Universal: "perl6 ${filename}"},
	"prolog":       {Universal: "swipl -q -t main -f ${filename}"},
	"racket":       {Universal: "racket ${filename}"},
	"raku":         {Universal: "raku ${filename}"},
	"reason":       {Universal: "refmt ${filename} && node ${fileNoExt}.js"},
	"red":          {Universal: "red ${filename}"},
	"solidity":     {Universal: "solc ${filename}"},
	"swift":        {Universal: "swift ${filename}"},
	"v":            {Universal: "v run ${filename}"},
	"vb":           {Universal: "vbnc ${filename} && mono ${fileNoExt}.exe"},
	"vbnet":        {Universal: "vbnc ${filename} && mono ${fileNoExt}.exe"},
	"vbs":          {Universal: "cscript ${filename}"},
	"zig":          {Universal: "zig run ${filename}"},
}

func GetRunnerJson() (map[string]RunCommand, error) {
	// where the commands stored in runner.json will be stored
	customRunnerCommands := map[string]RunCommand{}

	dirPath, err := GetOrInitializeDir()
	if err != nil {
		return customRunnerCommands, err
	}

	// write runner.json
	runnerJsonPath := filepath.Join(dirPath, "runner.json")
	if contents, err := os.ReadFile(runnerJsonPath); err == nil {
		customRunnerCommandsStr := map[string]string{}

		// parse to json
		err = json.Unmarshal(contents, &customRunnerCommandsStr)
		if err != nil {
			return customRunnerCommands, err
		}

		// convert to RunCommand
		for k, v := range customRunnerCommandsStr {
			customRunnerCommands[k] = RunCommand{Universal: v}
		}
	}

	return customRunnerCommands, err
}

func GetRunCommand(languageId string, filePath string) (string, error) {
	// get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	customRunCommands, err := GetRunnerJson()
	if err != nil {
		return "", err
	}

	runCommandList, ok := customRunCommands[languageId]
	if !ok {
		runCommandList, ok = defaultRunCommands[languageId]
		if !ok {
			return "", fmt.Errorf("no run command for language id %s", languageId)
		}
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
		"${filenameNoExt}", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		"${dir}", filepath.Dir(filePath),
		"${fileNoExt}", strings.TrimSuffix(filePath, filepath.Ext(filePath)),
	)

	runCommand = r.Replace(runCommand)
	if strings.Count(runCommand, "||") > 0 || strings.Count(runCommand, "&&") > 0 {
		// wrap the command in double quotes if it contains logical operators
		runCommand = fmt.Sprintf("\"%s\"", runCommand)
	}

	return fmt.Sprintf("%s -- %s", executablePath, runCommand), nil
}
