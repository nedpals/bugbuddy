package analysis

import (
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
)

var UnknownLanguage = &Language{
	Id: "unknown",
	NodeKind: func(n sitter.Node) SymbolKind {
		return UnknownSym
	},
	FileExtensions: []string{},
}

type Language struct {
	Id             string
	SitterLanguage *sitter.Language
	NodeKind       func(n sitter.Node) SymbolKind
	FileExtensions []string
	ErrorTemplates []*ErrorTemplate
}

type LanguageList []*Language

func (langs LanguageList) DetectByPath(path string) *Language {
	gotExt := filepath.Ext(path)

	for _, lang := range langs {
		for _, ext := range lang.FileExtensions {
			if gotExt == ext {
				return lang
			}
		}
	}

	return UnknownLanguage
}

var JavaLanguage = &Language{
	Id:             "java",
	SitterLanguage: java.GetLanguage(),
	NodeKind: func(n sitter.Node) SymbolKind {
		// TODO:
		return UnknownSym
	},
	FileExtensions: []string{".java"},
	ErrorTemplates: []*ErrorTemplate{
		NewErrorTemplate(ErrorTemplateConfig{
			Name:             "NullPointerException",
			Description:      "One of your variables have null values",
			StartPattern:     `Exception in thread "main" java.lang.NullPointerException`,
			BacktracePattern: `\n\s+at (?P<symbol>[a-zA-Z0-9_.]+)\((?P<file>\S+.java):(?P<pos>[0-9:]+)\)`,
		}),
		NewErrorTemplate(ErrorTemplateConfig{
			Name:             "ArrayIndexOutOfBoundsException",
			Description:      "You are accessing an element of the array outside the array's length.",
			StartPattern:     `Exception in thread "main" java.lang.ArrayIndexOutOfBoundsException: Index \d+ out of bounds for length \d+`,
			BacktracePattern: `\n\s+at (?P<symbol>[a-zA-Z0-9_.]+)\((?P<file>\S+.java):(?P<pos>[0-9:]+)\)`,
		}),
	},
}

var JavascriptLanguage = &Language{
	Id:             "javascript",
	SitterLanguage: javascript.GetLanguage(),
	NodeKind: func(n sitter.Node) SymbolKind {
		// TODO:
		return UnknownSym
	},
	FileExtensions: []string{".js", ".mjs", ".cjs"},
}

var PythonLanguage = &Language{
	Id:             "python",
	SitterLanguage: python.GetLanguage(),
	NodeKind: func(n sitter.Node) SymbolKind {
		// TODO:
		return UnknownSym
	},
	FileExtensions: []string{".py"},
}

var SupportedLangs = LanguageList{
	JavaLanguage,
	PythonLanguage,
	JavascriptLanguage,
}
