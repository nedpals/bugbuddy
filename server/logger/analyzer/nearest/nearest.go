package nearest

import (
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

const MAX_CLOSEST_FILE_DISTANCE = 6

func FilenameNearest(filePath string, indices map[string]int, filenames []string) string {
	if _, ok := indices[filePath]; !ok {
		// check if the filePath is already an alias
		found := fuzzy.RankFindNormalizedFold(filePath, filenames)

		// if the file path is not found, return the file path
		if len(found) == 0 {
			return filePath
		}

		// find the closest file name first before adding the value
		foundPath := found[0].Target
		distance := found[0].Distance

		if distance <= MAX_CLOSEST_FILE_DISTANCE && (strings.HasPrefix(foundPath, filePath) || strings.HasPrefix(filePath, foundPath)) {
			return foundPath
		}
	}

	return filePath
}
