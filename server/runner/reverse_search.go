package runner

import (
	"regexp"
	"runtime"
	"strings"

	"github.com/agnivade/levenshtein"
)

func GetIdAndPathFromCommand(command string) (string, string) {
	minDist := 0
	languageID := ""
	path := ""
	matchingCommand := ""

	// Iterate over the default run commands.
	for langID, runCommand := range defaultRunCommands {
		selectedCmdList := runCommand.Universal
		if runtime.GOOS == "windows" && len(runCommand.Windows) != 0 {
			selectedCmdList = runCommand.Windows
		} else if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && len(runCommand.Unix) != 0 {
			selectedCmdList = runCommand.Unix
		}

		for _, cmd := range selectedCmdList {
			dist := levenshtein.ComputeDistance(command, cmd)

			// If the distance is less than the minimum distance, update the minimum distance, language ID, and path.
			if minDist == 0 || dist < minDist {
				minDist = dist
				languageID = langID
				matchingCommand = cmd
			}
		}
	}

	// get the last tagged from the matching command
	cmdWords := strings.Fields(matchingCommand)
	idx := len(cmdWords) - 1

	for i, word := range cmdWords {
		if strings.Contains(word, "${") {
			idx = i
		}
	}

	// from idx to the end of the command is the path for regex matching
	matchingCommand = strings.Join(cmdWords[idx:], " ")

	// Replace the tags in the command with regex groups.
	tagReplacerRegex := regexp.MustCompile(`\$\{(\w+)\}`)
	cmdReplaced := tagReplacerRegex.ReplaceAllString(matchingCommand, `(?P<$1>[^-]\S+)`) + "$"

	// Compile the regex pattern
	pattern, err := regexp.Compile(cmdReplaced)
	if err != nil {
		return languageID, ""
	}

	// Find the matches in the command
	matches := pattern.FindStringSubmatch(command)
	if len(matches) == 0 {
		return languageID, ""
	}

	// Get the path from the matches
	for i, name := range pattern.SubexpNames() {
		if name == "filename" || name == "file" {
			path = matches[i]
			break
		}
	}

	path = strings.TrimLeft(path, " ")
	return languageID, path
}
