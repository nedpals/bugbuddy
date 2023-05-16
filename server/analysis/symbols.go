package analysis

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
	Version int
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

type SimplePosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Index  int
}
