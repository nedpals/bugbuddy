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
		customRunnerCommandsStr := map[string]any{}

		// parse to json
		err = json.Unmarshal(contents, &customRunnerCommandsStr)
		if err != nil {
			return customRunnerCommands, err
		}

		// convert to RunCommand
		for k, v := range customRunnerCommandsStr {
			commands := []string{}

			// allow both string and []string for flexibility
			switch v.(type) {
			case string:
				commands = []string{v.(string)}
			case []any:
				commands := []string{}
				for _, cmd := range v.([]any) {
					if cmdStr, ok := cmd.(string); ok {
						commands = append(commands, cmdStr)
					}
				}
			}

			customRunnerCommands[k] = RunCommand{Universal: commands}
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
	if runtime.GOOS == "windows" && len(runCommandList.Windows) != 0 {
		runCommand = runCommandList.Windows
	} else if len(runCommandList.Unix) != 0 {
		runCommand = runCommandList.Unix
	}

	if len(runCommand) == 0 {
		return "", fmt.Errorf("no run command for language id %s", languageId)
	}

	// replace the named placeholders
	r := strings.NewReplacer(
		"${file}", filePath,
		"${filename}", filepath.Base(filePath),
		"${filenameNoExt}", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		"${dir}", filepath.Dir(filePath),
		"${fileNoExt}", strings.TrimSuffix(filePath, filepath.Ext(filePath)),
	)

	runCommandStr := r.Replace(strings.Join(runCommand, " && "))
	if strings.Count(runCommandStr, "||") > 0 || strings.Count(runCommandStr, "&&") > 0 {
		// wrap the command in double quotes if it contains logical operators
		runCommandStr = fmt.Sprintf("\"%s\"", runCommandStr)
	}

	return fmt.Sprintf("%s -- %s", executablePath, runCommandStr), nil
}
