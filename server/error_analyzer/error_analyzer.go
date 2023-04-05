package error_analyzer

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func setupParsers() {
	// TODO: just added code to avoid go-tree-sitter to be removed
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
}

var Default = &ErrorAnalyzer{}

type ErrorAnalyzer struct{}

func (an *ErrorAnalyzer) Analyze(errorMsg string) string {
	return "test"
}
