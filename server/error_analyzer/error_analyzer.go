package error_analyzer

import (
	"fmt"

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
	tokenizer := NewTokenizer(errorMsg)

	for tok := tokenizer.Scan(); tok.Kind != EofKind && tok.Kind != ErrorKind; tok = tokenizer.Scan() {
		fmt.Println(tok)
	}

	return ""
}
