package store

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// TODO:
type SemanticModel struct {
	Parent   *SemanticModel
	Symbols  SymbolStore
	Pos      Position
	Children []*SemanticModel
}

type Document struct {
	content  []byte
	filepath string
	language *Language
	tree     *sitter.Tree
	model    *SemanticModel
}

type Store struct {
	// a map of file paths mapped to document contents
	Documents map[string]Document
	Symbols   SymbolStore
}

func (st *Store) InsertDocument(path string, content string) {
	detectedLang := supportedLanguages.DetectByPath(path)

	st.Documents[path] = Document{
		filepath: path,
		content:  []byte(content),
		language: detectedLang,
	}
}

type SymbolKind int

const (
	UnknownSym  SymbolKind = 0
	VariableSym SymbolKind = iota
	FunctionSym SymbolKind = iota
	ClassSym    SymbolKind = iota
	StructSym   SymbolKind = iota
	TypedefSym  SymbolKind = iota
)

type Symbol struct {
	version int
	Name    string     `json:"name"`
	Kind    SymbolKind `json:"kind"`
	Pos     Position   `json:"pos"`
}

type Position struct {
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
	StartIndex  int
	EndIndex    int
}

type SymbolStore []*Symbol

func (store SymbolStore) Add(doc Document, node sitter.Node) {
	store = append(store, &Symbol{
		version: 1,
		Name:    node.Content(doc.content),
		Kind:    doc.language.NodeKind(node),
		Pos: Position{
			StartLine:   int(node.StartPoint().Row),
			StartColumn: int(node.StartPoint().Column),
			EndLine:     int(node.EndPoint().Row),
			EndColumn:   int(node.EndPoint().Column),
			StartIndex:  int(node.StartByte()),
			EndIndex:    int(node.EndByte()),
		},
	})
}
