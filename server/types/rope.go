package types

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
	} else if position == len(r.left.text) {
		r.right.Insert(0, text)
	} else if position == 0 {
		r.left.Insert(len(r.left.text), text)
	} else {
		r.left.Insert(position, text)
	}
}

// Delete deletes text from the specified position in the rope.
func (r *Rope) Delete(position, length int) {
	if position < 0 || position >= len(r.text) || length <= 0 || position+length > len(r.text) {
		panic("Invalid position or length")
	}

	if r.left == nil && r.right == nil {
		r.text = r.text[:position] + r.text[position+length:]
		return
	}

	if position == len(r.left.text) {
		r.right.Delete(0, length)
	} else if position+length == len(r.left.text) {
		r.left.Delete(position, length)
	} else if position < len(r.left.text) {
		if position+length <= len(r.left.text) {
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
