package analysis

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type ErrorTemplate struct {
	Name             string
	StartPattern     *regexp.Regexp
	BacktracePattern *regexp.Regexp
	RuleMatcher      func(*sitter.Tree, string, *ErrorData) string
	Description      string
}

type ErrorTemplateConfig struct {
	Name             string
	StartPattern     string
	BacktracePattern string
	RuleMatcher      func(*sitter.Tree, string, *ErrorData) string
	Description      string
}

type Location struct {
	Symbol string
	File   string
	Pos    SimplePosition
}

func (temp ErrorTemplateConfig) Compile() *ErrorTemplate {
	// TODO: error handling
	startPattern, _ := regexp.Compile("^" + temp.StartPattern)
	backtracePattern, _ := regexp.Compile(temp.BacktracePattern)

	return &ErrorTemplate{
		Name:             temp.Name,
		StartPattern:     startPattern,
		BacktracePattern: backtracePattern,
	}
}

func NewErrorTemplate(cfg ErrorTemplateConfig) *ErrorTemplate {
	return cfg.Compile()
}

type ErrorData struct {
	Content         string
	Locations       []Location
	ErrorTemplate   *ErrorTemplate
	ConnectedErrors []Location
}

func (err ErrorData) AddError(filepath string, pos SimplePosition) {
	err.ConnectedErrors = append(err.ConnectedErrors, Location{
		Symbol: "",
		File:   filepath,
		Pos:    pos,
	})
}

// leave it here for idea:
// - rule-based system
//   - emulates how a person classifies errors (first do x, check for y, and etc.)
func DetectError(errorMsg string) (*ErrorData, error) {
	var foundTemplate *ErrorTemplate

	for _, l := range SupportedLangs {
		for _, t := range l.ErrorTemplates {
			if t.StartPattern.MatchString(errorMsg) && t.BacktracePattern.MatchString(errorMsg) {
				foundTemplate = t
				break
			}
		}
	}

	if foundTemplate == nil {
		return nil, fmt.Errorf("specific template for this error message was not found")
	}

	// TODO: parse message

	// parse backtrace
	offendingFiles := []Location{}
	rawMatches := foundTemplate.BacktracePattern.FindAllStringSubmatch(errorMsg, -1)
	// backtraceNames := foundTemplate.BacktracePattern.SubexpNames()

	for _, m := range rawMatches {
		// get symbol
		// symbolIdx := foundTemplate.BacktracePattern.SubexpIndex(backtraceNames[0])
		// fileIdx := foundTemplate.BacktracePattern.SubexpIndex(backtraceNames[1])
		// posIdx := foundTemplate.BacktracePattern.SubexpIndex(backtraceNames[2])

		symbol := m[1]
		file := m[2]
		pos := SimplePosition{}

		posses := strings.Split(m[3], ":")
		pos.Line, _ = strconv.Atoi(posses[0])
		if len(posses) > 1 {
			pos.Column, _ = strconv.Atoi(posses[1])
		}

		offendingFiles = append(offendingFiles, Location{
			Symbol: symbol,
			File:   file,
			Pos:    pos,
		})
	}

	errData := &ErrorData{
		Content:         errorMsg,
		Locations:       offendingFiles,
		ErrorTemplate:   foundTemplate,
		ConnectedErrors: []Location{},
	}

	// TODO:
	return errData, nil
}
