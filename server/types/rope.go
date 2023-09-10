package types

import (
	"strings"

	lsp "go.lsp.dev/protocol"
)

// Rope represents a text data structure.
type Rope struct {
	left  *Rope
	right *Rope
	text  string
}

// NewRope creates a new rope with the given text.
func NewRope(text string) *Rope {
	return &Rope{text: text}
}

// Insert inserts text at the specified position in the rope.
func (r *Rope) Insert(position int, text string) {
	if position < 0 || position > len(r.text) {
		panic("Invalid position")
	}

	if len(text) == 0 {
		return
	}

	if position == len(r.text) {
		r.text += text
		return
	}

	if r.left == nil {
		r.left = NewRope(r.text[:position])
		r.right = NewRope(r.text[position:])
	}

	if position == 0 {
		r.left.Insert(len(r.left.text), text)
	} else if position == len(r.left.text) {
		r.right.Insert(0, text)
	} else {
		r.left.Insert(position, text)
	}
}

// Delete deletes text from the specified position in the rope.
func (r *Rope) Delete(position, length int) {
	endPosition := position + length
	if position < 0 || length <= 0 {
		panic("Invalid position or length")
	}

	if r.left == nil && r.right == nil {
		if position >= len(r.text) || endPosition > len(r.text) {
			panic("Invalid position or length")
		}
		r.text = r.text[:position] + r.text[endPosition:]
		return
	}

	if position == len(r.left.text) {
		r.right.Delete(0, length)
	} else if endPosition == len(r.left.text) {
		r.left.Delete(position, length)
	} else if position < len(r.left.text) {
		if endPosition <= len(r.left.text) {
			r.left.Delete(position, length)
		} else {
			leftLength := len(r.left.text)
			r.left.Delete(position, leftLength-position)
			r.right.Delete(0, length-(leftLength-position))
		}
	}
}

// ToString returns the string representation of the rope.
func (r *Rope) ToString() string {
	if r.left == nil && r.right == nil {
		return r.text
	}

	return r.left.ToString() + r.right.ToString()
}

// OffsetFromPosition converts an LSP Position to a byte offset.
func (r *Rope) OffsetFromPosition(position lsp.Position) int {
	// Split the text into lines
	lines := strings.Split(r.ToString(), "\n")

	// Ensure line is within valid range
	lineCount := len(lines)
	if position.Line >= uint32(lineCount) {
		position.Line = uint32(lineCount - 1)
	}

	// Get the line text
	line := lines[position.Line]

	// Ensure character is within valid range
	if position.Character >= uint32(len(line)) {
		position.Character = uint32(len(line) - 1)
	}

	// Calculate byte offset
	offset := 0
	for i := 0; i < int(position.Line); i++ {
		offset += len(lines[i]) + 1 // Add 1 for the newline character
	}
	offset += int(position.Character)

	return offset
}
