package store

import (
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

// TODO:
type Language int

const (
	UnknownLanguage    Language = 0
	JavaLanguage       Language = iota
	PythonLanguage     Language = iota
	JavaScriptLanguage Language = iota
)

var formatToLanguageMaps = map[string]Language{
	".java": JavaLanguage,
	".py":   PythonLanguage,
	".js":   JavaScriptLanguage,
	".mjs":  JavaScriptLanguage,
	".cjs":  JavaScriptLanguage,
}

type Document struct {
	content  string
	filepath string
	language Language
}

type Store struct {
	// a map of file paths mapped to document contents
	Documents map[string]Document
	Symbols   SymbolStore
}

func (st *Store) InsertDocument(path string, content string) {
	detectedLang := UnknownLanguage
	if gotLang, ok := formatToLanguageMaps[filepath.Ext(path)]; ok {
		detectedLang = gotLang
	}

	st.Documents[path] = Document{
		filepath: path,
		content:  content,
		language: detectedLang,
	}
}

type SymbolKind int

const (
	VariableSym SymbolKind = 0
	FunctionSym SymbolKind = iota
	ClassSym    SymbolKind = iota
	StructSym   SymbolKind = iota
	TypedefSym  SymbolKind = iota
)

type Symbol struct {
	Name string     `json:"name"`
	Kind SymbolKind `json:"kind"`
	Pos  Position   `json:"pos"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type SymbolStore []Store

func (store *SymbolStore) AddFromSitter(node sitter.Node) {

}
