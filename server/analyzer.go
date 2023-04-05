package main

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func setupParsers() {
	// TODO: just added code to avoid go-tree-sitter to be removed
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
}

var defaultErrorAnalyzer = &ErrorAnalyzer{}

type ErrorAnalyzer struct{}

func (an *ErrorAnalyzer) analyze(errorMsg string) string {
	return "test"
}
