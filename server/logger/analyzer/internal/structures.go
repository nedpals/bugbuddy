package internal

import (
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type ResultStore[T any] struct {
	FilenameAliases  map[string]string
	FilenamesIndices map[string]int
	Filenames        []string
	Values           map[int]T
}

func (r *ResultStore[T]) FilenameNearest(filePath string) string {
	if _, ok := r.FilenamesIndices[filePath]; !ok {
		// check if the filePath is already an alias
		found := fuzzy.RankFindNormalizedFold(filePath, r.Filenames)

		// if the file path is not found, return the file path
		if len(found) == 0 {
			return filePath
		}

		// find the closest file name first before adding the value
		foundPath := found[0].Target
		distance := found[0].Distance

		if distance <= 6 && (strings.HasPrefix(foundPath, filePath) || strings.HasPrefix(filePath, foundPath)) {
			return foundPath
		}
	}

	return filePath
}

func (r *ResultStore[T]) checkAndUpdateFilename(filePath string) string {
	filePath = strings.TrimSpace(filePath)

	if alias, ok := r.FilenameAliases[filePath]; ok {
		// do not mutate the original file path
		return alias
	}

	// nearest based on levenstein distance
	if nearest := r.FilenameNearest(filePath); nearest != filePath && strings.HasPrefix(filePath, nearest) {
		// if it is, replace the found path with the file path
		r.Filenames[r.FilenamesIndices[nearest]] = filePath
		r.FilenameAliases[nearest] = filePath
		r.FilenamesIndices[filePath] = r.FilenamesIndices[nearest]
		delete(r.FilenamesIndices, nearest)
	} else if _, ok := r.FilenamesIndices[filePath]; !ok {
		// if it is not, add the file path
		r.FilenamesIndices[filePath] = len(r.Filenames)
		r.Filenames = append(r.Filenames, filePath)
	}

	return filePath
}

func (r *ResultStore[T]) nearestImmutableFilename(filePath string) string {
	if alias, ok := r.FilenameAliases[filePath]; ok {
		return alias
	} else if nearest := r.FilenameNearest(filePath); nearest != filePath && strings.HasPrefix(nearest, filePath) {
		return nearest
	}
	return filePath
}

func (r *ResultStore[T]) Set(filePath string, value T) {
	filePath = r.checkAndUpdateFilename(filePath)
	// fmt.Printf("Setting %s to %v\n", filePath, value)
	r.Values[r.FilenamesIndices[filePath]] = value
}

func (r *ResultStore[T]) Get(filePath string) T {
	filePath = r.nearestImmutableFilename(filePath)
	index, ok := r.FilenamesIndices[filePath]
	if !ok {
		index = -1
	}
	return r.Values[index]
}

func (r *ResultStore[T]) Has(filePath string) bool {
	nearest := r.nearestImmutableFilename(filePath)
	if nearest != filePath {
		return true
	}
	_, ok := r.FilenamesIndices[filePath]
	return ok
}

func (r *ResultStore[T]) GetOr(filePath string, defVal T) T {
	filePath = r.nearestImmutableFilename(filePath)
	index, ok := r.FilenamesIndices[filePath]
	if !ok {
		return defVal
	}
	return r.Values[index]
}

func NewResultStore[T any]() *ResultStore[T] {
	return &ResultStore[T]{
		FilenameAliases:  make(map[string]string),
		FilenamesIndices: make(map[string]int),
		Filenames:        []string{},
		Values:           make(map[int]T),
	}
}
