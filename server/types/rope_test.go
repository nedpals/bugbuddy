package types

import (
	"fmt"
	"testing"

	lsp "go.lsp.dev/protocol"
)

func TestRope(t *testing.T) {
	t.Run("Insert and ToString", func(t *testing.T) {
		r := NewRope("Hello, World!")
		r.Insert(7, "Awesome ")

		expected := "Hello, Awesome World!"
		result := r.ToString()

		if result != expected {
			t.Errorf("Expected: %s, Got: %s", expected, result)
		}
	})

	t.Run("Delete and ToString", func(t *testing.T) {
		r := NewRope("Hello, Awesome World!")
		r.Delete(6, 8)

		expected := "Hello, World!"
		result := r.ToString()

		if result != expected {
			t.Errorf("Expected: %s, Got: %s", expected, result)
		}
	})

	t.Run("Insert Delete and ToString", func(t *testing.T) {
		r := NewRope("Hello, World!")
		r.Insert(7, "Awesome ")

		fmt.Println(r.ToString())
		r.Delete(6, 8)

		expected := "Hello, World!"
		result := r.ToString()

		if result != expected {
			t.Errorf("Expected: %s, Got: %s", expected, result)
		}
	})

	t.Run("ToString Nil Rope", func(t *testing.T) {
		var r *Rope = nil

		expected := ""
		result := r.ToString()

		if result != expected {
			t.Errorf("Expected: %s, Got: %s", expected, result)
		}
	})

	t.Run("Invalid Insert", func(t *testing.T) {
		r := NewRope("Hello, World!")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Insert did not panic on invalid position")
			}
		}()

		r.Insert(15, "Invalid Insert")
	})

	t.Run("Insert on Nil Rope", func(t *testing.T) {
		var r *Rope = nil

		r.Insert(0, "Invalid Insert")
	})

	t.Run("Invalid Delete", func(t *testing.T) {
		r := NewRope("Hello, World!")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Delete did not panic on invalid position or length")
			}
		}()

		r.Delete(15, 10)
	})
}

func TestOffsetFromPosition(t *testing.T) {
	t.Run("Valid Position", func(t *testing.T) {
		r := NewRope("Hello, World!\nThis is a test.")
		position := lsp.Position{Line: 1, Character: 6} // Line 1, Character 6

		offset := r.OffsetFromPosition(position)

		expected := 20 // 18 bytes into the text
		if offset != expected {
			t.Errorf("Expected offset: %d, Got offset: %d", expected, offset)
		}

		if r.text[:offset] != "Hello, World!\nThis i" {
			t.Errorf("Expected text: %s, Got text: %s", "Hello, World!\nThis ", r.text[:offset])
		}
	})

	t.Run("Position Exceeds Line Length", func(t *testing.T) {
		r := NewRope("Hello, World!\nThis is a test.")
		position := lsp.Position{Line: 1, Character: 20} // Line 1, Character exceeds line length

		offset := r.OffsetFromPosition(position)

		expected := 29 // Maximum offset is 29 (end of text)
		if offset != expected {
			t.Errorf("Expected offset: %d, Got offset: %d", expected, offset)
		}

		if r.text[:offset] != "Hello, World!\nThis is a test." {
			t.Errorf("Expected text: %s, Got text: %s", "Hello, World!\nThis is a test.", r.text[:offset])
		}
	})

	t.Run("Position Exceeds Line Count", func(t *testing.T) {
		r := NewRope("Hello, World!\nThis is a test.")
		position := lsp.Position{Line: 3, Character: 6} // Line exceeds line count

		offset := r.OffsetFromPosition(position)

		expected := 20 // Maximum offset is 29 (end of text)
		if offset != expected {
			t.Errorf("Expected offset: %d, Got offset: %d", expected, offset)
		}

		if r.text[:offset] != "Hello, World!\nThis i" {
			t.Errorf("Expected text: %s, Got text: %s", "Hello, World!\nThis i", r.text[:offset])
		}
	})
}
