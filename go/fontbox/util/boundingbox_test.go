package util

import "testing"

// Written from org.apache.fontbox.util.BoundingBox; the Java suite has no test
// for it, so per migration/conventions/tdd.md these come from the source.

func TestBoundingBox(t *testing.T) {
	b := NewBoundingBoxOf(1, 2, 4, 6)

	if b.LowerLeftX() != 1 || b.LowerLeftY() != 2 || b.UpperRightX() != 4 || b.UpperRightY() != 6 {
		t.Fatalf("box = %v, want [1.0,2.0,4.0,6.0]", b)
	}
	if got := b.Width(); got != 3 {
		t.Errorf("Width = %v, want 3", got)
	}
	if got := b.Height(); got != 4 {
		t.Errorf("Height = %v, want 4", got)
	}
}

func TestBoundingBoxDefault(t *testing.T) {
	b := NewBoundingBox()
	if b.Width() != 0 || b.Height() != 0 {
		t.Errorf("a new box is %v, want the empty one", b)
	}
}

func TestBoundingBoxFromNumbers(t *testing.T) {
	b := NewBoundingBoxOfNumbers([]float32{1, 2, 4, 6})
	if b.UpperRightY() != 6 {
		t.Errorf("box = %v, want [1.0,2.0,4.0,6.0]", b)
	}
}

func TestBoundingBoxSetters(t *testing.T) {
	b := NewBoundingBox()
	b.SetLowerLeftX(1)
	b.SetLowerLeftY(2)
	b.SetUpperRightX(4)
	b.SetUpperRightY(6)
	if b.Width() != 3 || b.Height() != 4 {
		t.Errorf("box = %v, want [1.0,2.0,4.0,6.0]", b)
	}
}

// TestBoundingBoxContains pins that the edges count as inside.
func TestBoundingBoxContains(t *testing.T) {
	b := NewBoundingBoxOf(0, 0, 10, 10)
	cases := []struct {
		x, y float32
		want bool
	}{
		{5, 5, true},
		{0, 0, true},
		{10, 10, true},
		{10.1, 5, false},
		{-0.1, 5, false},
	}
	for _, c := range cases {
		if got := b.Contains(c.x, c.y); got != c.want {
			t.Errorf("Contains(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestBoundingBoxString(t *testing.T) {
	if got, want := NewBoundingBoxOf(1, 2, 4, 6).String(), "[1.0,2.0,4.0,6.0]"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
