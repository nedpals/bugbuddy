package store

import (
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

type Language struct {
	Id             string
	NodeKind       func(n sitter.Node) SymbolKind
	FileExtensions []string
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

var (
	UnknownLanguage = &Language{
		Id: "unknown",
		NodeKind: func(n sitter.Node) SymbolKind {
			return UnknownSym
		},
		FileExtensions: []string{},
	}
	supportedLanguages = LanguageList{
		&Language{
			Id: "java",
			NodeKind: func(n sitter.Node) SymbolKind {
				// TODO:
				return UnknownSym
			},
			FileExtensions: []string{".java"},
		},
		&Language{
			Id: "python",
			NodeKind: func(n sitter.Node) SymbolKind {
				// TODO:
				return UnknownSym
			},
			FileExtensions: []string{".py"},
		},
		&Language{
			Id: "javascript",
			NodeKind: func(n sitter.Node) SymbolKind {
				// TODO:
				return UnknownSym
			},
			FileExtensions: []string{".js", ".mjs", ".cjs"},
		},
	}
)
