package store

import (
	"context"
	"strings"

	"github.com/nedpals/bugbuddy-proto/server/analysis"
	sitter "github.com/smacker/go-tree-sitter"
)

type Document struct {
	Filepath string
	Content  []byte
	Resolved bool
	Language *analysis.Language
	Tree     *sitter.Tree
}

func (doc *Document) ParseTree() error {
	// TODO: reuse parser
	p := sitter.NewParser()
	p.SetLanguage(doc.Language.SitterLanguage)
	newTree, err := p.ParseCtx(context.Background(), doc.Tree, doc.Content)
	if err != nil {
		return err
	}
	doc.Tree = newTree
	return nil
}

type SymbolStore []*analysis.Symbol

func (store SymbolStore) Add(doc Document, node sitter.Node) {
	store = append(store, &analysis.Symbol{
		Version: 1,
		Name:    node.Content(doc.Content),
		Kind:    doc.Language.NodeKind(node),
		Pos: analysis.Position{
			StartLine:   int(node.StartPoint().Row),
			StartColumn: int(node.StartPoint().Column),
			EndLine:     int(node.EndPoint().Row),
			EndColumn:   int(node.EndPoint().Column),
			StartIndex:  int(node.StartByte()),
			EndIndex:    int(node.EndByte()),
		},
	})
}

type Store struct {
	// a map of file paths mapped to document contents
	Documents map[string]*Document
	Symbols   SymbolStore
	Errors    []*analysis.ErrorData
}

func NewStore() *Store {
	return &Store{
		Documents: map[string]*Document{},
		Symbols:   SymbolStore{},
		Errors:    []*analysis.ErrorData{},
	}
}

func (st *Store) InsertDocument(path string, content string) {
	detectedLang := analysis.SupportedLangs.DetectByPath(path)

	st.Documents[path] = &Document{
		Filepath: path,
		Content:  []byte(content),
		Language: detectedLang,
		Resolved: len(content) != 0,
	}
}

func (st *Store) ResolveDocument(fullPath string, content string) {
	foundPath := ""

	for gotPath := range st.Documents {
		if strings.HasSuffix(fullPath, gotPath) {
			foundPath = gotPath
			break
		}
	}

	if len(foundPath) != 0 {
		st.Documents[fullPath] = st.Documents[foundPath]
		st.Documents[fullPath].Content = []byte(content)
		delete(st.Documents, foundPath)
	} else {
		st.InsertDocument(fullPath, content)
	}

	st.Documents[fullPath].Resolved = true
	st.Documents[fullPath].ParseTree()
}
