package error_analyzer

import (
	"bufio"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenKind int

const (
	ErrorKind   TokenKind = -1
	EofKind     TokenKind = 0
	TextKind    TokenKind = iota
	KeywordKind TokenKind = iota
	RegexKind   TokenKind = iota
)

type Token struct {
	Content  string
	Kind     TokenKind
	Position Position
}

type Position struct {
	FromIndex int
	ToIndex   int
}

type Tokenizer struct {
	scanner      *bufio.Scanner
	CurrentIndex int
}

func (t *Tokenizer) Tokenize(kind TokenKind, content string, start int, end int) Token {
	return Token{
		Content:  content,
		Kind:     kind,
		Position: Position{start, end},
	}
}

func (t *Tokenizer) Next() (string, int, int) {
	if !t.scanner.Scan() {
		return "", 0, 0
	}

	content := t.scanner.Text()
	origIndex := t.CurrentIndex
	t.CurrentIndex += len(content)
	return content, origIndex + 1, t.CurrentIndex
}

func (t *Tokenizer) Scan() Token {
	content, start, end := t.Next()
	if len(content) == 0 && start == end {
		return t.Tokenize(EofKind, content, start, end)
	}

	if len(content) == 2 {
		if content[0] == '{' && content[1] == '{' {
			content, start, end = t.Next()

			if content == "repeat" {
				content, start, end = t.Next()
			}

			// expect closing
			if exp, _, _ := t.Next(); len(exp) == 2 && exp[0] == '}' && exp[1] == '}' {
				return t.Tokenize(RegexKind, content, start, end)
			}

			// FIXME:
			return t.Tokenize(ErrorKind, "invalid", 0, 0)
		}
	}

	return t.Tokenize(TextKind, content, start, end)
}

func SplitTokenFunc(data []byte, atEOF bool) (int, []byte, error) {
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
	}
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if r == '\\' && i+1 < len(data) {
			r1, width1 := utf8.DecodeRune(data[i+1:])
			if r1 == ' ' {
				i += width1
				continue
			}
		}
		if unicode.IsSpace(r) {
			return i + width, data[start:i], nil
		}
	}
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	return start, nil, nil

}

func NewTokenizer(input string) *Tokenizer {
	sc := bufio.NewScanner(strings.NewReader(input))
	sc.Split(SplitTokenFunc)

	return &Tokenizer{
		scanner:      sc,
		CurrentIndex: -1,
	}
}

type Template struct {
	Name     string
	Language string
}

type TemplateLoader struct {
	Templates []Template
}

func (l *TemplateLoader) Load() error {
	// execPath, err := os.Executable()
	// if err != nil {
	// 	return err
	// }

	// execDir := filepath.Dir(execPath)
	// templateDir := filepath.Join(execDir, "..", "error_analyzer", "error_templates")
	// level := 0
	// err := filepath.Walk(templateDir, func(path string, info fs.FileInfo, err error) error {
	// 	// if level == 0 &&
	// })
	return nil
}
