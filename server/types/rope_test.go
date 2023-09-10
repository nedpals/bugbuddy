package types

import (
	"testing"
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

	t.Run("Invalid Insert", func(t *testing.T) {
		r := NewRope("Hello, World!")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Insert did not panic on invalid position")
			}
		}()

		r.Insert(15, "Invalid Insert")
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
