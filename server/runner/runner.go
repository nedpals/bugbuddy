package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nedpals/bugbuddy/server/helpers"
)

func getJsonConfig() (map[string]RunCommand, error) {
	// where the commands stored in runner.json will be stored
	customRunnerCommands := map[string]RunCommand{}

	dirPath, err := helpers.GetOrInitializeDataDir()
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

func GetCommand(languageId string, filePath string) (string, error) {
	// get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	customRunCommands, err := getJsonConfig()
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
