package cmap

// JAVA-BUGS 16: `CMap.useCmap` builds its one-byte reverse code with `% 0xFF`
// where the two branches beside it use `& 0xFF`. The test is in-package
// because `useCmap` and `addCharMapping` are, as they are in the Java.

import (
	"bytes"
	"testing"
)

// TestUseCmapKeepsTheCodeFF is the one code the modulus gets wrong.
//
// `k % 0xFF` and `k & 0xFF` agree for every byte but 255: 255 % 255 is 0, and
// 255 & 255 is 255. So a parent CMap that maps the code 0xFF loses it, and the
// reverse map answers the code 0 for that character.
//
// The expected value is the arithmetic: the code the parent held is 0xFF.
func TestUseCmapKeepsTheCodeFF(t *testing.T) {
	parent := newCMap()
	parent.addCharMapping([]byte{0xFF}, "A")

	child := newCMap()
	child.useCmap(parent)

	got, found := child.GetCodesFromUnicode("A")
	if !found {
		t.Fatal("the child has no code for A")
	}
	if !bytes.Equal(got, []byte{0xFF}) {
		t.Errorf("the code for A is % x, want ff: %% 0xFF turns 255 into 0", got)
	}
}
